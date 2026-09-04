package repository

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const mutationMarker = "v1"

var (
	operationKeyRE = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)
	sha256RE       = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

type AppliedOperation struct {
	Commit           string   `json:"commit"`
	IdempotencyKey   string   `json:"idempotency_key"`
	RequestSHA256    string   `json:"request_sha256"`
	Operation        string   `json:"operation"`
	PrimaryMemoryID  string   `json:"primary_memory_id,omitempty"`
	TargetMemoryID   string   `json:"target_memory_id,omitempty"`
	ChangedMemoryIDs []string `json:"changed_memory_ids,omitempty"`
}

type Transaction interface {
	Root() string
	Commit(context.Context, string) (string, error)
	Close() error
}

type Writer interface {
	Reader
	BeginTransaction(context.Context, string) (Transaction, error)
	Publish(context.Context, string, string, string) error
	FindAppliedOperation(context.Context, string) (AppliedOperation, bool, error)
}

type StaleRevisionError struct {
	Expected string
	Current  string
}

func (e *StaleRevisionError) Error() string {
	return fmt.Sprintf("repository revision changed: expected %s, current %s", e.Expected, e.Current)
}

type DirtyRepositoryError struct {
	Entries []string
}

func (e *DirtyRepositoryError) Error() string {
	return "repository working tree is dirty"
}

type gitTransaction struct {
	git      string
	mainRoot string
	baseDir  string
	root     string
	closed   bool
}

func (g *Git) BeginTransaction(ctx context.Context, revision string) (Transaction, error) {
	revision = strings.TrimSpace(revision)
	if revision == "" {
		return nil, errors.New("transaction revision must not be empty")
	}
	if _, err := g.output(ctx, "rev-parse", "--verify", revision+"^{commit}"); err != nil {
		return nil, fmt.Errorf("resolve transaction revision %s: %w", revision, err)
	}
	base, err := os.MkdirTemp("", "runethread-transaction-*")
	if err != nil {
		return nil, fmt.Errorf("create transaction directory: %w", err)
	}
	root := filepath.Join(base, "worktree")
	if err := g.run(ctx, "-c", "core.autocrlf=false", "worktree", "add", "--detach", root, revision); err != nil {
		_ = os.RemoveAll(base)
		return nil, fmt.Errorf("create transaction worktree: %w", err)
	}
	return &gitTransaction{git: g.git, mainRoot: g.root, baseDir: base, root: root}, nil
}

func (t *gitTransaction) Root() string { return t.root }

func (t *gitTransaction) Commit(ctx context.Context, message string) (string, error) {
	if t.closed {
		return "", errors.New("transaction is closed")
	}
	if strings.TrimSpace(message) == "" {
		return "", errors.New("commit message must not be empty")
	}
	if err := t.run(ctx, "add", "-A"); err != nil {
		return "", fmt.Errorf("stage transaction changes: %w", err)
	}
	if err := t.run(ctx, "diff", "--cached", "--quiet"); err == nil {
		return "", errors.New("transaction has no staged changes")
	} else {
		var commandErr *CommandError
		if !errors.As(err, &commandErr) || commandErr.ExitCode != 1 {
			return "", fmt.Errorf("inspect staged transaction changes: %w", err)
		}
	}
	if err := t.run(ctx,
		"-c", "user.name=Runethread",
		"-c", "user.email=runethread@localhost",
		"commit", "--no-gpg-sign", "-m", message,
	); err != nil {
		return "", fmt.Errorf("commit transaction: %w", err)
	}
	out, err := t.output(ctx, "rev-parse", "--verify", "HEAD")
	if err != nil {
		return "", fmt.Errorf("read transaction commit: %w", err)
	}
	return strings.TrimSpace(out), nil
}

func (t *gitTransaction) Close() error {
	if t.closed {
		return nil
	}
	t.closed = true
	cmd := exec.Command(t.git, "-C", t.mainRoot, "worktree", "remove", "--force", t.root)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	removeErr := cmd.Run()
	fsErr := os.RemoveAll(t.baseDir)
	if removeErr != nil {
		return fmt.Errorf("remove transaction worktree: %w", commandErrorWithStderr(removeErr, stderr.String()))
	}
	if fsErr != nil {
		return fmt.Errorf("remove transaction directory: %w", fsErr)
	}
	return nil
}

func (g *Git) Publish(ctx context.Context, branch, expectedRevision, commit string) error {
	branch = strings.TrimSpace(branch)
	expectedRevision = strings.TrimSpace(expectedRevision)
	commit = strings.TrimSpace(commit)
	if branch == "" {
		return errors.New("publish requires a named branch")
	}
	if expectedRevision == "" || commit == "" {
		return errors.New("publish requires expected and new revisions")
	}
	state, err := g.State(ctx)
	if err != nil {
		return err
	}
	if !state.Clean {
		return &DirtyRepositoryError{Entries: append([]string(nil), state.DirtyEntries...)}
	}
	if state.Branch != branch {
		return fmt.Errorf("current branch is %q, expected %q", state.Branch, branch)
	}
	if state.Revision != expectedRevision {
		return &StaleRevisionError{Expected: expectedRevision, Current: state.Revision}
	}
	if err := g.run(ctx, "-c", "core.autocrlf=false", "merge", "--ff-only", "--no-edit", commit); err != nil {
		latest, stateErr := g.State(ctx)
		if stateErr == nil && latest.Revision != expectedRevision {
			return &StaleRevisionError{Expected: expectedRevision, Current: latest.Revision}
		}
		return fmt.Errorf("fast-forward transaction commit: %w", err)
	}
	latest, err := g.State(ctx)
	if err != nil {
		return fmt.Errorf("verify published transaction: %w", err)
	}
	if latest.Revision != commit {
		return fmt.Errorf("published revision is %s, expected %s", latest.Revision, commit)
	}
	if !latest.Clean {
		return &DirtyRepositoryError{Entries: append([]string(nil), latest.DirtyEntries...)}
	}
	return nil
}

