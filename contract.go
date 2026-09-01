package gitmemo

import "embed"

// ContractFS contains the versioned operational contract that a GitMemo memory
// repository needs in order to be self-describing to humans, LLMs, and tooling.
//
//go:embed MEMORY_PROTOCOL.md schema/memory-item.schema.json docs/MEMORY_SCHEMA.md docs/MEMORY_CONTENT_FORMAT.md docs/TAXONOMY.md docs/REPOSITORY_VALIDATION.md docs/USER_COMMANDS.md docs/EXTENDING_GITMEMO.md docs/TRUST_MODEL.md docs/SOURCES.md templates/*.md
var ContractFS embed.FS

// ContractPaths is the canonical set of operational-contract paths copied into
// a new memory repository by `gitmemo init`.
var contractPaths = []string{
	"MEMORY_PROTOCOL.md",
	"schema/memory-item.schema.json",
	"docs/MEMORY_SCHEMA.md",
	"docs/MEMORY_CONTENT_FORMAT.md",
	"docs/TAXONOMY.md",
	"docs/REPOSITORY_VALIDATION.md",
	"docs/USER_COMMANDS.md",
	"docs/EXTENDING_GITMEMO.md",
	"docs/TRUST_MODEL.md",
	"docs/SOURCES.md",
	"templates/fact.md",
	"templates/preference.md",
	"templates/decision.md",
	"templates/state.md",
	"templates/open_loop.md",
	"templates/correction.md",
	"templates/milestone.md",
	"templates/reference.md",
}

// ContractPaths returns a copy of the canonical operational-contract paths.
func ContractPaths() []string {
	return append([]string(nil), contractPaths...)
}
