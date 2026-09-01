package validation

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Karageorgiou/GitMemo/internal/indexer"
)

const (
	idA = "11111111-1111-4111-8111-111111111111"
	idB = "22222222-2222-4222-8222-222222222222"
)

type fixture struct {
	t    *testing.T
	root string
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, "schema"))
	mustMkdir(t, filepath.Join(root, "memories", "projects", "test"))
	mustMkdir(t, filepath.Join(root, "projects", "test"))
	schema, err := os.ReadFile(filepath.Join("..", "..", "schema", "memory-item.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(root, "schema", "memory-item.schema.json"), schema)
	mustWrite(t, filepath.Join(root, "projects", "test", "overview.md"), []byte("# Test — Project Overview\n"))
	return &fixture{t: t, root: root}
}

func (f *fixture) memory(id, slug string) map[string]any {
	return map[string]any{
		"schema_version": 1, "id": id, "title": "Memory " + slug, "type": "decision", "lifecycle": "active", "summary": "Summary for " + slug + ".",
		"projects": []any{"test"}, "topics": []any{"memory-systems"}, "tags": []any{"test"}, "aliases": []any{},
		"entities": []any{map[string]any{"kind": "project", "name": "Test"}}, "importance": "normal",
		"temporal":      map[string]any{"created_at": "2026-09-01T00:00:00Z", "updated_at": "2026-09-01T00:00:00Z", "effective_from": "2026-09-01", "effective_until": nil},
		"provenance":    map[string]any{"basis": "user_stated", "confidence": "high", "explicit_memory_request": false, "sources": []any{map[string]any{"kind": "conversation", "locator": "Unit-test fixture", "revision": nil, "note": nil}}},
		"relationships": []any{}, "content_path": "memories/projects/test/" + slug + "--" + id[:8] + ".md", "sensitivity": "routine",
	}
}

func (f *fixture) writePair(id, slug string, mutate func(map[string]any)) (string, string) {
	f.t.Helper()
	m := f.memory(id, slug)
	if mutate != nil {
		mutate(m)
	}
	side := filepath.Join(f.root, "memories", "projects", "test", slug+"--"+id[:8]+".json")
	md := strings.TrimSuffix(side, ".json") + ".md"
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		f.t.Fatal(err)
	}
	mustWrite(f.t, side, append(b, '\n'))
	mustWrite(f.t, md, []byte(markdownFor(m)))
	return side, md
}

func markdownFor(m map[string]any) string {
	title := m["title"].(string)
	id := m["id"].(string)
	typ := m["type"].(string)
	header := "# " + title + "\n\n**Memory ID:** `" + id + "`  \n**Type:** `" + typ + "`\n\n"
	if typ == "open_loop" {
		status, _ := m["open_loop_status"].(string)
		if status == "resolved" || status == "cancelled" {
			return header + "## Original question or task\n\nTask.\n\n## Outcome\n\nDone.\n\n## Closure basis\n\nVerified.\n"
		}
		return header + "## Open question or task\n\nTask.\n\n## Why it remains open\n\nPending.\n\n## Next useful action\n\nDo it.\n"
	}
	return header + "## Context\n\nContext.\n\n## Decision\n\nDecision.\n\n## Rationale\n\nRationale.\n"
}

