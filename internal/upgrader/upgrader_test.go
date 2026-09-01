package upgrader

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/Karageorgiou/GitMemo/internal/buildinfo"
	"github.com/Karageorgiou/GitMemo/internal/starter"
)

func TestApplyUpgradesV010RepositoryAndPreservesUserData(t *testing.T) {
	root := filepath.Join(t.TempDir(), "memory")
	if err := starter.Init(root); err != nil {
		t.Fatal(err)
	}

	oldConfig := repositoryConfig{RepositoryFormat: 1, SchemaVersion: 1, ContractVersion: 3}
	configData, err := json.MarshalIndent(oldConfig, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	configData = append(configData, '\n')
	if err := os.WriteFile(filepath.Join(root, ".gitmemo", "config.json"), configData, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, ".gitmemo", "lock.json")); err != nil {
		t.Fatal(err)
	}

	workflowPath := filepath.Join(root, ".github", "workflows", "validate.yml")
	oldWorkflow := "name: Validate GitMemo Memory\n\non:\n  push:\n  pull_request:\n\npermissions:\n  contents: read\n\njobs:\n  validate:\n    runs-on: ubuntu-latest\n    steps:\n      - uses: actions/checkout@v7\n      - name: Install pinned GitMemo CLI\n        run: go install github.com/Karageorgiou/GitMemo/cmd/gitmemo@v0.1.0\n"
	if err := os.WriteFile(workflowPath, []byte(oldWorkflow), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, rel := range []string{"docs/EXTENDING_GITMEMO.md", "docs/TRUST_MODEL.md", "docs/SOURCES.md", "docs/INDEX_FORMAT.md"} {
		if err := os.Remove(filepath.Join(root, filepath.FromSlash(rel))); err != nil {
			t.Fatal(err)
		}
	}

	// Recreate the generated-index shape used by early GitMemo releases so the
	// upgrade test proves obsolete v1 files are removed rather than merely
	// upgrading a repository that was initialized by the current binary.
	indexDir := filepath.Join(root, "index")
	if err := os.RemoveAll(indexDir); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(indexDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for rel, content := range map[string]string{
		"memories.jsonl": "",
		"projects.md":    "# Projects\n",
		"open-loops.md":  "# Open Loops\n",
		"preferences.md": "# Preferences\n",
	} {
		if err := os.WriteFile(filepath.Join(indexDir, rel), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	customPath := filepath.Join(root, "projects", "user-notes.md")
	const custom = "# User notes\n\nDo not rewrite me.\n"
	if err := os.WriteFile(customPath, []byte(custom), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Apply(root)
	if err != nil {
		t.Fatal(err)
	}
	if result.FromVersion != "v0.1.0" || result.ToVersion != buildinfo.ReleaseVersion {
		t.Fatalf("unexpected version transition: %#v", result)
	}
	if result.FromContract != 3 || result.ToContract != buildinfo.ContractVersion {
		t.Fatalf("unexpected contract transition: %#v", result)
	}
	if result.AlreadyCurrent {
		t.Fatal("upgrade incorrectly reported repository as already current")
	}

	data, err := os.ReadFile(customPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != custom {
		t.Fatalf("user project data changed: %q", data)
	}

	config, err := os.ReadFile(filepath.Join(root, ".gitmemo", "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	versionField := `"gitmemo_version": "` + buildinfo.ReleaseVersion + `"`
	contractField := `"contract_version": ` + strconv.Itoa(buildinfo.ContractVersion)
	if !strings.Contains(string(config), contractField) || !strings.Contains(string(config), versionField) {
		t.Fatalf("config not upgraded: %s", config)
	}

	lock, err := os.ReadFile(filepath.Join(root, ".gitmemo", "lock.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(lock), versionField) || !strings.Contains(string(lock), `"contract_sha256"`) || !strings.Contains(string(lock), `"docs/INDEX_FORMAT.md"`) {
		t.Fatalf("trust lock not installed with Index v2 contract: %s", lock)
	}

	workflow, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(workflow), "Stable bootstrap workflow v1") || !strings.Contains(string(workflow), "trust version") || !strings.Contains(string(workflow), "@"+buildinfo.BootstrapVerifierVersion) {
		t.Fatalf("workflow not upgraded to stable bootstrap: %s", workflow)
	}
	for _, rel := range []string{"docs/EXTENDING_GITMEMO.md", "docs/TRUST_MODEL.md", "docs/SOURCES.md", "docs/INDEX_FORMAT.md"} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); err != nil {
			t.Fatalf("contract file %s not installed: %v", rel, err)
		}
	}
	if _, err := os.Stat(filepath.Join(indexDir, "memories.jsonl")); !os.IsNotExist(err) {
		t.Fatalf("legacy monolithic machine index survived upgrade: %v", err)
	}
	if _, err := os.Stat(filepath.Join(indexDir, "catalog.json")); err != nil {
		t.Fatalf("Index v2 catalog not generated: %v", err)
	}
}

func TestApplyIsIdempotent(t *testing.T) {
	root := filepath.Join(t.TempDir(), "memory")
	if err := starter.Init(root); err != nil {
		t.Fatal(err)
	}
	result, err := Apply(root)
	if err != nil {
		t.Fatal(err)
	}
	if !result.AlreadyCurrent || len(result.ChangedPaths) != 0 {
		t.Fatalf("expected current repository to be a no-op: %#v", result)
	}
}

func TestApplyRefusesNewerContract(t *testing.T) {
	root := filepath.Join(t.TempDir(), "memory")
	if err := starter.Init(root); err != nil {
		t.Fatal(err)
	}
	cfg := repositoryConfig{
		RepositoryFormat: 1,
		SchemaVersion:    1,
		ContractVersion:  buildinfo.ContractVersion + 1,
		GitMemoVersion:   "v99.0.0",
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(root, ".gitmemo", "config.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(root); err == nil || !strings.Contains(err.Error(), "newer") {
		t.Fatalf("expected newer-contract refusal, got %v", err)
	}
}

func TestApplyRefusesCustomValidationWorkflow(t *testing.T) {
	root := filepath.Join(t.TempDir(), "memory")
	if err := starter.Init(root); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, ".github", "workflows", "validate.yml")
	const custom = "name: My Custom Workflow\non: push\njobs: {}\n"
	if err := os.WriteFile(path, []byte(custom), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(root); err == nil || !strings.Contains(err.Error(), "refusing to overwrite") {
		t.Fatalf("expected custom workflow refusal, got %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != custom {
		t.Fatal("custom workflow was modified")
	}
}
