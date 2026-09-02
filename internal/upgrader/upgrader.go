package upgrader

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	runethread "github.com/runethread/core"
	"github.com/runethread/core/internal/buildinfo"
	"github.com/runethread/core/internal/indexer"
	"github.com/runethread/core/internal/starter"
	"github.com/runethread/core/internal/trust"
	"github.com/runethread/core/internal/validation"
)

const (
	legacyManagedMetadataDir = ".gitmemo"
	legacyReleaseVersion     = "v0.5.0"
	legacyRepositoryFormat   = 1
	legacySchemaVersion      = 1
	legacyContractVersion    = 6
	legacyTrustLockVersion   = 1
	legacySourceRepository   = "Karageorgiou/GitMemo"
	legacyContractSHA256     = "d1ac047361967e67db3e35de040237c9bb3be55ba16c0b95cac9ee6287e9c67a"
	legacyWorkflowSHA256     = "4e6dbf1e573f01588fd046e1d18cbfa000ecd11b9cb6329caf687612aa1bbb66"
	legacyExtendingPath      = "docs/EXTENDING_GITMEMO.md"
)

type sourceKind int

const (
	sourceNative sourceKind = iota + 1
	sourceLegacyV050
)

type repositoryConfig struct {
	RepositoryFormat  int    `json:"repository_format"`
	SchemaVersion     int    `json:"schema_version"`
	ContractVersion   int    `json:"contract_version"`
	RunethreadVersion string `json:"runethread_version"`
}

type legacyRepositoryConfig struct {
	RepositoryFormat int    `json:"repository_format"`
	SchemaVersion    int    `json:"schema_version"`
	ContractVersion  int    `json:"contract_version"`
	GitMemoVersion   string `json:"gitmemo_version"`
}

type legacyLock struct {
	LockVersion      int               `json:"lock_version"`
	SourceRepository string            `json:"source_repository"`
	GitMemoVersion   string            `json:"gitmemo_version"`
	RepositoryFormat int               `json:"repository_format"`
	SchemaVersion    int               `json:"schema_version"`
	ContractVersion  int               `json:"contract_version"`
	ContractSHA256   string            `json:"contract_sha256"`
	FilesSHA256      map[string]string `json:"files_sha256"`
}

type sourceState struct {
	Kind            sourceKind
	Version         string
	ContractVersion int
}

type Result struct {
	FromVersion    string
	ToVersion      string
	FromContract   int
	ToContract     int
	ChangedPaths   []string
	AlreadyCurrent bool
}

type snapshot struct {
	exists bool
	data   []byte
	mode   fs.FileMode
}

