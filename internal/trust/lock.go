package trust

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"

	runethread "github.com/runethread/core"
	"github.com/runethread/core/internal/buildinfo"
)

var stableVersionRE = regexp.MustCompile(`^v([0-9]+)\.([0-9]+)\.([0-9]+)$`)

type Lock struct {
	LockVersion       int               `json:"lock_version"`
	SourceRepository  string            `json:"source_repository"`
	RunethreadVersion string            `json:"runethread_version"`
	RepositoryFormat  int               `json:"repository_format"`
	SchemaVersion     int               `json:"schema_version"`
	ContractVersion   int               `json:"contract_version"`
	ContractSHA256    string            `json:"contract_sha256"`
	FilesSHA256       map[string]string `json:"files_sha256"`
}

type Problem struct {
	Path    string
	Message string
}

type repositoryConfig struct {
	RepositoryFormat  int    `json:"repository_format"`
	SchemaVersion     int    `json:"schema_version"`
	ContractVersion   int    `json:"contract_version"`
	RunethreadVersion string `json:"runethread_version"`
}

// ExpectedLock builds the trust lock for the operational contract embedded in
// the running Runethread release.
func ExpectedLock() (Lock, error) {
	files := make(map[string]string, len(runethread.ContractPaths()))
	for _, rel := range runethread.ContractPaths() {
		data, err := fs.ReadFile(runethread.ContractFS, rel)
		if err != nil {
			return Lock{}, fmt.Errorf("read embedded contract %s: %w", rel, err)
		}
		files[rel] = sha256Hex(data)
	}
	return Lock{
		LockVersion:       buildinfo.TrustLockVersion,
		SourceRepository:  buildinfo.SourceRepository,
		RunethreadVersion: buildinfo.ContractReleaseVersion,
		RepositoryFormat:  buildinfo.RepositoryFormatVersion,
		SchemaVersion:     buildinfo.SchemaVersion,
		ContractVersion:   buildinfo.ContractVersion,
		ContractSHA256:    aggregateDigest(files),
		FilesSHA256:       files,
	}, nil
}

