package upgrader

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOptionalManagedFileRejectsSymlinkAncestorWhenLeafIsMissing(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	link := filepath.Join(root, ".github")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symbolic links unavailable: %v", err)
	}

	_, _, err := readOptionalRepositoryRegularFile(root, ".github/workflows/validate.yml")
	if err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("error = %v, want symbolic-link rejection", err)
	}
}

func TestMigrationSnapshotPreflightRejectsSymlinkWriteAncestor(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	link := filepath.Join(root, ".github")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symbolic links unavailable: %v", err)
	}

	_, err := takeRegularSnapshots(root, []string{".github/workflows/validate.yml"})
	if err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("error = %v, want write-ancestor symbolic-link rejection", err)
	}
	if _, statErr := os.Lstat(filepath.Join(outside, "workflows")); !os.IsNotExist(statErr) {
		t.Fatalf("outside path was modified during preflight: %v", statErr)
	}
}
