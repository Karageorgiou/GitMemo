package upgrader

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/runethread/core/internal/buildinfo"
	"github.com/runethread/core/internal/indexer"
	"github.com/runethread/core/internal/starter"
	"github.com/runethread/core/internal/trust"
)

const fixtureMemoryID = "11111111-1111-4111-8111-111111111111"

func TestApplyMigratesExactGitMemoV050SourceAndPreservesCanonicalData(t *testing.T) {
	root := legacyFixture(t)
	memoryJSON, memoryMD := writeFixtureMemory(t, root, false)
	customPath := filepath.Join(root, "projects", "user-notes.md")
	const custom = "# User notes\n\nDo not rewrite me.\n"
	mustWrite(t, customPath, []byte(custom))

	beforeJSON := mustRead(t, memoryJSON)
	beforeMD := mustRead(t, memoryMD)

	result, err := Apply(root)
	if err != nil {
		t.Fatal(err)
	}
	if result.FromVersion != legacyReleaseVersion || result.ToVersion != buildinfo.ReleaseVersion || result.FromContract != legacyContractVersion || result.ToContract != buildinfo.ContractVersion {
		t.Fatalf("unexpected migration result: %#v", result)
	}
	if result.AlreadyCurrent {
		t.Fatal("legacy migration incorrectly reported no changes")
	}
	if got := mustRead(t, memoryJSON); string(got) != string(beforeJSON) {
		t.Fatal("canonical memory JSON changed during identity migration")
	}
	if got := mustRead(t, memoryMD); string(got) != string(beforeMD) {
		t.Fatal("canonical memory Markdown changed during identity migration")
	}
	if got := string(mustRead(t, customPath)); got != custom {
		t.Fatalf("user-owned project file changed: %q", got)
	}

	if _, err := os.Stat(filepath.Join(root, ".gitmemo")); !os.IsNotExist(err) {
		t.Fatalf("legacy .gitmemo directory survived migration: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".runethread", "config.json")); err != nil {
		t.Fatalf("native config missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".runethread", "lock.json")); err != nil {
		t.Fatalf("native lock missing: %v", err)
	}
	config := string(mustRead(t, filepath.Join(root, ".runethread", "config.json")))
	if !strings.Contains(config, `"runethread_version": "`+buildinfo.ReleaseVersion+`"`) || strings.Contains(config, "gitmemo_version") {
		t.Fatalf("unexpected native config: %s", config)
	}
	lock := string(mustRead(t, filepath.Join(root, ".runethread", "lock.json")))
	if !strings.Contains(lock, `"source_repository": "runethread/core"`) || !strings.Contains(lock, `"runethread_version": "`+buildinfo.ReleaseVersion+`"`) || strings.Contains(lock, "gitmemo_version") {
		t.Fatalf("unexpected native lock: %s", lock)
	}
	if _, err := os.Stat(filepath.Join(root, legacyExtendingPath)); !os.IsNotExist(err) {
		t.Fatalf("legacy contract path survived migration: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "docs", "EXTENDING_RUNETHREAD.md")); err != nil {
		t.Fatalf("native extension contract missing: %v", err)
	}
	if problems := trust.Check(root); len(problems) != 0 {
		t.Fatalf("target trust check failed: %+v", problems)
	}
	if stale, err := indexer.Check(root); err != nil || len(stale) != 0 {
		t.Fatalf("target index is not fresh: stale=%v err=%v", stale, err)
	}
}

func TestApplyNativeRepositoryIsIdempotent(t *testing.T) {
	root := filepath.Join(t.TempDir(), "memory")
	if err := starter.Init(root); err != nil {
		t.Fatal(err)
	}
	result, err := Apply(root)
	if err != nil {
		t.Fatal(err)
	}
	if !result.AlreadyCurrent || len(result.ChangedPaths) != 0 {
		t.Fatalf("expected current native repository to be a no-op: %#v", result)
	}
}

func TestApplyRefusesTamperedLegacyContractBeforeWriting(t *testing.T) {
	root := legacyFixture(t)
	path := filepath.Join(root, "docs", "TRUST_MODEL.md")
	mustWrite(t, path, append(mustRead(t, path), []byte("\ntampered\n")...))
	legacyConfigBefore := mustRead(t, filepath.Join(root, ".gitmemo", "config.json"))

	if _, err := Apply(root); err == nil || !strings.Contains(err.Error(), "digest") {
		t.Fatalf("expected trusted-source digest refusal, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".runethread")); !os.IsNotExist(err) {
		t.Fatalf("target metadata was created despite preflight refusal: %v", err)
	}
	if got := mustRead(t, filepath.Join(root, ".gitmemo", "config.json")); string(got) != string(legacyConfigBefore) {
		t.Fatal("legacy config changed despite preflight refusal")
	}
}

func TestApplyRefusesMixedManagedMetadata(t *testing.T) {
	root := legacyFixture(t)
	mustWrite(t, filepath.Join(root, ".runethread", "config.json"), []byte("{}\n"))
	if _, err := Apply(root); err == nil || !strings.Contains(err.Error(), "mixed managed metadata") {
		t.Fatalf("expected mixed-state refusal, got %v", err)
	}
}

func TestApplyRefusesCustomValidationWorkflow(t *testing.T) {
	root := legacyFixture(t)
	path := filepath.Join(root, ".github", "workflows", "validate.yml")
	const custom = "name: My Custom Workflow\non: push\njobs: {}\n"
	mustWrite(t, path, []byte(custom))
	if _, err := Apply(root); err == nil || !strings.Contains(err.Error(), "refusing to overwrite") {
		t.Fatalf("expected custom workflow refusal, got %v", err)
	}
	if got := string(mustRead(t, path)); got != custom {
		t.Fatal("custom workflow was modified")
	}
	if _, err := os.Stat(filepath.Join(root, ".runethread")); !os.IsNotExist(err) {
		t.Fatalf("target metadata was created despite workflow refusal: %v", err)
	}
}

func TestApplyRollsBackLegacyMigrationWhenTargetValidationFails(t *testing.T) {
	root := legacyFixture(t)
	_, _ = writeFixtureMemory(t, root, true)
	legacyWorkflow := mustRead(t, filepath.Join(root, ".github", "workflows", "validate.yml"))
	legacyLock := mustRead(t, filepath.Join(root, ".gitmemo", "lock.json"))

	if _, err := Apply(root); err == nil || !strings.Contains(err.Error(), "failed validation") {
		t.Fatalf("expected post-migration validation failure, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".runethread")); !os.IsNotExist(err) {
		t.Fatalf("native metadata survived rollback: %v", err)
	}
	if got := mustRead(t, filepath.Join(root, ".gitmemo", "lock.json")); string(got) != string(legacyLock) {
		t.Fatal("legacy lock was not restored exactly")
	}
	if got := mustRead(t, filepath.Join(root, ".github", "workflows", "validate.yml")); string(got) != string(legacyWorkflow) {
		t.Fatal("legacy validation workflow was not restored exactly")
	}
	if _, err := os.Stat(filepath.Join(root, legacyExtendingPath)); err != nil {
		t.Fatalf("legacy extension contract was not restored: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "docs", "EXTENDING_RUNETHREAD.md")); !os.IsNotExist(err) {
		t.Fatalf("native extension contract survived rollback: %v", err)
	}
}

func TestApplyRefusesNewerNativeContract(t *testing.T) {
	root := filepath.Join(t.TempDir(), "memory")
	if err := starter.Init(root); err != nil {
		t.Fatal(err)
	}
	cfg := repositoryConfig{
		RepositoryFormat:  buildinfo.RepositoryFormatVersion,
		SchemaVersion:     buildinfo.SchemaVersion,
		ContractVersion:   buildinfo.ContractVersion + 1,
		RunethreadVersion: buildinfo.ReleaseVersion,
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(root, ".runethread", "config.json"), append(data, '\n'))
	if _, err := Apply(root); err == nil || !strings.Contains(err.Error(), "newer") {
		t.Fatalf("expected newer-contract refusal, got %v", err)
	}
}

func legacyFixture(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "memory")
	source := filepath.Join("testdata", "gitmemo-v0.5.0")
	if err := filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		target := filepath.Join(root, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	}); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "memories"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "projects"), 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}

func writeFixtureMemory(t *testing.T, root string, missingRelationship bool) (string, string) {
	t.Helper()
	base := filepath.Join(root, "memories", "projects", "test")
	if err := os.MkdirAll(base, 0o755); err != nil {
		t.Fatal(err)
	}
	projectDir := filepath.Join(root, "projects", "test")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(projectDir, "overview.md"), []byte("# Test — Project Overview\n"))

	relationships := []any{}
	if missingRelationship {
		relationships = []any{map[string]any{"type": "related_to", "target_id": "22222222-2222-4222-8222-222222222222", "note": nil}}
	}
	contentPath := "memories/projects/test/preserved-memory--11111111.md"
	memory := map[string]any{
		"schema_version": 1,
		"id":             fixtureMemoryID,
		"title":          "Preserved memory",
		"type":           "decision",
		"lifecycle":      "active",
		"summary":        "A canonical memory that must survive migration unchanged.",
		"projects":       []any{"test"},
		"topics":         []any{"migration"},
		"tags":           []any{"identity-cutover"},
		"aliases":        []any{},
		"entities":       []any{map[string]any{"kind": "project", "name": "Test"}},
		"importance":     "normal",
		"temporal": map[string]any{
			"created_at":      "2026-09-01T00:00:00Z",
			"updated_at":      "2026-09-01T00:00:00Z",
			"effective_from":  "2026-09-01",
			"effective_until": nil,
		},
		"provenance": map[string]any{
			"basis":                   "user_stated",
			"confidence":              "high",
			"explicit_memory_request": true,
			"sources": []any{map[string]any{
				"kind":     "conversation",
				"locator":  "migration fixture",
				"revision": nil,
				"note":     nil,
			}},
		},
		"relationships": relationships,
		"content_path":  contentPath,
		"sensitivity":   "routine",
	}
	data, err := json.MarshalIndent(memory, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	jsonPath := filepath.Join(base, "preserved-memory--11111111.json")
	mdPath := filepath.Join(base, "preserved-memory--11111111.md")
	mustWrite(t, jsonPath, append(data, '\n'))
	mustWrite(t, mdPath, []byte("# Preserved memory\n\n**Memory ID:** `"+fixtureMemoryID+"`  \n**Type:** `decision`\n\n## Context\n\nMigration test.\n\n## Decision\n\nPreserve this memory.\n\n## Rationale\n\nIdentity changes must not rewrite canonical memory data.\n"))
	return jsonPath, mdPath
}

func mustWrite(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
