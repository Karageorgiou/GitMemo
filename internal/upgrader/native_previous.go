package upgrader

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/runethread/core/internal/buildinfo"
	"github.com/runethread/core/internal/trust"
)

const (
	previousNativeReleaseVersion = "v0.6.0"
	previousNativeContractSHA256 = "5b245c55e640555c797c3f86f02b54a431da40e959bdd466f90c0c5c88c45766"
)

func verifyNativeSource(root string, cfg repositoryConfig) error {
	switch cfg.RunethreadVersion {
	case buildinfo.ReleaseVersion:
		if problems := trust.Check(root); len(problems) != 0 {
			return fmt.Errorf("current native Runethread trust check failed at %s: %s", problems[0].Path, problems[0].Message)
		}
		return nil
	case previousNativeReleaseVersion:
		return verifyPreviousNativeSource(root)
	default:
		return fmt.Errorf("unsupported native Runethread source release %q", cfg.RunethreadVersion)
	}
}

func verifyPreviousNativeSource(root string) error {
	expected, err := trust.ExpectedLock()
	if err != nil {
		return fmt.Errorf("build current trust anchor: %w", err)
	}
	if expected.ContractSHA256 != previousNativeContractSHA256 {
		return fmt.Errorf("embedded %s contract digest %s differs from the trusted %s compatible digest %s; explicit contract migration is required", buildinfo.ReleaseVersion, expected.ContractSHA256, previousNativeReleaseVersion, previousNativeContractSHA256)
	}
	expected.RunethreadVersion = previousNativeReleaseVersion

	lockPath := filepath.Join(root, buildinfo.ManagedMetadataDir, "lock.json")
	data, err := readRegularFile(lockPath)
	if err != nil {
		return fmt.Errorf("read previous native trust lock: %w", err)
	}
	var actual trust.Lock
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&actual); err != nil {
		return fmt.Errorf("parse previous native trust lock: %w", err)
	}
	if actual.LockVersion != expected.LockVersion ||
		actual.SourceRepository != expected.SourceRepository ||
		actual.RunethreadVersion != expected.RunethreadVersion ||
		actual.RepositoryFormat != expected.RepositoryFormat ||
		actual.SchemaVersion != expected.SchemaVersion ||
		actual.ContractVersion != expected.ContractVersion ||
		actual.ContractSHA256 != expected.ContractSHA256 {
		return fmt.Errorf("native trust lock is not the exact supported %s source anchor", previousNativeReleaseVersion)
	}
	if len(actual.FilesSHA256) != len(expected.FilesSHA256) {
		return fmt.Errorf("native trust lock contains %d control-plane paths, expected %d for %s", len(actual.FilesSHA256), len(expected.FilesSHA256), previousNativeReleaseVersion)
	}
	for _, rel := range sortedStringMapKeys(expected.FilesSHA256) {
		expectedHash := expected.FilesSHA256[rel]
		if actual.FilesSHA256[rel] != expectedHash {
			return fmt.Errorf("native trust lock digest for %s does not match trusted %s", rel, previousNativeReleaseVersion)
		}
		local, err := readRegularFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			return fmt.Errorf("verify previous native control-plane file %s: %w", rel, err)
		}
		if got := sha256Hex(local); got != expectedHash {
			return fmt.Errorf("native control-plane file %s has digest %s, expected %s from trusted %s", rel, got, expectedHash, previousNativeReleaseVersion)
		}
	}
	for rel := range actual.FilesSHA256 {
		if _, ok := expected.FilesSHA256[rel]; !ok {
			return fmt.Errorf("native trust lock contains unexpected control-plane path %s", rel)
		}
	}
	return nil
}
