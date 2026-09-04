package memory

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/runethread/core/internal/fsafety"
)

// ValidateSchemaContractStrict verifies the vendored schema without following
// repository-root, ancestor, or leaf symbolic links.
func ValidateSchemaContractStrict(root string) error {
	data, err := fsafety.ReadRegularFileUnder(root, "schema/memory-item.schema.json")
	if err != nil {
		return fmt.Errorf("read schema: %w", err)
	}
	hash, err := canonicalJSONSHA256(data)
	if err != nil {
		return fmt.Errorf("parse schema JSON: %w", err)
	}
	if hash != ExpectedSchemaContractSHA256 {
		return fmt.Errorf("schema contract hash is %s, expected %s; review and update Go validation rules with the schema", hash, ExpectedSchemaContractSHA256)
	}
	return nil
}

// DiscoverStrict discovers canonical memory pairs only after proving the whole
// memories tree is composed of real directories and regular files.
func DiscoverStrict(root string) (sidecars []string, markdown []string, err error) {
	base := filepath.Join(root, "memories")
	if _, statErr := os.Lstat(base); statErr != nil {
		if os.IsNotExist(statErr) {
			return nil, nil, nil
		}
		return nil, nil, statErr
	}
	if err := fsafety.RequireTree(root, "memories"); err != nil {
		return nil, nil, fmt.Errorf("unsafe memories tree: %w", err)
	}
	err = filepath.WalkDir(base, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		switch strings.ToLower(filepath.Ext(path)) {
		case ".json":
			sidecars = append(sidecars, path)
		case ".md":
			markdown = append(markdown, path)
		}
		return nil
	})
	sort.Strings(sidecars)
	sort.Strings(markdown)
	return sidecars, markdown, err
}

// LoadUnder reads one repository-relative memory sidecar without following
// symbolic links in the repository path.
func LoadUnder(root, rel string) (Memory, []SchemaProblem) {
	data, err := fsafety.ReadRegularFileUnder(root, rel)
	if err != nil {
		return Memory{}, []SchemaProblem{{Message: err.Error()}}
	}
	return Decode(data)
}

// LoadAllStrict loads all canonical sidecars from a strict memories tree.
func LoadAllStrict(root string) ([]Record, error) {
	sidecars, _, err := DiscoverStrict(root)
	if err != nil {
		return nil, err
	}
	records := make([]Record, 0, len(sidecars))
	for _, path := range sidecars {
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil, err
		}
		m, problems := LoadUnder(root, filepath.ToSlash(rel))
		if len(problems) > 0 {
			return nil, fmt.Errorf("%s: %s", relative(root, path), problems[0].Error())
		}
		records = append(records, Record{Path: path, Memory: m})
	}
	return records, nil
}