// Apply upgrades a supported memory repository to the Runethread contract
// embedded in the running binary. Native Runethread repositories are updated in
// place. The only legacy source accepted by v0.6.0 is the exact trusted GitMemo
// v0.5.0 format-1 state. User memories and project data are never rewritten by
// this operation.
func Apply(root string) (Result, error) {
	state, err := inspectSource(root)
	if err != nil {
		return Result{}, err
	}
	if err := checkWorkflowOwnership(root, state.Kind); err != nil {
		return Result{}, err
	}

	desired, err := desiredManagedFiles(root, state.Kind)
	if err != nil {
		return Result{}, err
	}
	obsolete := obsoleteManagedPaths(state.Kind)

	existingIndexPaths, err := indexer.ExistingPaths(root)
	if err != nil {
		return Result{}, fmt.Errorf("inspect existing indexes: %w", err)
	}
	generatedIndexPaths, err := indexer.GeneratedPaths(root)
	if err != nil {
		return Result{}, fmt.Errorf("plan regenerated indexes: %w", err)
	}

	trackedSet := map[string]bool{}
	for rel := range desired {
		trackedSet[rel] = true
	}
	for _, rel := range obsolete {
		trackedSet[rel] = true
	}
	for _, rel := range existingIndexPaths {
		trackedSet[rel] = true
	}
	for _, rel := range generatedIndexPaths {
		trackedSet[rel] = true
	}
	tracked := sortedSet(trackedSet)

	snapshots, err := takeSnapshots(root, tracked)
	if err != nil {
		return Result{}, err
	}

	changed := make([]string, 0, len(tracked))
	fail := func(cause error) (Result, error) {
		rollbackErr := rollback(root, snapshots, state.Kind)
		if rollbackErr != nil {
			return Result{}, fmt.Errorf("%v; rollback also failed: %w", cause, rollbackErr)
		}
		return Result{}, cause
	}

	for _, rel := range sortedByteMapKeys(desired) {
		data := desired[rel]
		old := snapshots[rel]
		if old.exists && bytes.Equal(old.data, data) {
			continue
		}
		if err := atomicWrite(root, rel, data, 0o644); err != nil {
			return fail(err)
		}
		changed = append(changed, rel)
	}

	for _, rel := range obsolete {
		if !snapshots[rel].exists {
			continue
		}
		if err := os.Remove(filepath.Join(root, filepath.FromSlash(rel))); err != nil {
			return fail(fmt.Errorf("remove obsolete managed path %s: %w", rel, err))
		}
		changed = append(changed, rel)
	}
	if state.Kind == sourceLegacyV050 {
		if err := os.Remove(filepath.Join(root, legacyManagedMetadataDir)); err != nil {
			return fail(fmt.Errorf("remove obsolete %s directory: %w", legacyManagedMetadataDir, err))
		}
	}

	if err := indexer.Write(root); err != nil {
		return fail(fmt.Errorf("regenerate indexes during upgrade: %w", err))
	}
	currentIndexPaths, err := indexer.ExistingPaths(root)
	if err != nil {
		return fail(fmt.Errorf("inspect regenerated indexes: %w", err))
	}
	indexUnion := map[string]bool{}
	for _, rel := range existingIndexPaths {
		indexUnion[rel] = true
	}
	for _, rel := range currentIndexPaths {
		indexUnion[rel] = true
	}
	for _, rel := range sortedSet(indexUnion) {
		old := snapshots[rel]
		data, readErr := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if os.IsNotExist(readErr) {
			if old.exists {
				changed = append(changed, rel)
			}
			continue
		}
		if readErr != nil {
			return fail(fmt.Errorf("read regenerated %s: %w", rel, readErr))
		}
		if !old.exists || !bytes.Equal(old.data, data) {
			changed = append(changed, rel)
		}
	}

	if err := verifyTargetMetadata(root); err != nil {
		return fail(err)
	}
	issues := validation.Validate(root)
	if validation.HasErrors(issues) {
		return fail(fmt.Errorf("upgraded repository failed validation: %s", validation.RenderText(issues)))
	}

	sort.Strings(changed)
	return Result{
		FromVersion:    state.Version,
		ToVersion:      buildinfo.ReleaseVersion,
		FromContract:   state.ContractVersion,
		ToContract:     buildinfo.ContractVersion,
		ChangedPaths:   unique(changed),
		AlreadyCurrent: len(changed) == 0,
	}, nil
}

func inspectSource(root string) (sourceState, error) {
	nativeDir := filepath.Join(root, buildinfo.ManagedMetadataDir)
	legacyDir := filepath.Join(root, legacyManagedMetadataDir)
	nativeExists, err := pathExists(nativeDir)
	if err != nil {
		return sourceState{}, err
	}
	legacyExists, err := pathExists(legacyDir)
	if err != nil {
		return sourceState{}, err
	}
	if nativeExists && legacyExists {
		return sourceState{}, fmt.Errorf("mixed managed metadata: both %s and %s exist; refusing ambiguous migration", buildinfo.ManagedMetadataDir, legacyManagedMetadataDir)
	}
	if nativeExists {
		cfg, err := readNativeConfig(root)
		if err != nil {
			return sourceState{}, err
		}
		if err := checkNativeCompatibility(cfg); err != nil {
			return sourceState{}, err
		}
		return sourceState{Kind: sourceNative, Version: cfg.RunethreadVersion, ContractVersion: cfg.ContractVersion}, nil
	}
	if legacyExists {
		if err := verifyLegacyV050Source(root); err != nil {
			return sourceState{}, err
		}
		return sourceState{Kind: sourceLegacyV050, Version: legacyReleaseVersion, ContractVersion: legacyContractVersion}, nil
	}
	return sourceState{}, fmt.Errorf("no supported managed metadata found; expected %s or trusted GitMemo v0.5.0 %s", buildinfo.ManagedMetadataDir, legacyManagedMetadataDir)
}

