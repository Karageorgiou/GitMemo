package fsafety

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// RequireDirectory requires path itself to be a real directory. Symbolic links
// and other special filesystem objects are rejected rather than followed.
func RequireDirectory(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%s is a symbolic link", path)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", path)
	}
	return nil
}

// ReadRegularFile requires path itself to be a regular file and refuses a
// symbolic-link leaf. Repository-owned paths should prefer ReadRegularFileUnder
// so ancestor directories are checked as well.
func ReadRegularFile(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("%s is a symbolic link", path)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%s is not a regular file", path)
	}
	return os.ReadFile(path)
}

// DirectoryUnder resolves a repository-relative path without following any
// symbolic link in the root, ancestor directories, or final directory.
func DirectoryUnder(root, rel string) (string, error) {
	return objectUnder(root, rel, true)
}

// RegularFileUnder resolves a repository-relative path without following any
// symbolic link in the root, ancestor directories, or final file. The final
// object must be a regular file.
func RegularFileUnder(root, rel string) (string, error) {
	return objectUnder(root, rel, false)
}

func objectUnder(root, rel string, wantDirectory bool) (string, error) {
	if strings.TrimSpace(rel) == "" {
		return "", fmt.Errorf("relative path must not be empty")
	}

	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve root: %w", err)
	}
	if err := RequireDirectory(rootAbs); err != nil {
		return "", fmt.Errorf("unsafe root: %w", err)
	}

	local := filepath.FromSlash(rel)
	if filepath.IsAbs(local) || filepath.VolumeName(local) != "" {
		return "", fmt.Errorf("path %q must be repository-relative", rel)
	}
	clean := filepath.Clean(local)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q escapes repository root", rel)
	}

	parts := strings.Split(clean, string(filepath.Separator))
	current := rootAbs
	for i, part := range parts {
		if part == "" || part == "." {
			continue
		}
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil {
			return "", err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("%s is a symbolic link", current)
		}
		if i < len(parts)-1 {
			if !info.IsDir() {
				return "", fmt.Errorf("%s is not a directory", current)
			}
			continue
		}
		if wantDirectory {
			if !info.IsDir() {
				return "", fmt.Errorf("%s is not a directory", current)
			}
		} else if !info.Mode().IsRegular() {
			return "", fmt.Errorf("%s is not a regular file", current)
		}
	}
	return current, nil
}

// RequireTree requires rel to resolve to a real directory beneath root and
// requires every traversed entry beneath it to be either a real directory or a
// regular file. Symbolic links and other special filesystem objects are
// rejected instead of followed or silently ignored.
func RequireTree(root, rel string) error {
	base, err := DirectoryUnder(root, rel)
	if err != nil {
		return err
	}
	return filepath.WalkDir(base, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("%s is a symbolic link", path)
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("%s is not a regular file", path)
		}
		return nil
	})
}

// ReadRegularFileUnder is the fail-closed file reader for repository-owned
// canonical and control-plane files.
func ReadRegularFileUnder(root, rel string) ([]byte, error) {
	path, err := RegularFileUnder(root, rel)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(path)
}
