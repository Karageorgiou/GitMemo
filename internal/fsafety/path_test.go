package fsafety

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadRegularFileUnderAcceptsNestedRegularFile(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "nested", "file.txt")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("ok\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	data, err := ReadRegularFileUnder(root, "nested/file.txt")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "ok\n" {
		t.Fatalf("data = %q", data)
	}
}

func TestDirectoryUnderAcceptsNestedDirectory(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "nested", "dir")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := DirectoryUnder(root, "nested/dir")
	if err != nil {
		t.Fatal(err)
	}
	if got != path {
		t.Fatalf("directory = %q, want %q", got, path)
	}
}

func TestRequireTreeAcceptsDirectoriesAndRegularFiles(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "tree", "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "tree", "nested", "file.txt"), []byte("ok\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := RequireTree(root, "tree"); err != nil {
		t.Fatal(err)
	}
}

func TestRequireTreeRejectsSymlinkEntry(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "tree"), 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "outside.txt")
	if err := os.WriteFile(outside, []byte("outside\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "tree", "linked.txt")); err != nil {
		t.Skipf("symbolic links unavailable: %v", err)
	}
	if err := RequireTree(root, "tree"); err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("error = %v, want symbolic-link rejection", err)
	}
}

func TestRegularFileUnderRejectsEscapingPath(t *testing.T) {
	root := t.TempDir()
	if _, err := RegularFileUnder(root, "../outside.txt"); err == nil || !strings.Contains(err.Error(), "escapes repository root") {
		t.Fatalf("error = %v, want repository escape rejection", err)
	}
}

func TestReadRegularFileUnderRejectsSymlinkLeaf(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.txt")
	if err := os.WriteFile(outside, []byte("outside\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link.txt")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symbolic links unavailable: %v", err)
	}
	if _, err := ReadRegularFileUnder(root, "link.txt"); err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("error = %v, want symbolic-link rejection", err)
	}
}

func TestReadRegularFileUnderRejectsSymlinkAncestor(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "file.txt"), []byte("outside\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	linkDir := filepath.Join(root, "linked")
	if err := os.Symlink(outside, linkDir); err != nil {
		t.Skipf("symbolic links unavailable: %v", err)
	}
	if _, err := ReadRegularFileUnder(root, "linked/file.txt"); err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("error = %v, want ancestor symbolic-link rejection", err)
	}
}

func TestReadRegularFileUnderRejectsSymlinkRoot(t *testing.T) {
	realRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(realRoot, "file.txt"), []byte("outside\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	parent := t.TempDir()
	rootLink := filepath.Join(parent, "root")
	if err := os.Symlink(realRoot, rootLink); err != nil {
		t.Skipf("symbolic links unavailable: %v", err)
	}
	if _, err := ReadRegularFileUnder(rootLink, "file.txt"); err == nil || !strings.Contains(err.Error(), "unsafe root") {
		t.Fatalf("error = %v, want unsafe-root rejection", err)
	}
}