func readNativeConfig(root string) (repositoryConfig, error) {
	path := filepath.Join(root, buildinfo.ManagedMetadataDir, "config.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return repositoryConfig{}, fmt.Errorf("read Runethread repository config: %w", err)
	}
	var cfg repositoryConfig
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&cfg); err != nil {
		return repositoryConfig{}, fmt.Errorf("parse Runethread repository config: %w", err)
	}
	return cfg, nil
}

func checkNativeCompatibility(cfg repositoryConfig) error {
	if cfg.RepositoryFormat != buildinfo.RepositoryFormatVersion {
		return fmt.Errorf("repository format %d is not supported by %s (supports %d)", cfg.RepositoryFormat, buildinfo.ReleaseVersion, buildinfo.RepositoryFormatVersion)
	}
	if cfg.SchemaVersion != buildinfo.SchemaVersion {
		if cfg.SchemaVersion > buildinfo.SchemaVersion {
			return fmt.Errorf("repository schema version %d is newer than %s supports (%d)", cfg.SchemaVersion, buildinfo.ReleaseVersion, buildinfo.SchemaVersion)
		}
		return fmt.Errorf("no Runethread schema migration from version %d to %d is implemented", cfg.SchemaVersion, buildinfo.SchemaVersion)
	}
	if cfg.ContractVersion != buildinfo.ContractVersion {
		if cfg.ContractVersion > buildinfo.ContractVersion {
			return fmt.Errorf("repository contract version %d is newer than %s supports (%d)", cfg.ContractVersion, buildinfo.ReleaseVersion, buildinfo.ContractVersion)
		}
		return fmt.Errorf("no native Runethread contract migration from version %d to %d is implemented", cfg.ContractVersion, buildinfo.ContractVersion)
	}
	if cfg.RunethreadVersion != buildinfo.ReleaseVersion {
		return fmt.Errorf("repository pins Runethread %q; %s only upgrades the exact native %s state or trusted GitMemo v0.5.0", cfg.RunethreadVersion, buildinfo.ReleaseVersion, buildinfo.ReleaseVersion)
	}
	return nil
}

func verifyLegacyV050Source(root string) error {
	dir := filepath.Join(root, legacyManagedMetadataDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read legacy managed metadata: %w", err)
	}
	if len(entries) != 2 || entries[0].Name() != "config.json" || entries[1].Name() != "lock.json" || entries[0].IsDir() || entries[1].IsDir() {
		return fmt.Errorf("legacy %s directory is not the exact supported v0.5.0 layout", legacyManagedMetadataDir)
	}

	configData, err := readRegularFile(filepath.Join(dir, "config.json"))
	if err != nil {
		return fmt.Errorf("read legacy repository config: %w", err)
	}
	var cfg legacyRepositoryConfig
	dec := json.NewDecoder(bytes.NewReader(configData))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&cfg); err != nil {
		return fmt.Errorf("parse legacy repository config: %w", err)
	}
	if cfg.RepositoryFormat != legacyRepositoryFormat || cfg.SchemaVersion != legacySchemaVersion || cfg.ContractVersion != legacyContractVersion || cfg.GitMemoVersion != legacyReleaseVersion {
		return fmt.Errorf("legacy repository config is not the supported GitMemo v0.5.0 source state")
	}

	lockData, err := readRegularFile(filepath.Join(dir, "lock.json"))
	if err != nil {
		return fmt.Errorf("read legacy trust lock: %w", err)
	}
	var lock legacyLock
	dec = json.NewDecoder(bytes.NewReader(lockData))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&lock); err != nil {
		return fmt.Errorf("parse legacy trust lock: %w", err)
	}
	if lock.LockVersion != legacyTrustLockVersion || lock.SourceRepository != legacySourceRepository || lock.GitMemoVersion != legacyReleaseVersion || lock.RepositoryFormat != legacyRepositoryFormat || lock.SchemaVersion != legacySchemaVersion || lock.ContractVersion != legacyContractVersion || lock.ContractSHA256 != legacyContractSHA256 {
		return fmt.Errorf("legacy trust lock is not the supported GitMemo v0.5.0 trust anchor")
	}
	if len(lock.FilesSHA256) != 19 || legacyAggregateDigest(lock.FilesSHA256) != legacyContractSHA256 {
		return fmt.Errorf("legacy trust lock contract digest does not match the supported GitMemo v0.5.0 contract")
	}
	for _, rel := range sortedStringMapKeys(lock.FilesSHA256) {
		data, err := readRegularFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			return fmt.Errorf("verify legacy control-plane file %s: %w", rel, err)
		}
		if got := sha256Hex(data); got != lock.FilesSHA256[rel] {
			return fmt.Errorf("legacy control-plane file %s has digest %s, expected %s from trusted GitMemo v0.5.0 lock", rel, got, lock.FilesSHA256[rel])
		}
	}
	return nil
}

