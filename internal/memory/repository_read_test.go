package memory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadUnderRejectsSymlinkAncestor(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	data, err := os.ReadFile(filepath.Join("..", "..", "internal", "upgrader", "testdata", "runethread-v0.7.0", ".runethread", "config.json"))
	if err != nil {
		// The fixture is deliberately not a memory sidecar. We only need bytes at
		// the target because filesystem rejection must happen before decoding.
		data = []byte("{}\n")
	}
	if err := os.WriteFile(filepath.Join(outside, "linked.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "linked")); err != nil {
		t.Skipf("symbolic links unavailable: %v", err)
	}
	_, problems := LoadUnder(root, "linked/linked.json")
	if len(problems) == 0 || !strings.Contains(problems[0].Error(), "symbolic link") {
		t.Fatalf("problems = %+v, want symbolic-link rejection", problems)
	}
}
