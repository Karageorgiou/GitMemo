package validation

import (
	"os"
	"path/filepath"

	"github.com/runethread/core/internal/buildinfo"
	"github.com/runethread/core/internal/fsafety"
	"github.com/runethread/core/internal/memory"
)

func validateSchemaContractForRepository(root string) error {
	if buildinfo.ContractVersion >= 8 {
		return memory.ValidateSchemaContractStrict(root)
	}
	return memory.ValidateSchemaContract(root)
}

func discoverMemoryFiles(root string) (sidecars []string, markdown []string, err error) {
	if buildinfo.ContractVersion >= 8 {
		return memory.DiscoverStrict(root)
	}
	return memory.Discover(root)
}

func loadMemorySidecar(root, path string) (memory.Memory, []memory.SchemaProblem) {
	if buildinfo.ContractVersion < 8 {
		return memory.Load(path)
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return memory.Memory{}, []memory.SchemaProblem{{Message: err.Error()}}
	}
	return memory.LoadUnder(root, filepath.ToSlash(rel))
}

func requireContentPathFile(root, rel string) error {
	if buildinfo.ContractVersion >= 8 {
		_, err := fsafety.RegularFileUnder(root, rel)
		return err
	}
	_, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel)))
	return err
}

func readMarkdownFile(root, path string) ([]byte, error) {
	if buildinfo.ContractVersion < 8 {
		return os.ReadFile(path)
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return nil, err
	}
	return fsafety.ReadRegularFileUnder(root, filepath.ToSlash(rel))
}
