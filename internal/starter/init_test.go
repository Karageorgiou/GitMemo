package starter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
		"docs/EXTENDING_GITMEMO.md",
		"schema/memory-item.schema.json",
		"templates/open_loop.md",
		".gitmemo/config.json",
		".github/workflows/validate.yml",
		"memories/.gitkeep",
		"projects/.gitkeep",
		"index/memories.jsonl",
		"index/projects.md",
		"index/open-loops.md",
		"index/preferences.md",
	} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); err != nil {
			t.Fatalf("missing %s: %v", rel, err)
		}
	}

	config, err := os.ReadFile(filepath.Join(root, ".gitmemo", "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	configText := string(config)
	if !strings.Contains(configText, `"repository_format": 1`) || !strings.Contains(configText, `"contract_version": 4`) || !strings.Contains(configText, `"gitmemo_version": "v0.2.0"`) {
		t.Fatalf("unexpected config: %s", config)
	}

	commands, err := os.ReadFile(filepath.Join(root, "docs", "USER_COMMANDS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(commands), "GitMemo: store") || !strings.Contains(string(commands), "GitMemo: search") {
		t.Fatalf("command contract missing store/search interface: %s", commands)
	}

	workflow, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "validate.yml"))
	if err != nil {
		t.Fatal(err)
	}
	workflowText := string(workflow)
	if !strings.Contains(workflowText, "contents: read") || !strings.Contains(workflowText, "@v0.2.0") || !strings.Contains(workflowText, "# Managed by GitMemo.") {
		t.Fatalf("validation workflow is not read-only, managed, and release-pinned: %s", workflow)
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
