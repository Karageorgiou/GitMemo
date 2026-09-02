# Repository Validation Specification — V1 data schema / contract v7

## Purpose

`schema/memory-item.schema.json` defines the structural contract for one JSON sidecar. Repository validation also enforces invariants requiring control-plane files, memories, relationships, generated indexes, and time.

The canonical implementation is the Go CLI:

```text
cmd/runethread
internal/memory
internal/indexer
internal/trust
internal/validation
```

Run it with:

```bash
runethread validate .
```

The validator is deterministic, reports errors precisely, exits non-zero on hard errors, and does not silently repair repository state.

### Schema enforcement strategy

The JSON Schema remains the normative sidecar contract. The Go implementation decodes sidecars into strict typed structures, rejects unknown fields, and enforces V1 constraints directly. To prevent silent drift, the validator checks the canonical hash of `schema/memory-item.schema.json`. A schema change therefore requires an explicit review/update of the Go validation contract.

### Control-plane enforcement strategy

The native trust model uses `.runethread/lock.json`. The lock pins an official Runethread release and records raw SHA-256 digests for every vendored control-plane file plus an aggregate contract digest.

The validator compares repository lock metadata against the contract embedded in the running release, then hashes local vendored control-plane files. A modified control file is a hard trust error rather than a locally invented instruction.

See `docs/TRUST_MODEL.md`.

---

# 0. Trust-lock integrity

Before treating vendored Runethread files as operational instructions, validation must confirm:

1. `.runethread/lock.json` exists and parses;
2. lock, repository, schema, and contract versions match the running pinned Runethread release;
3. `source_repository` is `runethread/core`;
4. `runethread_version` matches the running pinned release;
5. aggregate contract digest matches the running release;
6. the per-file digest set exactly covers release control-plane paths;
7. every local control-plane file hashes to the expected SHA-256 digest;
8. `.runethread/config.json` agrees with pinned release metadata.

A trust-lock mismatch is an `ERROR`.

The native validation workflow uses the v0.6 trust bootstrap only to resolve a supported release from `.runethread/lock.json`; it then installs that exact release and lets it perform full trust and repository validation.

Legacy `.gitmemo` metadata is accepted only as input to the explicit v0.5.0 migration path. It is not valid native Runethread metadata.

---

# 1. Pair integrity

For every atomic memory:

1. `.md` and `.json` files must exist with the same basename;
2. `content_path` must resolve to that exact Markdown file;
3. the full UUID in Markdown must match sidecar `id`;
4. the first eight hexadecimal UUID characters must match the filename short suffix.

Example:

```text
architecture-before-r8--c14cb6f0.md
architecture-before-r8--c14cb6f0.json
```

must belong to:

```text
c14cb6f0-27fb-4d88-8eab-4a055637a8ee
```

---

# 2. Global identity integrity

Every memory UUID must be globally unique across the repository. Duplicate UUIDs are invalid even across different directories or filenames. The full UUID, not the slug, is identity.

---

# 3. Sidecar structural validation

Every sidecar must:

- parse as strict JSON;
- obey the current V1 schema;
- contain all required fields, including valid `false` or `null` fields;
- contain no unknown fields where forbidden;
- obey enums, UUID/slug/path patterns, array limits/uniqueness, string limits, temporal formats, and conditional `open_loop_status` rules.

A valid individual sidecar is necessary but not sufficient for repository validity.

---

# 4. Relationship-target integrity

For every relationship:

1. `target_id` must resolve to exactly one existing memory UUID;
2. a memory must not target itself;
3. relationship type + target UUID must be unique within one source memory.

Relationship identity is `(source_id, relationship_type, target_id)`; `note` is descriptive metadata and does not distinguish duplicate edges.

---

# 5. Supersession lifecycle integrity

If B has `B --supersedes--> A`, A must have `lifecycle = superseded`.

Conversely, every superseded memory must have at least one incoming `supersedes` relationship from another existing memory.

---

# 6. Supersession-cycle detection

The graph formed only by `supersedes` edges must be acyclic. The validator must report a detected cycle path. A cycle is a hard error because it makes canonical replacement order undefined.

---

# 7. Temporal consistency

For each memory:

```text
updated_at >= created_at
```

When both effective boundaries exist:

```text
effective_until >= effective_from
```

