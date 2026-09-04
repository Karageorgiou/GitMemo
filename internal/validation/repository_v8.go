package validation

import (
	"path/filepath"

	"github.com/runethread/core/internal/buildinfo"
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
