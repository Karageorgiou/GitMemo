package upgrader

import (
	"testing"

	"github.com/runethread/core/internal/buildinfo"
)

func TestNativeCompatibilitySeparatesRuntimeAndContractRelease(t *testing.T) {
	cfg := repositoryConfig{
		RepositoryFormat:  buildinfo.RepositoryFormatVersion,
		SchemaVersion:     buildinfo.SchemaVersion,
		ContractVersion:   buildinfo.ContractVersion,
		RunethreadVersion: "v0.7.0",
	}

	if err := checkNativeCompatibilityFor(cfg, "v0.8.0", "v0.7.0"); err != nil {
		t.Fatalf("compatible contract pin rejected by newer runtime: %v", err)
	}

	cfg.RunethreadVersion = "v0.8.0"
	if err := checkNativeCompatibilityFor(cfg, "v0.8.0", "v0.7.0"); err == nil {
		t.Fatal("runtime release was incorrectly accepted as the repository contract pin")
	}
}
