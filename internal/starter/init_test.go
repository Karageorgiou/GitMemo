package starter

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/runethread/core/internal/buildinfo"
)

func TestInitCreatesSelfDescribingMemoryRepository(t *testing.T) {
	root := filepath.Join(t.TempDir(), "memory")
	if err := Init(root); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{
		"README.md",
		"MEMORY_PROTOCOL.md",
		"docs/USER_COMMANDS.md",
		"docs/EXTENDING_RUNETHREAD.md",
		"docs/TRUST_MODEL.md",
		"docs/SOURCES.md",
		"docs/INDEX_FORMAT.md",
		"schema/memory-item.schema.json",
		"templates/open_loop.md",
		".runethread/config.json",
		".runethread/lock.json",
		".github/workflows/validate.yml",
		"memories/.gitkeep",
		"projects/.gitkeep",
		"index/catalog.json",
		"index/projects.md",
		"index/open-loops.md",
		"index/preferences.md",
	} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); err != nil {
			t.Fatalf("missing %s: %v", rel, err)
		}
	}
	legacyMetadataDir := "." + "git" + "memo"
	legacyVersionField := "git" + "memo_version"
	if _, err := os.Stat(filepath.Join(root, legacyMetadataDir)); !os.IsNotExist(err) {
		t.Fatalf("native init must not create predecessor metadata directory: %v", err)
	}

	config, err := os.ReadFile(filepath.Join(root, ".runethread", "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	configText := string(config)
	versionField := `"runethread_version": "` + buildinfo.ContractReleaseVersion + `"`
	if !strings.Contains(configText, `"repository_format": `+strconv.Itoa(buildinfo.RepositoryFormatVersion)) || !strings.Contains(configText, `"schema_version": `+strconv.Itoa(buildinfo.SchemaVersion)) || !strings.Contains(configText, `"contract_version": `+strconv.Itoa(buildinfo.ContractVersion)) || !strings.Contains(configText, versionField) || strings.Contains(configText, legacyVersionField) {
		t.Fatalf("unexpected config: %s", config)
	}

	lock, err := os.ReadFile(filepath.Join(root, ".runethread", "lock.json"))
	if err != nil {
		t.Fatal(err)
	}
	lockText := string(lock)
	if !strings.Contains(lockText, `"lock_version": `+strconv.Itoa(buildinfo.TrustLockVersion)) || !strings.Contains(lockText, versionField) || !strings.Contains(lockText, `"source_repository": "runethread/core"`) || !strings.Contains(lockText, `"contract_sha256"`) || !strings.Contains(lockText, `"docs/TRUST_MODEL.md"`) || !strings.Contains(lockText, `"docs/INDEX_FORMAT.md"`) || strings.Contains(lockText, legacyVersionField) {
		t.Fatalf("unexpected trust lock: %s", lock)
	}

	readme, err := os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(readme), "Runethread: store") || !strings.Contains(string(readme), ".runethread/lock.json") {
		t.Fatalf("native README does not expose Runethread identity: %s", readme)
	}

	workflow, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "validate.yml"))
	if err != nil {
		t.Fatal(err)
	}
	workflowText := string(workflow)
	if !strings.Contains(workflowText, "contents: read") || !strings.Contains(workflowText, "github.com/runethread/core/cmd/runethread@"+buildinfo.BootstrapVerifierVersion) || !strings.Contains(workflowText, "trust version") || !strings.Contains(workflowText, "Managed by Runethread") {
		t.Fatalf("validation workflow is not the native read-only trust bootstrap: %s", workflow)
	}
}

func TestInitAllowsFreshGitRepository(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := Init(root); err != nil {
		t.Fatal(err)
	}
}

func TestInitRefusesNonEmptyDirectory(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "keep.txt"), []byte("do not replace"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Init(root); err == nil {
		t.Fatal("expected init to refuse a non-empty target")
	}
	data, err := os.ReadFile(filepath.Join(root, "keep.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "do not replace" {
		t.Fatal("existing file was modified")
	}
}
