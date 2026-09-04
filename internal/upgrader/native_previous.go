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
	nativeV060ReleaseVersion     = "v0.6.0"
	nativeV070ReleaseVersion     = "v0.7.0"
	previousNativeReleaseVersion = nativeV060ReleaseVersion // compatibility gate retained until the contract-v8 transition
	nativeContractV7SHA256       = "5b245c55e640555c797c3f86f02b54a431da40e959bdd466f90c0c5c88c45766"
)

type nativeSourceAnchor struct {
	ReleaseVersion   string
	RepositoryFormat int
	SchemaVersion     int
	ContractVersion   int
	LockVersion       int
	SourceRepository  string
	ContractSHA256    string
	FilesSHA256       map[string]string
}

var nativeContractV7Files = map[string]string{
	"MEMORY_PROTOCOL.md":             "d5c8c69fb79be4f99c8faab9f9b20b6a1b0dfdf2eac4678fac6e8b2c9dfbf980",
	"docs/EXTENDING_RUNETHREAD.md":   "842a8e9efbc340af52f01b240f2b1c4bd09457354c5de4f33dfbf657cb8acfce",
	"docs/INDEX_FORMAT.md":           "f79eaec6dcea659c85def05cb4050eefce0dc6641079285e172641ba1789cfeb",
	"docs/MEMORY_CONTENT_FORMAT.md":  "d4460812373bdc6448fbb840012820197145f3384008016c749f8b9960a46745",
	"docs/MEMORY_SCHEMA.md":          "42736c5a46fa7f3d81c16d4680f576e51bf694f74655d7ee63a410af6aaa8452",
	"docs/REPOSITORY_VALIDATION.md":  "91782907f7f70e4b189be8d8848bd85cdd6a70fbc2d7c64bd40ebd390b4e3bd3",
	"docs/SOURCES.md":                "93752501cee548a5b3c4863f57190af9dd7c96a026c94b9a54d62bb62840c8d6",
	"docs/TAXONOMY.md":               "69fd1b23366163a2646ec4b8eaf73079a7f48aa7b94d8db283b6fdf5ebe30173",
	"docs/TRUST_MODEL.md":            "ea707b6b315a318bf801cf0af7da00631a9f994ea91f1350be78b966d21462fe",
	"docs/USER_COMMANDS.md":          "fc65b96d58fbb9d4de5a8325c3fe287185ff33ddc32d9223dcb449ea655fecba",
	"schema/memory-item.schema.json": "b4d03cb3fa85e95c98275701ffb44c4a14088bcf0b41cfb6a9dc5a092ab3b69a",
	"templates/correction.md":        "400a57ac853a3e28381235094401fadeebba1bc3e18745295e85e267b080dfc6",
	"templates/decision.md":          "facbf0490a5e138f43a73636a8b8341764bc055d1df03907ed30ebaa04d8038b",
	"templates/fact.md":              "d3d03a2277654875677eb9641f7073beb11e1072f78244135018dc6cdc05c226",
	"templates/milestone.md":         "8c54066877448dc1fea9277bdfb65ed4cb6e51197ad328e52f9215f7c0e96061",
	"templates/open_loop.md":         "35c7a3091de80a0e2258ae33dd72b51390fbbfd92aebe0be55e65a9e04ef1f8d",
	"templates/preference.md":        "f909733d2efd7994a92427328bbe6ba700ca500f358cbae342c357577f81cb63",
	"templates/reference.md":         "9dade8d57fa99a1e3ae156ad6e7677cc06b246f0cd57bcb6ce9de71a346ee307",
	"templates/state.md":             "a83248bd386bbbebec16525be3f687be995bc33b697845a864cd5214baf8c6ce",
}

func nativeSourceAnchorFor(release string) (nativeSourceAnchor, bool) {
	switch release {
	case nativeV060ReleaseVersion, nativeV070ReleaseVersion:
		return nativeSourceAnchor{
			ReleaseVersion:   release,
			RepositoryFormat: 2,
			SchemaVersion:     1,
			ContractVersion:   7,
			LockVersion:       2,
			SourceRepository:  "runethread/core",
			ContractSHA256:    nativeContractV7SHA256,
			FilesSHA256:       nativeContractV7Files,
		}, true
	default:
		return nativeSourceAnchor{}, false
	}
}

func verifyNativeSource(root string, cfg repositoryConfig) error {
	if cfg.RunethreadVersion == buildinfo.ContractReleaseVersion {
		if problems := trust.Check(root); len(problems) != 0 {
			return fmt.Errorf("current native Runethread trust check failed at %s: %s", problems[0].Path, problems[0].Message)
		}
		return nil
	}
	anchor, ok := nativeSourceAnchorFor(cfg.RunethreadVersion)
	if !ok {
		return fmt.Errorf("unsupported native Runethread source release %q", cfg.RunethreadVersion)
	}
	return verifyNativeSourceAnchor(root, anchor)
}

func verifyNativeSourceAnchor(root string, anchor nativeSourceAnchor) error {
	if got := legacyAggregateDigest(anchor.FilesSHA256); got != anchor.ContractSHA256 {
		return fmt.Errorf("internal trusted %s source anchor digest %s does not match declared contract digest %s", anchor.ReleaseVersion, got, anchor.ContractSHA256)
	}

	lockPath := filepath.Join(root, buildinfo.ManagedMetadataDir, "lock.json")
	data, err := readRegularFile(lockPath)
	if err != nil {
		return fmt.Errorf("read native %s trust lock: %w", anchor.ReleaseVersion, err)
	}
	var actual trust.Lock
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&actual); err != nil {
		return fmt.Errorf("parse native %s trust lock: %w", anchor.ReleaseVersion, err)
	}
	if actual.LockVersion != anchor.LockVersion ||
		actual.SourceRepository != anchor.SourceRepository ||
		actual.RunethreadVersion != anchor.ReleaseVersion ||
		actual.RepositoryFormat != anchor.RepositoryFormat ||
		actual.SchemaVersion != anchor.SchemaVersion ||
		actual.ContractVersion != anchor.ContractVersion ||
		actual.ContractSHA256 != anchor.ContractSHA256 {
		return fmt.Errorf("native trust lock is not the exact supported %s source anchor", anchor.ReleaseVersion)
	}
	if len(actual.FilesSHA256) != len(anchor.FilesSHA256) {
		return fmt.Errorf("native trust lock contains %d control-plane paths, expected %d for %s", len(actual.FilesSHA256), len(anchor.FilesSHA256), anchor.ReleaseVersion)
	}
	for _, rel := range sortedStringMapKeys(anchor.FilesSHA256) {
		expectedHash := anchor.FilesSHA256[rel]
		if actual.FilesSHA256[rel] != expectedHash {
			return fmt.Errorf("native trust lock digest for %s does not match trusted %s", rel, anchor.ReleaseVersion)
		}
		local, err := readRegularFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			return fmt.Errorf("verify native %s control-plane file %s: %w", anchor.ReleaseVersion, rel, err)
		}
		if got := sha256Hex(local); got != expectedHash {
			return fmt.Errorf("native control-plane file %s has digest %s, expected %s from trusted %s", rel, got, expectedHash, anchor.ReleaseVersion)
		}
	}
	for rel := range actual.FilesSHA256 {
		if _, ok := anchor.FilesSHA256[rel]; !ok {
			return fmt.Errorf("native trust lock contains unexpected control-plane path %s", rel)
		}
	}
	return nil
}