func (g *Git) FindAppliedOperation(ctx context.Context, key string) (AppliedOperation, bool, error) {
	key = strings.TrimSpace(key)
	if !operationKeyRE.MatchString(key) {
		return AppliedOperation{}, false, fmt.Errorf("invalid idempotency key %q", key)
	}
	pattern := "Runethread-Operation-ID: " + key
	out, err := g.output(ctx, "log", "HEAD", "--format=%H%x1f%B%x1e", "--fixed-strings", "--grep="+pattern)
	if err != nil {
		return AppliedOperation{}, false, fmt.Errorf("search mutation history: %w", err)
	}
	var matches []AppliedOperation
	for _, record := range strings.Split(out, "\x1e") {
		record = strings.TrimSpace(record)
		if record == "" {
			continue
		}
		parts := strings.SplitN(record, "\x1f", 2)
		if len(parts) != 2 {
			continue
		}
		operation, ok := parseAppliedOperation(strings.TrimSpace(parts[0]), parts[1])
		if ok && operation.IdempotencyKey == key {
			matches = append(matches, operation)
		}
	}
	if len(matches) == 0 {
		return AppliedOperation{}, false, nil
	}
	if len(matches) > 1 {
		return AppliedOperation{}, false, fmt.Errorf("idempotency key %q appears in multiple Runethread mutation commits", key)
	}
	return matches[0], true, nil
}

func FormatMutationCommitMessage(subject string, operation AppliedOperation) (string, error) {
	subject = strings.TrimSpace(subject)
	if subject == "" {
		return "", errors.New("mutation commit subject must not be empty")
	}
	if !operationKeyRE.MatchString(operation.IdempotencyKey) {
		return "", fmt.Errorf("invalid idempotency key %q", operation.IdempotencyKey)
	}
	if !sha256RE.MatchString(operation.RequestSHA256) {
		return "", fmt.Errorf("invalid request SHA-256 %q", operation.RequestSHA256)
	}
	changed := append([]string(nil), operation.ChangedMemoryIDs...)
	sort.Strings(changed)
	lines := []string{
		subject,
		"",
		"Runethread-Mutation: " + mutationMarker,
		"Runethread-Operation-ID: " + operation.IdempotencyKey,
		"Runethread-Request-SHA256: " + operation.RequestSHA256,
		"Runethread-Operation: " + strings.TrimSpace(operation.Operation),
	}
	if operation.PrimaryMemoryID != "" {
		lines = append(lines, "Runethread-Primary-Memory-ID: "+operation.PrimaryMemoryID)
	}
	if operation.TargetMemoryID != "" {
		lines = append(lines, "Runethread-Target-Memory-ID: "+operation.TargetMemoryID)
	}
	if len(changed) > 0 {
		lines = append(lines, "Runethread-Changed-Memory-IDs: "+strings.Join(changed, ","))
	}
	return strings.Join(lines, "\n"), nil
}

func parseAppliedOperation(commit, message string) (AppliedOperation, bool) {
	trailers := map[string]string{}
	for _, line := range strings.Split(strings.ReplaceAll(message, "\r\n", "\n"), "\n") {
		for _, key := range []string{
			"Runethread-Mutation",
			"Runethread-Operation-ID",
			"Runethread-Request-SHA256",
			"Runethread-Operation",
			"Runethread-Primary-Memory-ID",
			"Runethread-Target-Memory-ID",
			"Runethread-Changed-Memory-IDs",
		} {
			prefix := key + ": "
			if strings.HasPrefix(line, prefix) {
				trailers[key] = strings.TrimSpace(strings.TrimPrefix(line, prefix))
			}
		}
	}
	if trailers["Runethread-Mutation"] != mutationMarker ||
		!operationKeyRE.MatchString(trailers["Runethread-Operation-ID"]) ||
		!sha256RE.MatchString(trailers["Runethread-Request-SHA256"]) ||
		trailers["Runethread-Operation"] == "" {
		return AppliedOperation{}, false
	}
	changed := splitCommaList(trailers["Runethread-Changed-Memory-IDs"])
	return AppliedOperation{
		Commit:           commit,
		IdempotencyKey:   trailers["Runethread-Operation-ID"],
		RequestSHA256:    trailers["Runethread-Request-SHA256"],
		Operation:        trailers["Runethread-Operation"],
		PrimaryMemoryID:  trailers["Runethread-Primary-Memory-ID"],
		TargetMemoryID:   trailers["Runethread-Target-Memory-ID"],
		ChangedMemoryIDs: changed,
	}, true
}

func splitCommaList(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func (g *Git) run(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, g.git, append([]string{"-C", g.root}, args...)...)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return commandErrorWithStderr(err, stderr.String())
	}
	return nil
}

func (t *gitTransaction) run(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, t.git, append([]string{"-C", t.root}, args...)...)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return commandErrorWithStderr(err, stderr.String())
	}
	return nil
}

func (t *gitTransaction) output(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, t.git, append([]string{"-C", t.root}, args...)...)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", commandErrorWithStderr(err, stderr.String())
	}
	return string(out), nil
}

func commandErrorWithStderr(err error, stderr string) error {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return &CommandError{ExitCode: exitErr.ExitCode(), Stderr: strings.TrimSpace(stderr)}
	}
	return err
}