func desiredManagedFiles(root string, kind sourceKind) (map[string][]byte, error) {
	desired := make(map[string][]byte, len(runethread.ContractPaths())+4)
	for _, rel := range runethread.ContractPaths() {
		data, err := fs.ReadFile(runethread.ContractFS, rel)
		if err != nil {
			return nil, fmt.Errorf("read embedded contract %s: %w", rel, err)
		}
		desired[rel] = data
	}
	cfg, err := starter.ConfigJSON()
	if err != nil {
		return nil, fmt.Errorf("render repository config: %w", err)
	}
	desired[buildinfo.ManagedMetadataDir+"/config.json"] = cfg
	lock, err := trust.JSON()
	if err != nil {
		return nil, fmt.Errorf("render trust lock: %w", err)
	}
	desired[buildinfo.ManagedMetadataDir+"/lock.json"] = lock
	desired[".github/workflows/validate.yml"] = starter.ValidationWorkflow()

	readmePath := filepath.Join(root, "README.md")
	if data, err := os.ReadFile(readmePath); err == nil {
		updated := updateReadme(data, kind)
		if !bytes.Equal(data, updated) {
			desired["README.md"] = updated
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("read README.md: %w", err)
	}
	return desired, nil
}

func updateReadme(data []byte, kind sourceKind) []byte {
	text := string(data)
	switch kind {
	case sourceLegacyV050:
		if strings.HasPrefix(text, "# GitMemo Memory\n") && strings.Contains(text, ".gitmemo/lock.json") {
			return starter.MemoryRepoReadme()
		}
	case sourceNative:
		if strings.HasPrefix(text, "# Runethread Memory\n") && strings.Contains(text, ".runethread/lock.json") {
			return starter.MemoryRepoReadme()
		}
	}
	return data
}

func obsoleteManagedPaths(kind sourceKind) []string {
	if kind != sourceLegacyV050 {
		return nil
	}
	return []string{
		legacyManagedMetadataDir + "/config.json",
		legacyManagedMetadataDir + "/lock.json",
		legacyExtendingPath,
	}
}

func checkWorkflowOwnership(root string, kind sourceKind) error {
	path := filepath.Join(root, ".github", "workflows", "validate.yml")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read validation workflow: %w", err)
	}
	switch kind {
	case sourceNative:
		if bytes.Equal(data, starter.ValidationWorkflow()) {
			return nil
		}
	case sourceLegacyV050:
		if sha256Hex(data) == legacyWorkflowSHA256 {
			return nil
		}
	}
	return fmt.Errorf(".github/workflows/validate.yml does not exactly match the managed workflow for the detected source state; refusing to overwrite a custom workflow")
}

