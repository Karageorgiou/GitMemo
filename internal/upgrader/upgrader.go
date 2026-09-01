package upgrader

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	gitmemo "github.com/Karageorgiou/GitMemo"
	"github.com/Karageorgiou/GitMemo/internal/buildinfo"
	"github.com/Karageorgiou/GitMemo/internal/indexer"
	"github.com/Karageorgiou/GitMemo/internal/starter"
	"github.com/Karageorgiou/GitMemo/internal/trust"
	"github.com/Karageorgiou/GitMemo/internal/validation"
)

type repositoryConfig struct {
	RepositoryFormat int    `json:"repository_format"`
	SchemaVersion    int    `json:"schema_version"`
	ContractVersion  int    `json:"contract_version"`
	GitMemoVersion   string `json:"gitmemo_version,omitempty"`
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

// Apply upgrades a GitMemo memory repository to the repository contract embedded
// in the running GitMemo binary. User memories and project data are never
// rewritten by this operation.
func Apply(root string) (Result, error) {
	cfg, err := readConfig(root)
	if err != nil {
		return Result{}, err
	}
	if err := checkCompatibility(cfg); err != nil {
		return Result{}, err
	}

	fromVersion := cfg.GitMemoVersion
	if fromVersion == "" {
		fromVersion = inferVersion(root)
	}

	desired, err := desiredManagedFiles(root, fromVersion)
	if err != nil {
		return Result{}, err
	}
	if err := checkWorkflowOwnership(root); err != nil {
		return Result{}, err
	}

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
	for rel, data := range desired {
		old := snapshots[rel]
		if old.exists && bytes.Equal(old.data, data) {
			continue
		}
		if err := atomicWrite(root, rel, data, 0o644); err != nil {
			_ = restoreSnapshots(root, snapshots)
			return Result{}, err
		}
		changed = append(changed, rel)
	}

	if err := indexer.Write(root); err != nil {
		_ = restoreSnapshots(root, snapshots)
		return Result{}, fmt.Errorf("regenerate indexes during upgrade: %w", err)
	}
	currentIndexPaths, err := indexer.ExistingPaths(root)
	if err != nil {
		_ = restoreSnapshots(root, snapshots)
		return Result{}, fmt.Errorf("inspect regenerated indexes: %w", err)
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
			_ = restoreSnapshots(root, snapshots)
			return Result{}, fmt.Errorf("read regenerated %s: %w", rel, readErr)
		}
		if !old.exists || !bytes.Equal(old.data, data) {
			changed = append(changed, rel)
		}
	}

	issues := validation.Validate(root)
	if validation.HasErrors(issues) {
		rollbackErr := restoreSnapshots(root, snapshots)
		if rollbackErr != nil {
			return Result{}, fmt.Errorf("upgraded repository failed validation and rollback also failed: %s; rollback: %v", validation.RenderText(issues), rollbackErr)
		}
		return Result{}, fmt.Errorf("upgraded repository failed validation; changes rolled back: %s", validation.RenderText(issues))
	}

	sort.Strings(changed)
	return Result{
		FromVersion:    fromVersion,
		ToVersion:      buildinfo.ReleaseVersion,
		FromContract:   cfg.ContractVersion,
		ToContract:     buildinfo.ContractVersion,
		ChangedPaths:   unique(changed),
		AlreadyCurrent: len(changed) == 0,
	}, nil
}

func readConfig(root string) (repositoryConfig, error) {
	path := filepath.Join(root, ".gitmemo", "config.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return repositoryConfig{}, fmt.Errorf("read GitMemo repository config: %w", err)
	}
	var cfg repositoryConfig
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&cfg); err != nil {
		return repositoryConfig{}, fmt.Errorf("parse GitMemo repository config: %w", err)
	}
	return cfg, nil
}

