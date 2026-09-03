package repository

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
)

func TestTransactionPublishesVerifiedFastForward(t *testing.T) {
	git, root, repo := makeGitRepo(t)
	ctx := context.Background()
	before, err := repo.State(ctx)
	if err != nil {
		t.Fatal(err)
	}

	txn, err := repo.BeginTransaction(ctx, before.Revision)
	if err != nil {
		t.Fatal(err)
	}
	defer txn.Close()
	if err := os.WriteFile(filepath.Join(txn.Root(), "README.md"), []byte("transaction\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	message, err := FormatMutationCommitMessage("memory: test transaction", AppliedOperation{
		IdempotencyKey:   "test-op-1",
		RequestSHA256:    "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Operation:        "update",
		PrimaryMemoryID:  "11111111-1111-4111-8111-111111111111",
		ChangedMemoryIDs: []string{"11111111-1111-4111-8111-111111111111"},
	})
	if err != nil {
		t.Fatal(err)
	}
	commit, err := txn.Commit(ctx, message)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "initial\n" {
		t.Fatalf("canonical worktree changed before publish: %q", data)
	}
	if err := repo.Publish(ctx, "main", before.Revision, commit); err != nil {
		t.Fatal(err)
	}
	data, err = os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "transaction\n" {
		t.Fatalf("published README = %q", data)
	}
	after, err := repo.State(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if after.Revision != commit || !after.Clean {
		t.Fatalf("published state = %+v, commit = %s", after, commit)
	}

	applied, found, err := repo.FindAppliedOperation(ctx, "test-op-1")
	if err != nil {
		t.Fatal(err)
	}
	if !found || applied.Commit != commit || applied.Operation != "update" || applied.RequestSHA256 == "" {
		t.Fatalf("applied operation = %+v, found=%v", applied, found)
	}
	if !reflect.DeepEqual(applied.ChangedMemoryIDs, []string{"11111111-1111-4111-8111-111111111111"}) {
		t.Fatalf("changed IDs = %v", applied.ChangedMemoryIDs)
	}
	_ = git
}

func TestPublishRejectsStaleRevisionWithoutOverwritingCompetitor(t *testing.T) {
	git, root, repo := makeGitRepo(t)
	ctx := context.Background()
	before, err := repo.State(ctx)
	if err != nil {
		t.Fatal(err)
	}
	txn, err := repo.BeginTransaction(ctx, before.Revision)
	if err != nil {
		t.Fatal(err)
	}
	defer txn.Close()
	if err := os.WriteFile(filepath.Join(txn.Root(), "README.md"), []byte("transaction\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	commit, err := txn.Commit(ctx, "transaction")
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("competitor\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, git, root, "add", "README.md")
	runGit(t, git, root, "commit", "-m", "competitor")
	competitor, err := repo.State(ctx)
	if err != nil {
		t.Fatal(err)
	}

	err = repo.Publish(ctx, "main", before.Revision, commit)
	var stale *StaleRevisionError
	if !errors.As(err, &stale) {
		t.Fatalf("publish error = %v, want StaleRevisionError", err)
	}
	if stale.Current != competitor.Revision {
		t.Fatalf("stale current = %s, want %s", stale.Current, competitor.Revision)
	}
	data, err := os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "competitor\n" {
		t.Fatalf("canonical README overwritten: %q", data)
	}
}

func TestFormatMutationCommitMessageSortsChangedIDs(t *testing.T) {
	message, err := FormatMutationCommitMessage("memory: create", AppliedOperation{
		IdempotencyKey: "abc-123",
		RequestSHA256:  "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		Operation:      "create",
		ChangedMemoryIDs: []string{
			"22222222-2222-4222-8222-222222222222",
			"11111111-1111-4111-8111-111111111111",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	parsed, ok := parseAppliedOperation("deadbeef", message)
	if !ok {
		t.Fatal("formatted mutation message was not parseable")
	}
	want := []string{
		"11111111-1111-4111-8111-111111111111",
		"22222222-2222-4222-8222-222222222222",
	}
	if !reflect.DeepEqual(parsed.ChangedMemoryIDs, want) {
		t.Fatalf("changed IDs = %v, want %v", parsed.ChangedMemoryIDs, want)
	}
}

func makeGitRepo(t *testing.T) (string, string, *Git) {
	t.Helper()
	git, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git is not available")
	}
	root := t.TempDir()
	runGit(t, git, root, "init", "-b", "main")
	runGit(t, git, root, "config", "user.name", "Runethread Test")
	runGit(t, git, root, "config", "user.email", "runethread-test@example.invalid")
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("initial\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, git, root, "add", "README.md")
	runGit(t, git, root, "commit", "-m", "initial")
	repo, err := OpenGit(root)
	if err != nil {
		t.Fatal(err)
	}
	return git, root, repo
}
