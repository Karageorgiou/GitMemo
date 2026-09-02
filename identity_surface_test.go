package runethread

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNativeIdentitySurfaceHasNoLegacyOperationalNames(t *testing.T) {
	for _, path := range []string{
		"gitmemo-bootstrap.json",
		"cmd/gitmemo",
		"docs/EXTENDING_GITMEMO.md",
	} {
		if _, err := os.Stat(path); err == nil {
			t.Errorf("legacy native path still exists: %s", path)
		} else if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("inspect %s: %v", path, err)
		}
	}

	for _, path := range []string{
		"runethread-bootstrap.json",
		"cmd/runethread",
		"docs/EXTENDING_RUNETHREAD.md",
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("required Runethread path %s is missing: %v", path, err)
		}
	}

	// Runtime packages are native Runethread implementation. The predecessor
	// name belongs only in the finite migration implementation under
	// internal/upgrader and in historically truthful migration documentation.
	for _, root := range []string{
		"cmd/runethread",
		"internal/buildinfo",
		"internal/indexer",
		"internal/memory",
		"internal/starter",
		"internal/trust",
		"internal/validation",
	} {
		walkTextFiles(t, root, func(path, text string) {
			if strings.Contains(strings.ToLower(text), "gitmemo") {
				t.Errorf("legacy product name remains in native runtime file %s", path)
			}
		})
	}

	// Current onboarding and operational surfaces may truthfully describe the
	// finite GitMemo v0.5.0 predecessor migration, but they must not advertise
	// old commands, old source authority, or old native implementation paths.
	for _, path := range []string{
		"README.md",
		"AI_SETUP.md",
		"MEMORY_PROTOCOL.md",
		"runethread-bootstrap.json",
		"docs/EXTENDING_RUNETHREAD.md",
		"docs/GETTING_STARTED.md",
		"docs/INDEX_FORMAT.md",
		"docs/INDEX_SCALE.md",
		"docs/MEMORY_CONTENT_FORMAT.md",
		"docs/REPOSITORY_ROLES.md",
		"docs/REPOSITORY_VALIDATION.md",
		"docs/SOURCES.md",
		"docs/TAXONOMY.md",
		"docs/TRUST_MODEL.md",
		"docs/USER_COMMANDS.md",
		".github/workflows/validate.yml",
		".github/workflows/release.yml",
	} {
		text := readTextFile(t, path)
		for _, legacy := range []string{
			"github.com/Karageorgiou/GitMemo",
			"cmd/gitmemo",
			"gitmemo init",
			"gitmemo upgrade",
			"gitmemo validate",
			"gitmemo search",
			"gitmemo index",
			"gitmemo trust",
			"GitMemo: store",
			"GitMemo: search",
			"GitMemo validation",
			"GitMemo generated indexes",
			"GITMEMO_SCALE_N",
			".gitmemo-index-",
		} {
			if strings.Contains(text, legacy) {
				t.Errorf("legacy operational identity %q remains in %s", legacy, path)
			}
		}
	}
}

func walkTextFiles(t *testing.T, root string, check func(path, text string)) {
	t.Helper()
	if err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		switch filepath.Ext(path) {
		case ".go", ".md", ".json", ".yml", ".yaml":
			check(filepath.ToSlash(path), readTextFile(t, path))
		}
		return nil
	}); err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
}

func readTextFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}
