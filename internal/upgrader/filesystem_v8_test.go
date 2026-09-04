package upgrader

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/runethread/core/internal/starter"
)

func TestManagedGitAttributesOwnershipAcceptsAbsentOrExactFile(t *testing.T) {
	root := t.TempDir()
	if err := checkManagedGitAttributesOwnership(root); err != nil {
		t.Fatalf("absent managed file rejected: %v", err)
	}
	mustWrite(t, filepath.Join(root, managedGitAttributesPath), starter.GitAttributes())
	if err := checkManagedGitAttributesOwnership(root); err != nil {
		t.Fatalf("exact managed file rejected: %v", err)
	}
}

func TestManagedGitAttributesOwnershipRejectsCustomOrSymlink(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, managedGitAttributesPath)
	mustWrite(t, path, []byte("*.md text\n"))
	if err := checkManagedGitAttributesOwnership(root); err == nil || !strings.Contains(err.Error(), "refusing to overwrite") {
		t.Fatalf("custom .gitattributes error = %v", err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "attributes")
	if err := os.WriteFile(outside, starter.GitAttributes(), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, path); err != nil {
		t.Skipf("symbolic links unavailable: %v", err)
	}
	if err := checkManagedGitAttributesOwnership(root); err == nil || !strings.Contains(err.Error(), "not the managed regular file") {
		t.Fatalf("symlink .gitattributes error = %v", err)
	}
}

func TestRequireMigrationRootRejectsSymlink(t *testing.T) {
	realRoot := t.TempDir()
	parent := t.TempDir()
	rootLink := filepath.Join(parent, "repo")
	if err := os.Symlink(realRoot, rootLink); err != nil {
		t.Skipf("symbolic links unavailable: %v", err)
	}
	if err := requireMigrationRoot(rootLink); err == nil || !strings.Contains(err.Error(), "unsafe repository root") {
		t.Fatalf("symlink root error = %v", err)
	}
}

func TestTakeRegularSnapshotsRejectsSymlinkAndPreservesRegularBytes(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "managed.txt"), []byte("managed\n"))
	snaps, err := takeRegularSnapshots(root, []string{"managed.txt", "absent.txt"})
	if err != nil {
		t.Fatal(err)
	}
	if !snaps["managed.txt"].exists || string(snaps["managed.txt"].data) != "managed\n" {
		t.Fatalf("unexpected regular snapshot: %#v", snaps["managed.txt"])
	}
	if snaps["absent.txt"].exists {
		t.Fatal("absent path was recorded as existing")
	}

	outside := filepath.Join(t.TempDir(), "outside.txt")
	if err := os.WriteFile(outside, []byte("outside\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "linked.txt")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symbolic links unavailable: %v", err)
	}
	if _, err := takeRegularSnapshots(root, []string{"linked.txt"}); err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("symlink snapshot error = %v", err)
	}
}
