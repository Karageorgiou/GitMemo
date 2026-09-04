package upgrader

import (
	"fmt"

	"github.com/runethread/core/internal/buildinfo"
)

type nativeCompatibilityTarget struct {
	RuntimeRelease   string
	ContractRelease  string
	RepositoryFormat int
	SchemaVersion    int
	ContractVersion  int
}

func currentNativeCompatibilityTarget() nativeCompatibilityTarget {
	return nativeCompatibilityTarget{
		RuntimeRelease:   buildinfo.ReleaseVersion,
		ContractRelease:  buildinfo.ContractReleaseVersion,
		RepositoryFormat: buildinfo.RepositoryFormatVersion,
		SchemaVersion:    buildinfo.SchemaVersion,
		ContractVersion:  buildinfo.ContractVersion,
	}
}

func checkNativeCompatibilityForTarget(cfg repositoryConfig, target nativeCompatibilityTarget) error {
	if cfg.RepositoryFormat != target.RepositoryFormat {
		return fmt.Errorf("repository format %d is not supported by runtime %s (supports %d)", cfg.RepositoryFormat, target.RuntimeRelease, target.RepositoryFormat)
	}
	if cfg.SchemaVersion != target.SchemaVersion {
		if cfg.SchemaVersion > target.SchemaVersion {
			return fmt.Errorf("repository schema version %d is newer than runtime %s supports (%d)", cfg.SchemaVersion, target.RuntimeRelease, target.SchemaVersion)
		}
		return fmt.Errorf("no Runethread schema migration from version %d to %d is implemented", cfg.SchemaVersion, target.SchemaVersion)
	}
	if cfg.ContractVersion > target.ContractVersion {
		return fmt.Errorf("repository contract version %d is newer than runtime %s supports (%d)", cfg.ContractVersion, target.RuntimeRelease, target.ContractVersion)
	}

	if cfg.ContractVersion == target.ContractVersion && cfg.RunethreadVersion == target.ContractRelease {
		return nil
	}

	anchor, ok := nativeSourceAnchorFor(cfg.RunethreadVersion)
	if ok && anchor.RepositoryFormat == cfg.RepositoryFormat && anchor.SchemaVersion == cfg.SchemaVersion && anchor.ContractVersion == cfg.ContractVersion {
		return nil
	}

	if cfg.ContractVersion < target.ContractVersion {
		return fmt.Errorf("no trusted native Runethread migration from contract %d / release %q to contract %d / contract release %q is implemented", cfg.ContractVersion, cfg.RunethreadVersion, target.ContractVersion, target.ContractRelease)
	}
	return fmt.Errorf("repository contract %d pins Runethread %q; runtime %s expects contract release %q", cfg.ContractVersion, cfg.RunethreadVersion, target.RuntimeRelease, target.ContractRelease)
}