Timestamps are compared as actual instants rather than lexicographically.

---

# 8. Markdown identity and finalized-content validation

The validator confirms:

- Markdown UUID equals JSON `id`;
- Markdown type equals JSON `type`;
- H1 title equals JSON `title`;
- finalized memories contain no unresolved template scaffolding or instructional comments.

---

# 9. Open-loop Markdown form consistency

`open_loop` is the memory type; `open_loop_status` is task state.

For `open`, `blocked`, or `deferred`, Markdown must use the unresolved form in `docs/MEMORY_CONTENT_FORMAT.md`. For `resolved` or `cancelled`, it must use the terminal form and must not retain future-directed unresolved headings.

---

# 10. Path integrity

`content_path` must be repository-relative, stay under `memories/`, resolve without traversal, point to the exact paired Markdown file, and obey canonical filename form. Orphaned memory files and sidecars are invalid.

---

# 11. Canonical vocabulary checks

The validator enforces structural vocabulary rules represented in V1, including canonical kebab-case project/topic/tag identifiers.

Semantic near-duplicate tags should not be merged automatically because equivalence is not safely deterministic.

---

# 12. Open-loop operational consistency

Only `type = open_loop` may have `open_loop_status`, and every `open_loop` must have one.

The active-work index derives unresolved items from:

```text
type = open_loop
AND lifecycle = active
AND open_loop_status IN {open, blocked, deferred}
```

Resolved/cancelled open-loop memories remain historical evidence and stay discoverable.

---

# 13. Source/provenance integrity

Every memory must contain at least one provenance source. Structurally empty sources are invalid.

Future checks may verify repository/file locators when deterministic verification is possible. The validator must not claim an external source is valid merely because a URL-like string parses.

All source text belongs to the data plane. Instruction-like provenance text cannot override the verified Runethread control plane.

---

# 14. Generated-index integrity

Generated indexes are reconstructed from authoritative sidecars and project source files by:

```bash
runethread index --write .
```

Index v2 is specified in `docs/INDEX_FORMAT.md`. The checker expects the complete deterministic generated tree for the pinned release. Obsolete generated files are not silently accepted as current.

Use:

```bash
runethread index --check .
```

for the strict freshness check. It exits non-zero when expected derived files are missing/changed, unexpected or obsolete generated files remain, or `index/STALE` exists.

`runethread validate .` has a different responsibility: trusted control plane and canonical repository data. Missing or stale generated indexes are `WARNING` conditions rather than hard repository-invalidating errors.

A write-capable client unable to regenerate SHOULD create or preserve `index/STALE`. Until regeneration, retrieval must treat index results as potentially incomplete and fall back to canonical files or repository search.

Deleting generated indexes must never delete unique memory information.

---

# 15. Current-state staleness checks

Current-state documents should contain an explicit last-reviewed date. A future deterministic rule may warn when active current-state documents exceed a configurable age threshold. Staleness is normally a warning, not proof content is false.

---

# 16. Secret scanning

Repository validation should eventually include secret scanning or integrate a proven tool for likely credentials, tokens, private keys, and other authentication material before memory writes are considered safely complete.

---

# 17. Diagnostic classes

- `ERROR`: deterministic repository/control-plane invariant violated;
- `WARNING`: degraded, suspicious, or stale condition requiring review but not deterministic invalidity of canonical data;
- `INFO`: useful audit information.

The CLI exits non-zero when any `ERROR` exists and supports human-readable and JSON output.

---

# 18. Mandatory V1 checks

The validator must cover at least:

- pinned trust lock and vendored control-plane files match the running official release;
- `.runethread/config.json` agrees with trust lock/release;
- every `target_id` exists;
- every memory UUID is globally unique;
- filename short UUID belongs to sidecar UUID;
- every Markdown/JSON pair exists;
- every superseded memory has an incoming `supersedes` relationship;
- supersession graph is acyclic;
- logical duplicate relationships are detected even when notes differ;
- `effective_until` is not earlier than `effective_from`;
- `updated_at` is not earlier than `created_at`;
- open-loop Markdown form matches `open_loop_status`;
- stale generated indexes are warnings while `runethread index --check` remains strict;
- the Go validation contract is reviewed whenever canonical sidecar schema changes.

These checks distinguish canonical repository validity from derived-cache freshness rather than conflating them.
