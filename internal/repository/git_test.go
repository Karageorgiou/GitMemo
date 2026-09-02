package repository

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestGitStateTracksRevisionAndDirtiness(t *testing.T) {
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
	state, err := repo.State(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if state.Revision == "" {
		t.Fatal("expected non-empty revision")
	}
	if state.Branch != "main" {
		t.Fatalf("branch = %q, want main", state.Branch)
	}
	if !state.Clean || len(state.DirtyEntries) != 0 {
		t.Fatalf("expected clean repository, got %+v", state)
	}

	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	state, err = repo.State(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if state.Clean {
		t.Fatal("expected modified repository to be dirty")
	}
	if len(state.DirtyEntries) != 1 {
		t.Fatalf("dirty entries = %v, want one entry", state.DirtyEntries)
	}
}

func TestParsePorcelain(t *testing.T) {
	got := parsePorcelain(" M README.md\n?? new.txt\n")
	if len(got) != 2 || got[0] != " M README.md" || got[1] != "?? new.txt" {
		t.Fatalf("parsePorcelain = %#v", got)
	}
	if got := parsePorcelain("\n"); got != nil {
		t.Fatalf("empty parsePorcelain = %#v, want nil", got)
	}
}

func runGit(t *testing.T, git, root string, args ...string) {
	t.Helper()
	cmd := exec.Command(git, append([]string{"-C", root}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}
