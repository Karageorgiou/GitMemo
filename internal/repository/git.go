package repository

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

type State struct {
	Revision     string   `json:"revision"`
	Branch       string   `json:"branch,omitempty"`
	Clean        bool     `json:"clean"`
	DirtyEntries []string `json:"dirty_entries,omitempty"`
}

type Reader interface {
	Root() string
	State(context.Context) (State, error)
}

type Git struct {
	root string
	git  string
}

func OpenGit(root string) (*Git, error) {
	git, err := exec.LookPath("git")
	if err != nil {
		return nil, fmt.Errorf("find git executable: %w", err)
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve repository root: %w", err)
	}
	cmd := exec.Command(git, "-C", abs, "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("resolve Git worktree root: %w", commandError(err))
	}
	top := strings.TrimSpace(string(out))
	if top == "" {
		return nil, errors.New("resolve Git worktree root: git returned an empty path")
	}
	return &Git{root: filepath.Clean(top), git: git}, nil
}

func (g *Git) Root() string { return g.root }

func (g *Git) State(ctx context.Context) (State, error) {
	revision, err := g.output(ctx, "rev-parse", "--verify", "HEAD")
	if err != nil {
		return State{}, fmt.Errorf("read HEAD revision: %w", err)
	}
	branch, err := g.output(ctx, "symbolic-ref", "--quiet", "--short", "HEAD")
	if err != nil {
		var commandErr *CommandError
		if !errors.As(err, &commandErr) || commandErr.ExitCode != 1 {
			return State{}, fmt.Errorf("read current branch: %w", err)
		}
		branch = ""
	}
	status, err := g.output(ctx, "status", "--porcelain=v1", "--untracked-files=all")
	if err != nil {
		return State{}, fmt.Errorf("read working tree status: %w", err)
	}
	entries := parsePorcelain(status)
	return State{
		Revision:     strings.TrimSpace(revision),
		Branch:       strings.TrimSpace(branch),
		Clean:        len(entries) == 0,
		DirtyEntries: entries,
	}, nil
}

type CommandError struct {
	ExitCode int
	Stderr   string
}

func (e *CommandError) Error() string {
	if e.Stderr == "" {
		return fmt.Sprintf("git exited with code %d", e.ExitCode)
	}
	return fmt.Sprintf("git exited with code %d: %s", e.ExitCode, e.Stderr)
}

func (g *Git) output(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, g.git, append([]string{"-C", g.root}, args...)...)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return "", &CommandError{ExitCode: exitErr.ExitCode(), Stderr: strings.TrimSpace(stderr.String())}
		}
		return "", err
	}
	return string(out), nil
}

func commandError(err error) error {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return &CommandError{ExitCode: exitErr.ExitCode(), Stderr: strings.TrimSpace(string(exitErr.Stderr))}
	}
	return err
}

func parsePorcelain(text string) []string {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}