func checkCompatibility(cfg repositoryConfig) error {
	if cfg.RepositoryFormat != buildinfo.RepositoryFormatVersion {
		return fmt.Errorf("repository format %d is not supported by %s (supports %d)", cfg.RepositoryFormat, buildinfo.ReleaseVersion, buildinfo.RepositoryFormatVersion)
	}
	if cfg.SchemaVersion > buildinfo.SchemaVersion {
		return fmt.Errorf("repository schema version %d is newer than %s supports (%d)", cfg.SchemaVersion, buildinfo.ReleaseVersion, buildinfo.SchemaVersion)
	}
	if cfg.SchemaVersion < buildinfo.SchemaVersion {
		return fmt.Errorf("no schema migration from version %d to %d is implemented", cfg.SchemaVersion, buildinfo.SchemaVersion)
	}
	if cfg.ContractVersion > buildinfo.ContractVersion {
		return fmt.Errorf("repository contract version %d is newer than %s supports (%d)", cfg.ContractVersion, buildinfo.ReleaseVersion, buildinfo.ContractVersion)
	}
	if cfg.ContractVersion < 1 {
		return fmt.Errorf("invalid repository contract version %d", cfg.ContractVersion)
	}
	return nil
}

func desiredManagedFiles(root, fromVersion string) (map[string][]byte, error) {
	desired := make(map[string][]byte, len(gitmemo.ContractPaths())+4)
	for _, rel := range gitmemo.ContractPaths() {
		data, err := fs.ReadFile(gitmemo.ContractFS, rel)
		if err != nil {
			return nil, fmt.Errorf("read embedded contract %s: %w", rel, err)
		}
		desired[rel] = data
	}
	cfg, err := starter.ConfigJSON()
	if err != nil {
		return nil, fmt.Errorf("render repository config: %w", err)
	}
	desired[".gitmemo/config.json"] = cfg
	lock, err := trust.JSON()
	if err != nil {
		return nil, fmt.Errorf("render trust lock: %w", err)
	}
	desired[".gitmemo/lock.json"] = lock
	desired[".github/workflows/validate.yml"] = starter.ValidationWorkflow()

	readmePath := filepath.Join(root, "README.md")
	if data, err := os.ReadFile(readmePath); err == nil {
		updated := updateReadme(data, fromVersion)
		if !bytes.Equal(data, updated) {
			desired["README.md"] = updated
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("read README.md: %w", err)
	}
	return desired, nil
}

func updateReadme(data []byte, fromVersion string) []byte {
	text := string(data)
	if strings.HasPrefix(text, "# GitMemo Memory\n") && (strings.Contains(text, "GitMemo: store") || strings.Contains(text, "GitMemo: remember")) {
		return starter.MemoryRepoReadme()
	}
	text = strings.ReplaceAll(text, "`GitMemo: remember ...` — explicit durable memory write.", "`GitMemo: store ...` — explicit durable memory write.")
	text = strings.ReplaceAll(text, "user-facing remember/search command contract", "user-facing store/search command contract")
	if fromVersion != "" && fromVersion != "unversioned" {
		text = strings.ReplaceAll(text, "pinned to GitMemo "+fromVersion, "pinned to GitMemo "+buildinfo.ReleaseVersion)
	}
	return []byte(text)
}

func checkWorkflowOwnership(root string) error {
	path := filepath.Join(root, ".github", "workflows", "validate.yml")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read validation workflow: %w", err)
	}
	text := string(data)
	if strings.Contains(text, "# Managed by GitMemo.") {
		return nil
	}
	if strings.Contains(text, "name: Validate GitMemo Memory") && strings.Contains(text, "github.com/Karageorgiou/GitMemo/cmd/gitmemo@") {
		return nil
	}
	return fmt.Errorf(".github/workflows/validate.yml does not look GitMemo-managed; refusing to overwrite a custom workflow")
}

func inferVersion(root string) string {
	path := filepath.Join(root, ".github", "workflows", "validate.yml")
	data, err := os.ReadFile(path)
	if err != nil {
		return "unversioned"
	}
	text := string(data)
	marker := "github.com/Karageorgiou/GitMemo/cmd/gitmemo@"
	idx := strings.Index(text, marker)
	if idx < 0 {
		return "unversioned"
	}
	value := text[idx+len(marker):]
	if end := strings.IndexAny(value, " \t\r\n\"'"); end >= 0 {
		value = value[:end]
	}
	if value == "" {
		return "unversioned"
	}
	return value
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
	tmp, err := os.CreateTemp(filepath.Dir(path), ".gitmemo-upgrade-*")
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
