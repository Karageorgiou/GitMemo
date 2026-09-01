package starter

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	gitmemo "github.com/Karageorgiou/GitMemo"
	"github.com/Karageorgiou/GitMemo/internal/indexer"
)

const contractVersion = 2

const memoryRepoReadme = `# GitMemo Memory

Private, user-owned persistent memory for AI assistants.

> **AI / LLM OPERATORS:** Read and follow [MEMORY_PROTOCOL.md](MEMORY_PROTOCOL.md) before retrieving from or modifying this repository. The user-facing command convention is defined in [docs/USER_COMMANDS.md](docs/USER_COMMANDS.md).

This repository contains memory data and a pinned operational contract. The GitMemo implementation itself lives separately at ` + "`github.com/Karageorgiou/GitMemo`" + `.

## Quick commands

- ` + "`GitMemo: remember ...`" + ` — explicit durable memory write.
- ` + "`GitMemo: search ...`" + ` — retrieval-only search; do not modify memories.

## Repository contents

- ` + "`MEMORY_PROTOCOL.md`" + ` — mandatory operating instructions.
- ` + "`docs/USER_COMMANDS.md`" + ` — user-facing remember/search command contract.
- ` + "`schema/`" + ` — machine-readable memory schema.
- ` + "`docs/`" + ` — memory format, taxonomy, and validation contract.
- ` + "`templates/`" + ` — authoring scaffolds for the eight memory types.
- ` + "`memories/`" + ` — atomic Markdown + JSON memory pairs.
- ` + "`projects/`" + ` — current-state views for active projects.
- ` + "`index/`" + ` — generated discovery indexes.
- ` + "`.gitmemo/config.json`" + ` — repository-format metadata.

Do not store credentials, authentication secrets, private keys, recovery codes, or other secret material in this repository.
`

type config struct {
	RepositoryFormat int `json:"repository_format"`
	SchemaVersion    int `json:"schema_version"`
	ContractVersion  int `json:"contract_version"`
}

// Init creates a new GitMemo memory repository skeleton at root. The target
// must not exist, must be empty, or may contain only a .git directory so that
// initialization is safe inside a freshly-created Git repository.
func Init(root string) error {
	if root == "" {
		return errors.New("target directory must not be empty")
	}
	if err := preflight(root); err != nil {
		return err
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return fmt.Errorf("create target directory: %w", err)
	}

	for _, rel := range gitmemo.ContractPaths() {
		data, err := fs.ReadFile(gitmemo.ContractFS, rel)
		if err != nil {
			return fmt.Errorf("read embedded contract %s: %w", rel, err)
		}
		if err := writeNew(root, rel, data); err != nil {
			return err
		}
	}

	if err := writeNew(root, "README.md", []byte(memoryRepoReadme)); err != nil {
		return err
	}
	cfg, err := json.MarshalIndent(config{RepositoryFormat: 1, SchemaVersion: 1, ContractVersion: contractVersion}, "", "  ")
	if err != nil {
		return err
	}
	cfg = append(cfg, '\n')
	if err := writeNew(root, ".gitmemo/config.json", cfg); err != nil {
		return err
	}

	for _, rel := range []string{"memories/.gitkeep", "projects/.gitkeep"} {
		if err := writeNew(root, rel, nil); err != nil {
			return err
		}
	}

	if err := indexer.Write(root); err != nil {
		return fmt.Errorf("generate initial indexes: %w", err)
	}
	return nil
}

func preflight(root string) error {
	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("inspect target: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("target %q is not a directory", root)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return fmt.Errorf("read target directory: %w", err)
	}
	var unexpected []string
	for _, entry := range entries {
		if entry.Name() == ".git" && entry.IsDir() {
			continue
		}
		unexpected = append(unexpected, entry.Name())
	}
	if len(unexpected) > 0 {
		sort.Strings(unexpected)
		return fmt.Errorf("target directory is not empty; refusing to overwrite: %v", unexpected)
	}
	return nil
}

func writeNew(root, rel string, data []byte) error {
	path := filepath.Join(root, filepath.FromSlash(rel))
	if _, err := os.Lstat(path); err == nil {
		return fmt.Errorf("refusing to overwrite existing path %s", rel)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect %s: %w", rel, err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create parent for %s: %w", rel, err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", rel, err)
	}
	return nil
}
