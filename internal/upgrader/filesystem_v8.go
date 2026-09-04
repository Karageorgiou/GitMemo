package upgrader

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/runethread/core/internal/fsafety"
	"github.com/runethread/core/internal/starter"
)

const managedGitAttributesPath = ".gitattributes"

func requireMigrationRoot(root string) error {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("resolve repository root: %w", err)
	}
	if err := fsafety.RequireDirectory(rootAbs); err != nil {
		return fmt.Errorf("unsafe repository root: %w", err)
	}
	return nil
}

func checkManagedGitAttributesOwnership(root string) error {
	path := filepath.Join(root, managedGitAttributesPath)
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect %s: %w", managedGitAttributesPath, err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return fmt.Errorf("%s exists but is not the managed regular file; refusing to overwrite it", managedGitAttributesPath)
	}
	data, err := fsafety.ReadRegularFileUnder(root, managedGitAttributesPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", managedGitAttributesPath, err)
	}
	if !bytes.Equal(data, starter.GitAttributes()) {
		return fmt.Errorf("%s does not exactly match the managed Runethread file; refusing to overwrite custom Git attributes", managedGitAttributesPath)
	}
	return nil
}

func requireRepositoryDirectory(root, rel string) error {
	if _, err := fsafety.DirectoryUnder(root, rel); err != nil {
		return fmt.Errorf("unsafe repository directory %s: %w", rel, err)
	}
	return nil
}

func readRepositoryRegularFile(root, rel string) ([]byte, error) {
	return fsafety.ReadRegularFileUnder(root, rel)
}

func readOptionalRepositoryRegularFile(root, rel string) ([]byte, bool, error) {
	path := filepath.Join(root, filepath.FromSlash(rel))
	if _, err := os.Lstat(path); os.IsNotExist(err) {
		return nil, false, nil
	} else if err != nil {
		return nil, false, fmt.Errorf("inspect %s: %w", rel, err)
	}
	data, err := readRepositoryRegularFile(root, rel)
	if err != nil {
		return nil, false, err
	}
	return data, true, nil
}

func takeRegularSnapshots(root string, paths []string) (map[string]snapshot, error) {
	if err := requireMigrationRoot(root); err != nil {
		return nil, err
	}
	out := make(map[string]snapshot, len(paths))
	for _, rel := range paths {
		path := filepath.Join(root, filepath.FromSlash(rel))
		info, err := os.Lstat(path)
		if os.IsNotExist(err) {
			out[rel] = snapshot{}
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("inspect %s: %w", rel, err)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return nil, fmt.Errorf("managed path %s must be a regular file before migration", rel)
		}
		data, err := fsafety.ReadRegularFileUnder(root, rel)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", rel, err)
		}
		out[rel] = snapshot{exists: true, data: data, mode: info.Mode().Perm()}
	}
	return out, nil
}
