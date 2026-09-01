package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitCommandCreatesRepository(t *testing.T) {
	root := filepath.Join(t.TempDir(), "memory")
	if code := run([]string{"init", root}); code != 0 {
		t.Fatalf("init exit code = %d", code)
	}
	if _, err := os.Stat(filepath.Join(root, "MEMORY_PROTOCOL.md")); err != nil {
		t.Fatalf("init did not create protocol: %v", err)
	}
}

func TestInitCommandRejectsTooManyTargets(t *testing.T) {
	if code := run([]string{"init", "one", "two"}); code != 2 {
		t.Fatalf("init exit code = %d, want 2", code)
	}
}