func verifyTargetMetadata(root string) error {
	if exists, err := pathExists(filepath.Join(root, legacyManagedMetadataDir)); err != nil {
		return err
	} else if exists {
		return fmt.Errorf("legacy managed metadata %s remains after migration", legacyManagedMetadataDir)
	}
	for _, rel := range []string{buildinfo.ManagedMetadataDir + "/config.json", buildinfo.ManagedMetadataDir + "/lock.json"} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); err != nil {
			return fmt.Errorf("target managed path %s is missing: %w", rel, err)
		}
	}
	return nil
}

func pathExists(path string) (bool, error) {
	_, err := os.Lstat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, fmt.Errorf("inspect %s: %w", path, err)
}

func readRegularFile(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("expected regular file, found mode %s", info.Mode())
	}
	return os.ReadFile(path)
}

func legacyAggregateDigest(files map[string]string) string {
	h := sha256.New()
	for _, rel := range sortedStringMapKeys(files) {
		_, _ = h.Write([]byte(rel))
		_, _ = h.Write([]byte{0})
		_, _ = h.Write([]byte(files[rel]))
		_, _ = h.Write([]byte{'\n'})
	}
	return hex.EncodeToString(h.Sum(nil))
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func sortedStringMapKeys(values map[string]string) []string {
	out := make([]string, 0, len(values))
	for key := range values {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func sortedByteMapKeys(values map[string][]byte) []string {
	out := make([]string, 0, len(values))
	for key := range values {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func takeSnapshots(root string, paths []string) (map[string]snapshot, error) {
	out := make(map[string]snapshot, len(paths))
	for _, rel := range paths {
		path := filepath.Join(root, filepath.FromSlash(rel))
		info, err := os.Stat(path)
		if os.IsNotExist(err) {
			out[rel] = snapshot{}
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("inspect %s: %w", rel, err)
		}
		if info.IsDir() {
			return nil, fmt.Errorf("managed path %s is unexpectedly a directory", rel)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", rel, err)
		}
		out[rel] = snapshot{exists: true, data: data, mode: info.Mode().Perm()}
	}
	return out, nil
}

func rollback(root string, snapshots map[string]snapshot, kind sourceKind) error {
	err := restoreSnapshots(root, snapshots)
	if kind != sourceLegacyV050 {
		return err
	}
	nativeDir := filepath.Join(root, buildinfo.ManagedMetadataDir)
	if removeErr := os.Remove(nativeDir); removeErr != nil && !os.IsNotExist(removeErr) {
		if err != nil {
			return fmt.Errorf("%v; remove rollback directory %s: %w", err, buildinfo.ManagedMetadataDir, removeErr)
		}
		return fmt.Errorf("remove rollback directory %s: %w", buildinfo.ManagedMetadataDir, removeErr)
	}
	return err
}

func restoreSnapshots(root string, snapshots map[string]snapshot) error {
	var failures []string
	for rel, snap := range snapshots {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if !snap.exists {
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				failures = append(failures, fmt.Sprintf("remove %s: %v", rel, err))
			}
			continue
		}
		mode := snap.mode
		if mode == 0 {
			mode = 0o644
		}
		if err := atomicWrite(root, rel, snap.data, mode); err != nil {
			failures = append(failures, err.Error())
		}
	}
	if len(failures) > 0 {
		sort.Strings(failures)
		return fmt.Errorf("%s", strings.Join(failures, "; "))
	}
	return nil
}

func atomicWrite(root, rel string, data []byte, mode fs.FileMode) error {
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create parent for %s: %w", rel, err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".runethread-upgrade-*")
	if err != nil {
		return fmt.Errorf("create temporary file for %s: %w", rel, err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write temporary file for %s: %w", rel, err)
	}
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return fmt.Errorf("chmod temporary file for %s: %w", rel, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temporary file for %s: %w", rel, err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("replace %s: %w", rel, err)
	}
	return nil
}

func sortedSet(values map[string]bool) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func unique(values []string) []string {
	if len(values) < 2 {
		return values
	}
	out := values[:0]
	var prev string
	for i, value := range values {
		if i == 0 || value != prev {
			out = append(out, value)
		}
		prev = value
	}
	return out
}
