package indexer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setup(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	mustDir(t, filepath.Join(root, "memories", "projects", "test"))
	mustDir(t, filepath.Join(root, "projects", "test"))
	mustWrite(t, filepath.Join(root, "projects", "test", "overview.md"), "# Test — Project Overview\n")
	return root
}
func writeMemory(t *testing.T, root, id, slug, typ, status string) {
	t.Helper()
	m := map[string]any{"schema_version": 1, "id": id, "title": "Memory " + slug, "type": typ, "lifecycle": "active", "summary": "Summary.", "projects": []any{"test"}, "topics": []any{"topic"}, "tags": []any{"tag"}, "aliases": []any{}, "entities": []any{}, "importance": "normal", "temporal": map[string]any{"created_at": "2026-09-01T00:00:00Z", "updated_at": "2026-09-01T00:00:00Z", "effective_from": nil, "effective_until": nil}, "provenance": map[string]any{"basis": "user_stated", "confidence": "high", "explicit_memory_request": false, "sources": []any{map[string]any{"kind": "conversation", "locator": "test", "revision": nil, "note": nil}}}, "relationships": []any{}, "content_path": "memories/projects/test/" + slug + "--" + id[:8] + ".md", "sensitivity": "routine"}
	if typ == "open_loop" {
		m["open_loop_status"] = status
	}
	b, _ := json.MarshalIndent(m, "", "  ")
	mustWrite(t, filepath.Join(root, "memories", "projects", "test", slug+"--"+id[:8]+".json"), string(b)+"\n")
	mustWrite(t, filepath.Join(root, "memories", "projects", "test", slug+"--"+id[:8]+".md"), "# Memory "+slug+"\n")
}
func mustDir(t *testing.T, p string) {
	t.Helper()
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatal(err)
	}
}
func mustWrite(t *testing.T, p, s string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestMemoriesJSONLIsSortedByID(t *testing.T) {
	root := setup(t)
	writeMemory(t, root, "22222222-2222-4222-8222-222222222222", "b", "decision", "")
	writeMemory(t, root, "11111111-1111-4111-8111-111111111111", "a", "decision", "")
	r, err := Generate(root)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(r.Files["index/memories.jsonl"])), "\n")
	if !strings.Contains(lines[0], "11111111-") {
		t.Fatalf("not sorted: %s", r.Files["index/memories.jsonl"])
	}
}
func TestOpenLoopsExcludeResolved(t *testing.T) {
	root := setup(t)
	writeMemory(t, root, "11111111-1111-4111-8111-111111111111", "open", "open_loop", "open")
	writeMemory(t, root, "22222222-2222-4222-8222-222222222222", "done", "open_loop", "resolved")
	r, err := Generate(root)
	if err != nil {
		t.Fatal(err)
	}
	s := string(r.Files["index/open-loops.md"])
	if !strings.Contains(s, "Memory open") || strings.Contains(s, "Memory done") {
		t.Fatalf("wrong open-loop index:\n%s", s)
	}
}
func TestPreferencesEmptyMessage(t *testing.T) {
	root := setup(t)
	r, err := Generate(root)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(r.Files["index/preferences.md"]), "No atomic preference memories") {
		t.Fatal("missing empty preference message")
	}
}
func TestProjectsIndexUsesOverviewTitle(t *testing.T) {
	root := setup(t)
	r, err := Generate(root)
	if err != nil {
		t.Fatal(err)
	}
	s := string(r.Files["index/projects.md"])
	if !strings.Contains(s, "## Test") || !strings.Contains(s, "projects/test/overview.md") {
		t.Fatalf("bad projects index:\n%s", s)
	}
}
func TestWriteAndCheck(t *testing.T) {
	root := setup(t)
	writeMemory(t, root, "11111111-1111-4111-8111-111111111111", "a", "decision", "")
	if err := Write(root); err != nil {
		t.Fatal(err)
	}
	stale, err := Check(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(stale) != 0 {
		t.Fatalf("unexpected stale indexes: %v", stale)
	}
}
func TestCheckDetectsStale(t *testing.T) {
	root := setup(t)
	writeMemory(t, root, "11111111-1111-4111-8111-111111111111", "a", "decision", "")
	if err := Write(root); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(root, "index", "open-loops.md"), "stale\n")
	stale, err := Check(root)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, p := range stale {
		if p == "index/open-loops.md" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected stale open-loops index, got %v", stale)
	}
}
func TestGenerateDeterministic(t *testing.T) {
	root := setup(t)
	writeMemory(t, root, "11111111-1111-4111-8111-111111111111", "a", "decision", "")
	a, err := Generate(root)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Generate(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range IndexPaths {
		if string(a.Files[p]) != string(b.Files[p]) {
			t.Fatalf("non-deterministic output for %s", p)
		}
	}
}
