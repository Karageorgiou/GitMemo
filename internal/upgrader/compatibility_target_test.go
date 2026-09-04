package upgrader

import (
	"strings"
	"testing"
)

func TestCurrentCompatibilityTargetAcceptsPublishedNativeV060AndV070(t *testing.T) {
	target := currentNativeCompatibilityTarget()
	for _, release := range []string{nativeV060ReleaseVersion, nativeV070ReleaseVersion} {
		cfg := repositoryConfig{
			RepositoryFormat:  2,
			SchemaVersion:     1,
			ContractVersion:   7,
			RunethreadVersion: release,
		}
		if err := checkNativeCompatibilityForTarget(cfg, target); err != nil {
			t.Fatalf("%s rejected by current target: %v", release, err)
		}
	}
}

func TestRuntimeOnlyFutureReleaseKeepsOlderContractPin(t *testing.T) {
	target := nativeCompatibilityTarget{
		RuntimeRelease:   "v0.9.0",
		ContractRelease:  "v0.8.0",
		RepositoryFormat: 2,
		SchemaVersion:    1,
		ContractVersion:  8,
	}
	cfg := repositoryConfig{
		RepositoryFormat:  2,
		SchemaVersion:     1,
		ContractVersion:   8,
		RunethreadVersion: "v0.8.0",
	}
	if err := checkNativeCompatibilityForTarget(cfg, target); err != nil {
		t.Fatalf("runtime-only release rejected unchanged contract pin: %v", err)
	}

	cfg.RunethreadVersion = "v0.9.0"
	if err := checkNativeCompatibilityForTarget(cfg, target); err == nil || !strings.Contains(err.Error(), "expects contract release \"v0.8.0\"") {
		t.Fatalf("incorrect runtime pin was not rejected as a contract pin: %v", err)
	}
}

func TestContractV8TargetRetainsExactContractV7SourceAnchors(t *testing.T) {
	target := nativeCompatibilityTarget{
		RuntimeRelease:   "v0.8.0",
		ContractRelease:  "v0.8.0",
		RepositoryFormat: 2,
		SchemaVersion:    1,
		ContractVersion:  8,
	}
	for _, release := range []string{nativeV060ReleaseVersion, nativeV070ReleaseVersion} {
		cfg := repositoryConfig{
			RepositoryFormat:  2,
			SchemaVersion:     1,
			ContractVersion:   7,
			RunethreadVersion: release,
		}
		if err := checkNativeCompatibilityForTarget(cfg, target); err != nil {
			t.Fatalf("contract-v7 source %s rejected by v8 target: %v", release, err)
		}
	}
}

func TestCompatibilityTargetRejectsUnknownNewerContract(t *testing.T) {
	target := nativeCompatibilityTarget{
		RuntimeRelease:   "v0.8.0",
		ContractRelease:  "v0.8.0",
		RepositoryFormat: 2,
		SchemaVersion:    1,
		ContractVersion:  8,
	}
	cfg := repositoryConfig{
		RepositoryFormat:  2,
		SchemaVersion:     1,
		ContractVersion:   9,
		RunethreadVersion: "v0.9.0",
	}
	if err := checkNativeCompatibilityForTarget(cfg, target); err == nil || !strings.Contains(err.Error(), "newer") {
		t.Fatalf("newer contract was not rejected: %v", err)
	}
}
