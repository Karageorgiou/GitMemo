package memory

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const validID = "11111111-1111-4111-8111-111111111111"

func validMemoryMap() map[string]any {
	return map[string]any{
		"schema_version": 1,
		"id":             validID,
		"title":          "Test decision",
		"type":           "decision",
		"lifecycle":      "active",
		"summary":        "A valid test memory.",
		"projects":       []any{"test"},
		"topics":         []any{"memory-systems"},
		"tags":           []any{"test"},
		"aliases":        []any{},
		"entities":       []any{map[string]any{"kind": "project", "name": "Test"}},
		"importance":     "normal",
		"temporal":       map[string]any{"created_at": "2026-09-01T00:00:00Z", "updated_at": "2026-09-01T00:00:00Z", "effective_from": "2026-09-01", "effective_until": nil},
		"provenance":     map[string]any{"basis": "user_stated", "confidence": "high", "explicit_memory_request": false, "sources": []any{map[string]any{"kind": "conversation", "locator": "Unit test", "revision": nil, "note": nil}}},
		"relationships":  []any{},
		"content_path":   "memories/projects/test/test-decision--11111111.md",
		"sensitivity":    "routine",
	}
}

func encode(t *testing.T, value any) []byte {
	t.Helper()
	b, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func problemContains(problems []SchemaProblem, needle string) bool {
	for _, p := range problems {
		if strings.Contains(p.Error(), needle) {
			return true
		}
	}
	return false
}

func symlinkOrSkip(t *testing.T, oldname, newname string) {
	t.Helper()
	if err := os.Symlink(oldname, newname); err != nil {
		t.Skipf("symbolic links unavailable: %v", err)
	}
}

func TestSchemaContractMatchesRepositorySchema(t *testing.T) {
	root := filepath.Join("..", "..")
	if err := ValidateSchemaContract(root); err != nil {
		t.Fatalf("schema contract mismatch: %v", err)
	}
}

func TestDecodeValidMemory(t *testing.T) {
	m, problems := Decode(encode(t, validMemoryMap()))
	if len(problems) != 0 {
		t.Fatalf("unexpected problems: %+v", problems)
	}
	if m.ID != validID || m.Type != "decision" {
		t.Fatalf("unexpected decode: %+v", m)
	}
}

func TestDecodeRejectsUnknownRootField(t *testing.T) {
	v := validMemoryMap()
	v["invented_field"] = true
	_, problems := Decode(encode(t, v))
	if !problemContains(problems, "unknown field") {
		t.Fatalf("expected unknown field problem, got %+v", problems)
	}
}

func TestDecodeRejectsUnknownNestedField(t *testing.T) {
	v := validMemoryMap()
	v["temporal"].(map[string]any)["invented_field"] = true
	_, problems := Decode(encode(t, v))
	if !problemContains(problems, "unknown field") {
		t.Fatalf("expected nested unknown field problem, got %+v", problems)
	}
}

func TestDecodeRequiresFalseBooleanFieldPresence(t *testing.T) {
	v := validMemoryMap()
	delete(v["provenance"].(map[string]any), "explicit_memory_request")
	_, problems := Decode(encode(t, v))
	if !problemContains(problems, "provenance.explicit_memory_request") {
		t.Fatalf("expected missing boolean problem, got %+v", problems)
	}
}

func TestDecodeOpenLoopRequiresStatus(t *testing.T) {
	v := validMemoryMap()
	v["type"] = "open_loop"
	_, problems := Decode(encode(t, v))
	if !problemContains(problems, "open_loop_status") {
		t.Fatalf("expected open_loop_status problem, got %+v", problems)
	}
}

func TestDecodeNonOpenLoopRejectsStatus(t *testing.T) {
	v := validMemoryMap()
	v["open_loop_status"] = "open"
	_, problems := Decode(encode(t, v))
	if !problemContains(problems, "only allowed") {
		t.Fatalf("expected conditional status problem, got %+v", problems)
	}
}

func TestSchemaContractDetectsSemanticChange(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "schema"), 0o755); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join("..", "..", "schema", "memory-item.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	var value map[string]any
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatal(err)
	}
	value["title"] = "Changed"
	changed := encode(t, value)
	if err := os.WriteFile(filepath.Join(root, "schema", "memory-item.schema.json"), changed, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ValidateSchemaContract(root); err == nil {
		t.Fatal("expected schema contract mismatch")
	}
}

func TestSchemaContractRejectsSymlinkedSchema(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "schema"), 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "memory-item.schema.json")
	data, err := os.ReadFile(filepath.Join("..", "..", "schema", "memory-item.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outside, data, 0o644); err != nil {
		t.Fatal(err)
	}
	symlinkOrSkip(t, outside, filepath.Join(root, "schema", "memory-item.schema.json"))
	if err := ValidateSchemaContract(root); err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("error = %v, want symbolic-link rejection", err)
	}
}

func TestDiscoverRejectsSymlinkedMemoriesRoot(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	symlinkOrSkip(t, outside, filepath.Join(root, "memories"))
	if _, _, err := Discover(root); err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("error = %v, want symbolic-link rejection", err)
	}
}

func TestDiscoverRejectsSymlinkedMemoryFile(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "memories", "projects", "test")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "outside.json")
	if err := os.WriteFile(outside, encode(t, validMemoryMap()), 0o644); err != nil {
		t.Fatal(err)
	}
	symlinkOrSkip(t, outside, filepath.Join(dir, "linked--11111111.json"))
	if _, _, err := Discover(root); err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("error = %v, want symbolic-link rejection", err)
	}
}

func TestLoadRejectsSymlinkedSidecar(t *testing.T) {
	outside := filepath.Join(t.TempDir(), "outside.json")
	if err := os.WriteFile(outside, encode(t, validMemoryMap()), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "linked.json")
	symlinkOrSkip(t, outside, link)
	_, problems := Load(link)
	if !problemContains(problems, "symbolic link") {
		t.Fatalf("expected symbolic-link problem, got %+v", problems)
	}
}
