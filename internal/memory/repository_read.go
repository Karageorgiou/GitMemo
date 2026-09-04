package memory

import "github.com/runethread/core/internal/fsafety"

// LoadUnder reads a repository-relative memory sidecar without following
// symbolic links in the repository root, ancestor directories, or final file.
func LoadUnder(root, rel string) (Memory, []SchemaProblem) {
	data, err := fsafety.ReadRegularFileUnder(root, rel)
	if err != nil {
		return Memory{}, []SchemaProblem{{Message: err.Error()}}
	}
	return Decode(data)
}
