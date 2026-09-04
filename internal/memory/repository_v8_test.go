package memory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscoverStrictAcceptsRegularMemoryTree(t *testing.T) {
	root := t.TempDir()
	base := filepath.Join(root, "memories", "projects", "test")
	if err := os.MkdirAll(base, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, "item--11111111.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, "item--11111111.md"), []byte("# Item\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sidecars, markdown, err := DiscoverStrict(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(sidecars) != 1 || len(markdown) != 1 {
		t.Fatalf("strict discovery returned sidecars=%v markdown=%v", sidecars, markdown)
	}
}

func TestDiscoverStrictRejectsSymlinkedMemoryTree(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "memories")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "memories")); err != nil {
		t.Skipf("symbolic links unavailable: %v", err)
	}
	if _, _, err := DiscoverStrict(root); err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("strict discovery error = %v", err)
	}
}

func TestLoadUnderRejectsSymlinkedSidecar(t *testing.T) {
	root := t.TempDir()
	base := filepath.Join(root, "memories", "projects", "test")
	if err := os.MkdirAll(base, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "sidecar.json")
	if err := os.WriteFile(outside, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	rel := "memories/projects/test/item--11111111.json"
	if err := os.Symlink(outside, filepath.Join(root, filepath.FromSlash(rel))); err != nil {
		t.Skipf("symbolic links unavailable: %v", err)
	}
	_, problems := LoadUnder(root, rel)
	if len(problems) == 0 || !strings.Contains(problems[0].Error(), "symbolic link") {
		t.Fatalf("strict sidecar problems = %+v", problems)
	}
}

func TestValidateSchemaContractStrictRejectsSymlinkedSchema(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "schema"), 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "memory-item.schema.json")
	if err := os.WriteFile(outside, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "schema", "memory-item.schema.json")); err != nil {
		t.Skipf("symbolic links unavailable: %v", err)
	}
	if err := ValidateSchemaContractStrict(root); err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("strict schema error = %v", err)
	}
}
