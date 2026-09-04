package fsafety

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRequireSafeWritePathAcceptsMissingNestedTarget(t *testing.T) {
	root := t.TempDir()
	if err := RequireSafeWritePath(root, "nested/deeper/file.txt"); err != nil {
		t.Fatalf("missing safe write target rejected: %v", err)
	}
}

func TestRequireSafeWritePathRejectsSymlinkAncestor(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	link := filepath.Join(root, "nested")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symbolic links unavailable: %v", err)
	}
	if err := RequireSafeWritePath(root, "nested/deeper/file.txt"); err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("error = %v, want symbolic-link rejection", err)
	}
}

func TestRequireSafeWritePathRejectsDirectoryLeaf(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "target"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := RequireSafeWritePath(root, "target"); err == nil || !strings.Contains(err.Error(), "not a regular file") {
		t.Fatalf("error = %v, want non-regular leaf rejection", err)
	}
}
