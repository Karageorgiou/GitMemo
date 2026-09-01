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
	writeMemoryText(t, root, id, slug, typ, status, "Memory "+slug, "Summary.")
}

func writeMemoryText(t *testing.T, root, id, slug, typ, status, title, summary string) {
	t.Helper()
	m := map[string]any{
		"schema_version": 1,
		"id":             id,
		"title":          title,
		"type":           typ,
		"lifecycle":      "active",
		"summary":        summary,
		"projects":       []any{"test"},
		"topics":         []any{"topic"},
		"tags":           []any{"tag"},
		"aliases":        []any{},
		"entities":       []any{},
		"importance":     "normal",
		"temporal": map[string]any{
			"created_at": "2026-09-01T00:00:00Z", "updated_at": "2026-09-01T00:00:00Z", "effective_from": nil, "effective_until": nil,
		},
		"provenance": map[string]any{
			"basis": "user_stated", "confidence": "high", "explicit_memory_request": false,
			"sources": []any{map[string]any{"kind": "conversation", "locator": "test", "revision": nil, "note": nil}},
		},
		"relationships": []any{},
		"content_path": "memories/projects/test/" + slug + "--" + id[:8] + ".md",
		"sensitivity": "routine",
	}
	if typ == "open_loop" {
		m["open_loop_status"] = status
	}
	b, _ := json.MarshalIndent(m, "", "  ")
	mustWrite(t, filepath.Join(root, "memories", "projects", "test", slug+"--"+id[:8]+".json"), string(b)+"\n")
	mustWrite(t, filepath.Join(root, "memories", "projects", "test", slug+"--"+id[:8]+".md"), "# "+title+"\n")
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

func TestGenerateUsesShardedIndexV2(t *testing.T) {
	root := setup(t)
	writeMemory(t, root, "22222222-2222-4222-8222-222222222222", "b", "decision", "")
	writeMemory(t, root, "11111111-1111-4111-8111-111111111111", "a", "decision", "")

	r, err := Generate(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{
		"index/catalog.json",
		"index/by-id/11.json",
		"index/by-id/22.json",
		"index/by-project/test.json",
		"index/by-topic/topic.json",
		"index/by-tag/tag.json",
		"index/by-type/decision.json",
		"index/by-lifecycle/active.json",
	} {
		if _, ok := r.Files[path]; !ok {
			t.Fatalf("missing generated path %s", path)
		}
	}
	if _, ok := r.Files["index/memories.jsonl"]; ok {
		t.Fatal("v2 index must not emit the monolithic memories.jsonl file")
	}

	var cat catalog
	if err := json.Unmarshal(r.Files["index/catalog.json"], &cat); err != nil {
		t.Fatal(err)
	}
	if cat.IndexVersion != 2 || cat.RecordCount != 2 || cat.MemorySourceSHA256 == "" {
		t.Fatalf("unexpected catalog: %#v", cat)
	}

	var shard map[string]indexEntry
	if err := json.Unmarshal(r.Files["index/by-id/11.json"], &shard); err != nil {
		t.Fatal(err)
	}
	if shard["11111111-1111-4111-8111-111111111111"].Title != "Memory a" {
		t.Fatalf("unexpected ID shard: %#v", shard)
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
	if _, ok := r.Files["index/by-open-loop-status/open.json"]; !ok {
		t.Fatal("missing open-loop status machine index")
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

func TestWriteRemovesLegacyAndStaleFiles(t *testing.T) {
	root := setup(t)
	writeMemory(t, root, "11111111-1111-4111-8111-111111111111", "a", "decision", "")
	mustWrite(t, filepath.Join(root, "index", "memories.jsonl"), "legacy\n")
	mustWrite(t, filepath.Join(root, filepath.FromSlash(StaleMarkerPath)), "stale\n")
	if err := Write(root); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{"index/memories.jsonl", StaleMarkerPath} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); !os.IsNotExist(err) {
			t.Fatalf("obsolete generated path %s survived regeneration", rel)
		}
	}
}

func TestCheckDetectsStaleAndUnexpectedFiles(t *testing.T) {
	root := setup(t)
	writeMemory(t, root, "11111111-1111-4111-8111-111111111111", "a", "decision", "")
	if err := Write(root); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(root, "index", "open-loops.md"), "stale\n")
	mustWrite(t, filepath.Join(root, "index", "unexpected.json"), "{}\n")
	stale, err := Check(root)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(stale, "\n")
	if !strings.Contains(joined, "index/open-loops.md") || !strings.Contains(joined, "index/unexpected.json") {
		t.Fatalf("expected stale and unexpected paths, got %v", stale)
	}
}

func TestMarkStale(t *testing.T) {
	root := setup(t)
	if err := MarkStale(root); err != nil {
		t.Fatal(err)
	}
	if !IsMarkedStale(root) {
		t.Fatal("expected explicit stale marker")
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
	if len(a.Files) != len(b.Files) {
		t.Fatalf("file count changed between deterministic runs: %d != %d", len(a.Files), len(b.Files))
	}
	for path, expected := range a.Files {
		if string(expected) != string(b.Files[path]) {
			t.Fatalf("non-deterministic output for %s", path)
		}
	}
}

func TestUnicodeTermsUseReadableShardPrefix(t *testing.T) {
	root := setup(t)
	writeMemoryText(t, root, "11111111-1111-4111-8111-111111111111", "athens", "fact", "", "Αθήνα ταξίδι", "Σημειώσεις για την Αθήνα.")
	r, err := Generate(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := r.Files["index/terms/αθ.json"]; !ok {
		t.Fatalf("expected readable Unicode term shard, got generated paths: %v", mapKeys(r.Files))
	}
}

func mapKeys(m map[string][]byte) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	return keys
}
