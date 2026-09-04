package validation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRequireContentPathFileRejectsSymlinkLeaf(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.md")
	if err := os.WriteFile(outside, []byte("outside\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "linked.md")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symbolic links unavailable: %v", err)
	}
	if err := requireContentPathFile(root, "linked.md"); err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("error = %v, want symbolic-link rejection", err)
	}
}

func TestReadMarkdownFileRejectsSymlinkAncestor(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "memory.md"), []byte("outside\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	linkDir := filepath.Join(root, "linked")
	if err := os.Symlink(outside, linkDir); err != nil {
		t.Skipf("symbolic links unavailable: %v", err)
	}
	if _, err := readMarkdownFile(root, filepath.Join(linkDir, "memory.md")); err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("error = %v, want ancestor symbolic-link rejection", err)
	}
}
