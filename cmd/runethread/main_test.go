package main

import (
	"encoding/json"
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
	if _, err := os.Stat(filepath.Join(root, "index", "catalog.json")); err != nil {
		t.Fatalf("init did not create Index v2 catalog: %v", err)
	}
}

func TestTrustVersionReadsInitializedRepository(t *testing.T) {
	root := filepath.Join(t.TempDir(), "memory")
	if code := run([]string{"init", root}); code != 0 {
		t.Fatalf("init exit code = %d", code)
	}
	if code := run([]string{"trust", "version", root}); code != 0 {
		t.Fatalf("trust version exit code = %d", code)
	}
}

func TestInitCommandRejectsTooManyTargets(t *testing.T) {
	if code := run([]string{"init", "one", "two"}); code != 2 {
		t.Fatalf("init exit code = %d, want 2", code)
	}
}

func TestIndexMarkStaleCommand(t *testing.T) {
	root := filepath.Join(t.TempDir(), "memory")
	if code := run([]string{"init", root}); code != 0 {
		t.Fatalf("init exit code = %d", code)
	}
	if code := run([]string{"index", "--mark-stale", root}); code != 0 {
		t.Fatalf("index --mark-stale exit code = %d", code)
	}
	if _, err := os.Stat(filepath.Join(root, "index", "STALE")); err != nil {
		t.Fatalf("stale marker missing: %v", err)
	}
	if code := run([]string{"index", "--check", root}); code == 0 {
		t.Fatal("strict index check should fail while STALE marker exists")
	}
	if code := run([]string{"index", "--write", root}); code != 0 {
		t.Fatalf("index --write exit code = %d", code)
	}
	if _, err := os.Stat(filepath.Join(root, "index", "STALE")); !os.IsNotExist(err) {
		t.Fatalf("successful regeneration should remove stale marker, got %v", err)
	}
}

func TestSearchCommandFindsIndexedMemory(t *testing.T) {
	root := filepath.Join(t.TempDir(), "memory")
	if code := run([]string{"init", root}); code != 0 {
		t.Fatalf("init exit code = %d", code)
	}
	id := "11111111-1111-4111-8111-111111111111"
	base := filepath.Join(root, "memories", "projects", "test")
	if err := os.MkdirAll(base, 0o755); err != nil {
		t.Fatal(err)
	}
	contentPath := "memories/projects/test/carbonara--11111111.md"
	sidecar := map[string]any{
		"schema_version": 1,
		"id": id,
		"title": "Carbonara recipe preference",
		"type": "preference",
		"lifecycle": "active",
		"summary": "Prefer carbonara without cream.",
		"projects": []string{"test"},
		"topics": []string{"food"},
		"tags": []string{"carbonara"},
		"aliases": []string{"carbonara"},
		"entities": []any{},
		"importance": "normal",
		"temporal": map[string]any{"created_at":"2026-09-01T00:00:00Z","updated_at":"2026-09-01T00:00:00Z","effective_from":nil,"effective_until":nil},
		"provenance": map[string]any{"basis":"user_stated","confidence":"high","explicit_memory_request":true,"sources":[]any{map[string]any{"kind":"conversation","locator":"test","revision":nil,"note":nil}}},
		"relationships": []any{},
		"content_path": contentPath,
		"sensitivity": "routine",
	}
	data, err := json.MarshalIndent(sidecar, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, "carbonara--11111111.json"), append(data, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, "carbonara--11111111.md"), []byte("# Carbonara recipe preference\n\n**Memory ID:** `"+id+"`  \n**Type:** `preference`\n\n## Preference\n\nPrefer carbonara without cream.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if code := run([]string{"index", "--write", root}); code != 0 {
		t.Fatalf("index --write exit code = %d", code)
	}
	if code := run([]string{"search", "--root", root, "carbonara"}); code != 0 {
		t.Fatalf("search exit code = %d", code)
	}
}
