package upgrader

import (
	"encoding/json"
	"os"
	"path/filepath"
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

	workflowPath := filepath.Join(root, ".github", "workflows", "validate.yml")
	workflow, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatal(err)
	}
	workflow = []byte(strings.ReplaceAll(string(workflow), "@"+buildinfo.ReleaseVersion, "@v0.1.0"))
	workflow = []byte(strings.ReplaceAll(string(workflow), "# Managed by GitMemo. Updated by gitmemo upgrade.\n", ""))
	if err := os.WriteFile(workflowPath, workflow, 0o644); err != nil {
		t.Fatal(err)
	}

	extensionPath := filepath.Join(root, "docs", "EXTENDING_GITMEMO.md")
	if err := os.Remove(extensionPath); err != nil {
		t.Fatal(err)
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
	if !strings.Contains(string(config), `"contract_version": 4`) || !strings.Contains(string(config), `"gitmemo_version": "v0.2.0"`) {
		t.Fatalf("config not upgraded: %s", config)
	}

	workflow, err = os.ReadFile(workflowPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(workflow), "@v0.2.0") || !strings.Contains(string(workflow), "# Managed by GitMemo.") {
		t.Fatalf("workflow not upgraded: %s", workflow)
	}
	if _, err := os.Stat(extensionPath); err != nil {
		t.Fatalf("extension contract not installed: %v", err)
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