func (f *fixture) syncIndexes() {
	f.t.Helper()
	if err := indexer.Write(f.root); err != nil {
		f.t.Fatal(err)
	}
}
func codes(issues []Issue) map[string]bool {
	m := map[string]bool{}
	for _, i := range issues {
		m[i.Code] = true
	}
	return m
}
func mustMkdir(t *testing.T, p string) {
	t.Helper()
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatal(err)
	}
}
func mustWrite(t *testing.T, p string, b []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, b, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestValidRepositoryPasses(t *testing.T) {
	f := newFixture(t)
	f.writePair(idA, "valid-memory", nil)
	f.syncIndexes()
	if got := Validate(f.root); len(got) != 0 {
		t.Fatalf("unexpected issues: %+v", got)
	}
}
func TestMissingRelationshipTarget(t *testing.T) {
	f := newFixture(t)
	f.writePair(idA, "source", func(m map[string]any) {
		m["relationships"] = []any{map[string]any{"type": "related_to", "target_id": idB, "note": nil}}
	})
	f.syncIndexes()
	if !codes(Validate(f.root))["REL_TARGET_MISSING"] {
		t.Fatal("expected REL_TARGET_MISSING")
	}
}
func TestDuplicateUUID(t *testing.T) {
	f := newFixture(t)
	f.writePair(idA, "first", nil)
	f.writePair(idA, "second", nil)
	f.syncIndexes()
	if !codes(Validate(f.root))["UUID_DUPLICATE"] {
		t.Fatal("expected UUID_DUPLICATE")
	}
}
func TestFilenameUUIDMismatch(t *testing.T) {
	f := newFixture(t)
	side, md := f.writePair(idA, "wrong-suffix", nil)
	newSide := strings.Replace(side, idA[:8], "aaaaaaaa", 1)
	newMD := strings.Replace(md, idA[:8], "aaaaaaaa", 1)
	if err := os.Rename(side, newSide); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(md, newMD); err != nil {
		t.Fatal(err)
	}
	if !codes(Validate(f.root))["UUID_FILENAME"] {
		t.Fatal("expected UUID_FILENAME")
	}
}
func TestMissingMarkdown(t *testing.T) {
	f := newFixture(t)
	_, md := f.writePair(idA, "missing-markdown", nil)
	if err := os.Remove(md); err != nil {
		t.Fatal(err)
	}
	if !codes(Validate(f.root))["MISSING_MARKDOWN"] {
		t.Fatal("expected MISSING_MARKDOWN")
	}
}
func TestSupersededRequiresIncoming(t *testing.T) {
	f := newFixture(t)
	f.writePair(idA, "old", func(m map[string]any) { m["lifecycle"] = "superseded" })
	f.syncIndexes()
	if !codes(Validate(f.root))["SUPERSEDED_NO_INCOMING"] {
		t.Fatal("expected SUPERSEDED_NO_INCOMING")
	}
}
func TestSupersedesTargetMustBeSuperseded(t *testing.T) {
	f := newFixture(t)
	f.writePair(idA, "old", nil)
	f.writePair(idB, "new", func(m map[string]any) {
		m["relationships"] = []any{map[string]any{"type": "supersedes", "target_id": idA, "note": nil}}
	})
	f.syncIndexes()
	if !codes(Validate(f.root))["SUPERSEDES_TARGET_NOT_SUPERSEDED"] {
		t.Fatal("expected target lifecycle error")
	}
}
func TestSupersessionCycle(t *testing.T) {
	f := newFixture(t)
	f.writePair(idA, "a", func(m map[string]any) {
		m["lifecycle"] = "superseded"
		m["relationships"] = []any{map[string]any{"type": "supersedes", "target_id": idB, "note": nil}}
	})
	f.writePair(idB, "b", func(m map[string]any) {
		m["lifecycle"] = "superseded"
		m["relationships"] = []any{map[string]any{"type": "supersedes", "target_id": idA, "note": nil}}
	})
	f.syncIndexes()
	if !codes(Validate(f.root))["SUPERSESSION_CYCLE"] {
		t.Fatal("expected SUPERSESSION_CYCLE")
	}
}
func TestDuplicateRelationshipIgnoresNote(t *testing.T) {
	f := newFixture(t)
	f.writePair(idB, "target", nil)
	f.writePair(idA, "source", func(m map[string]any) {
		m["relationships"] = []any{map[string]any{"type": "related_to", "target_id": idB, "note": "one"}, map[string]any{"type": "related_to", "target_id": idB, "note": "two"}}
	})
	f.syncIndexes()
	if !codes(Validate(f.root))["REL_DUPLICATE"] {
		t.Fatal("expected REL_DUPLICATE")
	}
}
func TestEffectiveRange(t *testing.T) {
	f := newFixture(t)
	f.writePair(idA, "dates", func(m map[string]any) {
		tm := m["temporal"].(map[string]any)
		tm["effective_from"] = "2026-09-02"
		tm["effective_until"] = "2026-09-01"
	})
	f.syncIndexes()
	if !codes(Validate(f.root))["EFFECTIVE_RANGE_REVERSED"] {
		t.Fatal("expected EFFECTIVE_RANGE_REVERSED")
	}
}
func TestUpdatedBeforeCreated(t *testing.T) {
	f := newFixture(t)
	f.writePair(idA, "times", func(m map[string]any) {
		tm := m["temporal"].(map[string]any)
		tm["created_at"] = "2026-09-02T00:00:00Z"
		tm["updated_at"] = "2026-09-01T00:00:00Z"
	})
	f.syncIndexes()
	if !codes(Validate(f.root))["UPDATED_BEFORE_CREATED"] {
		t.Fatal("expected UPDATED_BEFORE_CREATED")
	}
}
func TestMarkdownIdentityMismatch(t *testing.T) {
	f := newFixture(t)
	_, md := f.writePair(idA, "md", nil)
	f.syncIndexes()
	data, _ := os.ReadFile(md)
	mustWrite(t, md, []byte(strings.Replace(string(data), idA, idB, 1)))
	if !codes(Validate(f.root))["MARKDOWN_UUID"] {
		t.Fatal("expected MARKDOWN_UUID")
	}
}
func TestTemplateScaffold(t *testing.T) {
	f := newFixture(t)
	_, md := f.writePair(idA, "scaffold", nil)
	f.syncIndexes()
	file, _ := os.OpenFile(md, os.O_APPEND|os.O_WRONLY, 0)
	_, _ = file.WriteString("\n<!-- instruction -->\n")
	_ = file.Close()
	if !codes(Validate(f.root))["TEMPLATE_SCAFFOLD"] {
		t.Fatal("expected TEMPLATE_SCAFFOLD")
	}
}
func TestUnknownSchemaField(t *testing.T) {
	f := newFixture(t)
	side, _ := f.writePair(idA, "unknown", nil)
	f.syncIndexes()
	var v map[string]any
	b, _ := os.ReadFile(side)
	_ = json.Unmarshal(b, &v)
	v["invented_field"] = true
	b, _ = json.Marshal(v)
	mustWrite(t, side, b)
	if !codes(Validate(f.root))["SCHEMA"] {
		t.Fatal("expected SCHEMA")
	}
}
func TestResolvedOpenLoopValid(t *testing.T) {
	f := newFixture(t)
	f.writePair(idA, "resolved", func(m map[string]any) { m["type"] = "open_loop"; m["open_loop_status"] = "resolved" })
	f.syncIndexes()
	if got := Validate(f.root); len(got) != 0 {
		t.Fatalf("unexpected issues: %+v", got)
	}
}
func TestResolvedOpenLoopRejectsUnresolvedHeadings(t *testing.T) {
	f := newFixture(t)
	_, md := f.writePair(idA, "resolved", func(m map[string]any) { m["type"] = "open_loop"; m["open_loop_status"] = "resolved" })
	f.syncIndexes()
	mustWrite(t, md, []byte("# Memory resolved\n\n**Memory ID:** `"+idA+"`  \n**Type:** `open_loop`\n\n## Open question or task\n\nTask.\n\n## Why it remains open\n\nNope.\n\n## Next useful action\n\nNope.\n"))
	if !codes(Validate(f.root))["OPEN_LOOP_MARKDOWN_FORM"] {
		t.Fatal("expected OPEN_LOOP_MARKDOWN_FORM")
	}
}
func TestUnresolvedOpenLoopRequiresNextAction(t *testing.T) {
	f := newFixture(t)
	_, md := f.writePair(idA, "open", func(m map[string]any) { m["type"] = "open_loop"; m["open_loop_status"] = "blocked" })
	f.syncIndexes()
	data, _ := os.ReadFile(md)
	mustWrite(t, md, []byte(strings.Replace(string(data), "## Next useful action", "## Something else", 1)))
	if !codes(Validate(f.root))["OPEN_LOOP_MARKDOWN_FORM"] {
		t.Fatal("expected OPEN_LOOP_MARKDOWN_FORM")
	}
}
func TestSchemaContractDrift(t *testing.T) {
	f := newFixture(t)
	path := filepath.Join(f.root, "schema", "memory-item.schema.json")
	var v map[string]any
	b, _ := os.ReadFile(path)
	_ = json.Unmarshal(b, &v)
	v["title"] = "changed"
	b, _ = json.Marshal(v)
	mustWrite(t, path, b)
	if !codes(Validate(f.root))["SCHEMA_CONTRACT"] {
		t.Fatal("expected SCHEMA_CONTRACT")
	}
}
func TestStaleIndexDetected(t *testing.T) {
	f := newFixture(t)
	side, _ := f.writePair(idA, "stale", nil)
	f.syncIndexes()
	var v map[string]any
	b, _ := os.ReadFile(side)
	_ = json.Unmarshal(b, &v)
	v["summary"] = "Changed summary."
	b, _ = json.MarshalIndent(v, "", "  ")
	mustWrite(t, side, b)
	if !codes(Validate(f.root))["INDEX_STALE"] {
		t.Fatal("expected INDEX_STALE")
	}
}