func JSON() ([]byte, error) {
	lock, err := ExpectedLock()
	if err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(lock, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

// ReadPinnedVersion is intentionally forward-tolerant. The stable validation
// bootstrap only needs the lock envelope version and pinned control-plane release.
// Future lock files may add fields without requiring the bootstrap itself to
// change.
func ReadPinnedVersion(root string) (string, error) {
	path := filepath.Join(root, buildinfo.ManagedMetadataDir, "lock.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read trust lock: %w", err)
	}
	var envelope struct {
		LockVersion       int    `json:"lock_version"`
		RunethreadVersion string `json:"runethread_version"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return "", fmt.Errorf("parse trust lock: %w", err)
	}
	if envelope.LockVersion < 1 {
		return "", fmt.Errorf("invalid trust lock version %d", envelope.LockVersion)
	}
	if !isSupportedPinnedVersion(envelope.RunethreadVersion) {
		return "", fmt.Errorf("trust lock pins unsupported release %q; stable bootstrap requires v0.6.0 or newer", envelope.RunethreadVersion)
	}
	return envelope.RunethreadVersion, nil
}

// Check verifies that the repository's lock, config, and vendored control-plane
// files exactly match the contract embedded in the running release.
func Check(root string) []Problem {
	expected, err := ExpectedLock()
	lockRel := buildinfo.ManagedMetadataDir + "/lock.json"
	if err != nil {
		return []Problem{{Path: lockRel, Message: err.Error()}}
	}

	lockPath := filepath.Join(root, buildinfo.ManagedMetadataDir, "lock.json")
	data, err := os.ReadFile(lockPath)
	if err != nil {
		return []Problem{{Path: lockRel, Message: fmt.Sprintf("read trust lock: %v", err)}}
	}
	var actual Lock
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&actual); err != nil {
		return []Problem{{Path: lockRel, Message: fmt.Sprintf("parse trust lock: %v", err)}}
	}

	var problems []Problem
	addLock := func(message string) {
		problems = append(problems, Problem{Path: lockRel, Message: message})
	}
	if actual.LockVersion != expected.LockVersion {
		addLock(fmt.Sprintf("lock_version is %d, expected %d for %s", actual.LockVersion, expected.LockVersion, buildinfo.ReleaseVersion))
	}
	if actual.SourceRepository != expected.SourceRepository {
		addLock(fmt.Sprintf("source_repository is %q, expected %q", actual.SourceRepository, expected.SourceRepository))
	}
	if actual.RunethreadVersion != expected.RunethreadVersion {
		addLock(fmt.Sprintf("runethread_version is %q, expected pinned contract release %q (running release %q)", actual.RunethreadVersion, expected.RunethreadVersion, buildinfo.ReleaseVersion))
	}
	if actual.RepositoryFormat != expected.RepositoryFormat || actual.SchemaVersion != expected.SchemaVersion || actual.ContractVersion != expected.ContractVersion {
		addLock(fmt.Sprintf("repository/schema/contract versions are %d/%d/%d, expected %d/%d/%d", actual.RepositoryFormat, actual.SchemaVersion, actual.ContractVersion, expected.RepositoryFormat, expected.SchemaVersion, expected.ContractVersion))
	}
	if actual.ContractSHA256 != expected.ContractSHA256 {
		addLock(fmt.Sprintf("contract_sha256 is %q, expected %q", actual.ContractSHA256, expected.ContractSHA256))
	}
	if len(actual.FilesSHA256) != len(expected.FilesSHA256) {
		addLock(fmt.Sprintf("files_sha256 contains %d paths, expected %d", len(actual.FilesSHA256), len(expected.FilesSHA256)))
	}

	for _, rel := range sortedPaths(expected.FilesSHA256) {
		expectedHash := expected.FilesSHA256[rel]
		lockedHash, ok := actual.FilesSHA256[rel]
		if !ok {
			addLock("missing control-plane digest for " + rel)
			continue
		}
		if lockedHash != expectedHash {
			addLock(fmt.Sprintf("locked digest for %s is %s, expected %s", rel, lockedHash, expectedHash))
			continue
		}
		local, readErr := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if readErr != nil {
			problems = append(problems, Problem{Path: rel, Message: fmt.Sprintf("read control-plane file: %v", readErr)})
			continue
		}
		if localHash := sha256Hex(local); localHash != expectedHash {
			problems = append(problems, Problem{Path: rel, Message: fmt.Sprintf("control-plane digest is %s, expected %s from pinned release %s", localHash, expectedHash, expected.RunethreadVersion)})
		}
	}
	for rel := range actual.FilesSHA256 {
		if _, ok := expected.FilesSHA256[rel]; !ok {
			addLock("unexpected control-plane digest path " + rel)
		}
	}

	problems = append(problems, checkConfig(root, expected)...)
	return problems
}

func checkConfig(root string, expected Lock) []Problem {
	rel := buildinfo.ManagedMetadataDir + "/config.json"
	path := filepath.Join(root, buildinfo.ManagedMetadataDir, "config.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return []Problem{{Path: rel, Message: fmt.Sprintf("read repository config: %v", err)}}
	}
	var cfg repositoryConfig
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&cfg); err != nil {
		return []Problem{{Path: rel, Message: fmt.Sprintf("parse repository config: %v", err)}}
	}
	if cfg.RepositoryFormat != expected.RepositoryFormat || cfg.SchemaVersion != expected.SchemaVersion || cfg.ContractVersion != expected.ContractVersion || cfg.RunethreadVersion != expected.RunethreadVersion {
		return []Problem{{Path: rel, Message: fmt.Sprintf("config pins %d/%d/%d/%s, expected %d/%d/%d/%s", cfg.RepositoryFormat, cfg.SchemaVersion, cfg.ContractVersion, cfg.RunethreadVersion, expected.RepositoryFormat, expected.SchemaVersion, expected.ContractVersion, expected.RunethreadVersion)}}
	}
	return nil
}

func aggregateDigest(files map[string]string) string {
	h := sha256.New()
	for _, rel := range sortedPaths(files) {
		_, _ = h.Write([]byte(rel))
		_, _ = h.Write([]byte{0})
		_, _ = h.Write([]byte(files[rel]))
		_, _ = h.Write([]byte{'\n'})
	}
	return hex.EncodeToString(h.Sum(nil))
}

func sortedPaths(files map[string]string) []string {
	paths := make([]string, 0, len(files))
	for rel := range files {
		paths = append(paths, rel)
	}
	sort.Strings(paths)
	return paths
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func isSupportedPinnedVersion(version string) bool {
	m := stableVersionRE.FindStringSubmatch(version)
	if m == nil {
		return false
	}
	major, _ := strconv.Atoi(m[1])
	minor, _ := strconv.Atoi(m[2])
	patch, _ := strconv.Atoi(m[3])
	if major > 0 {
		return true
	}
	if minor > 6 {
		return true
	}
	return minor == 6 && patch >= 0
}
