package indexer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRequireOptionalRegularTreeRejectsSymlinkRoot(t *testing.T) {
	realRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(realRoot, "index"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(realRoot, "index", "catalog.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	parent := t.TempDir()
	rootLink := filepath.Join(parent, "repo")
	if err := os.Symlink(realRoot, rootLink); err != nil {
		t.Skipf("symbolic links unavailable: %v", err)
	}
	if err := requireOptionalRegularTree(rootLink, "index"); err == nil || !strings.Contains(err.Error(), "unsafe root") {
		t.Fatalf("error = %v, want unsafe-root rejection", err)
	}
}

func TestRequireOptionalRegularTreeAcceptsMissingOrRegularTree(t *testing.T) {
	root := t.TempDir()
	if err := requireOptionalRegularTree(root, "index"); err != nil {
		t.Fatalf("missing tree rejected: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "index", "terms"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "index", "terms", "abc.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := requireOptionalRegularTree(root, "index"); err != nil {
		t.Fatalf("regular tree rejected: %v", err)
	}
}
