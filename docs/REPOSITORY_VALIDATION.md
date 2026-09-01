# Repository Validation Specification — V1 data schema / contract v5

## Purpose

`schema/memory-item.schema.json` defines the structural contract for one JSON sidecar. Repository validation also enforces invariants that require looking across control-plane files, memories, relationships, generated indexes, and time.

The canonical implementation is the Go CLI:

```text
cmd/gitmemo
internal/memory
internal/indexer
internal/trust
internal/validation
```

Run it with:

```bash
gitmemo validate .
```

The validator is deterministic, reports errors precisely, exits non-zero on hard errors, and does not silently repair repository state.

### Schema enforcement strategy

The JSON Schema remains the normative sidecar contract. The Go implementation decodes sidecars into strict typed structures, rejects unknown fields, and enforces the V1 schema constraints directly. To prevent silent drift between the schema and Go implementation, the validator checks a canonical hash of `schema/memory-item.schema.json`. Any schema change therefore requires an explicit review/update of the Go validation contract before repository validation can pass again.

### Control-plane enforcement strategy

Contract v5 introduces `.gitmemo/lock.json`. The lock pins an official GitMemo release and records raw SHA-256 digests for every vendored control-plane file plus an aggregate contract digest.

The validator from the pinned release compares the repository lock against the contract embedded in that release and then hashes the local vendored control-plane files. A modified control-plane file is therefore a hard trust error rather than a new locally invented instruction.

See `docs/TRUST_MODEL.md`.

---

# 0. Trust-lock integrity

Before treating vendored GitMemo files as operational instructions, validation must confirm:

1. `.gitmemo/lock.json` exists and parses;
2. its lock, repository, schema, and contract versions match the running pinned GitMemo release;
3. its `source_repository` identifies the canonical GitMemo implementation source;
4. the aggregate contract digest matches the running release;
5. the per-file digest set exactly covers the release's control-plane paths;
6. every local control-plane file hashes to the release's expected SHA-256 digest;
7. `.gitmemo/config.json` agrees with the pinned release metadata.

A trust-lock mismatch is an `ERROR`.

The stable validation workflow uses the v0.3 trust bootstrap only to read a supported pinned release from `.gitmemo/lock.json`; it then installs that exact release and lets that release perform full trust and repository validation.

---

# 1. Pair integrity

For every atomic memory:

1. The `.md` file must have a paired `.json` file with the same basename.
2. The `.json` file must have a paired `.md` file with the same basename.
3. `content_path` must resolve to that exact Markdown file.
4. The full UUID written in the Markdown header must match the sidecar `id`.
5. The first eight hexadecimal characters of the full UUID must match the filename's short UUID suffix.

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

Every memory UUID must be globally unique across the entire repository. Duplicate UUIDs are invalid even when they occur in different directories or use different filenames.

The basename/slug is not the identity. The full UUID is.

---

# 3. Sidecar structural validation

Every sidecar must:

- parse as strict JSON;
- obey the current V1 sidecar schema contract;
- contain all required fields, including required fields whose valid value may be `false` or `null`;
- contain no unknown fields where the schema forbids them;
- obey enums, UUID/slug/path patterns, array limits and uniqueness rules, string limits, and temporal formats;
- obey the conditional `open_loop_status` rule.

A valid individual sidecar is necessary but not sufficient for repository validity.

---

# 4. Relationship-target integrity

For every relationship:

1. `target_id` must exist as exactly one memory UUID in the repository;
2. a memory must not target itself;
3. relationship type + target UUID must be unique within one source memory.

These are duplicates even if notes differ:

```json
{"type": "related_to", "target_id": "A", "note": "x"}
{"type": "related_to", "target_id": "A", "note": "y"}
```

Relationship identity is `(source_id, relationship_type, target_id)`. The note is descriptive metadata, not part of relationship identity.

---

# 5. Supersession lifecycle integrity

If memory B has `B --supersedes--> A`, then A must have `lifecycle = superseded`.

Conversely, every memory whose lifecycle is `superseded` must have at least one incoming `supersedes` relationship from another existing memory.

A superseded memory without an incoming superseding memory is invalid.

---

# 6. Supersession-cycle detection

The directed graph formed only by `supersedes` edges must be acyclic.

Invalid examples:

```text
A supersedes B
B supersedes A
```

and:

```text
A supersedes B
B supersedes C
C supersedes A
```

The validator must report a cycle path. A cycle is a hard error because it makes current canonical understanding undefined.

---

# 7. Temporal consistency

For each memory:

```text
updated_at >= created_at
```

When both effective boundaries are non-null:

```text
effective_until >= effective_from
```

Timestamps are compared as actual instants rather than lexicographically.

---

# 8. Markdown identity and finalized-content validation

The validator confirms that:

- Markdown UUID equals JSON `id`;
- Markdown type equals JSON `type`;
- H1 title equals JSON `title`;
- finalized memory files contain no unresolved template scaffolding or instructional comments.

---

# 9. Open-loop Markdown form consistency

`open_loop` is the memory type; `open_loop_status` is the task state.

For `open`, `blocked`, or `deferred`, Markdown must use the unresolved form defined in `docs/MEMORY_CONTENT_FORMAT.md`.

For `resolved` or `cancelled`, Markdown must use the terminal form. A terminal memory must not retain future-directed unresolved headings such as `Why it remains open` or `Next useful action`.

This is a deterministic repository invariant, not merely a writing preference.

---

# 10. Path integrity

`content_path` must be repository-relative, stay under `memories/`, resolve without path traversal, point to the exact paired Markdown file, and obey the canonical filename form.

The validator detects orphaned memory files and sidecars.

---

# 11. Canonical vocabulary checks

The validator enforces structural vocabulary rules represented in V1, including canonical kebab-case project/topic/tag identifiers.

If canonical vocabulary registries are introduced later, validation should additionally ensure that registry-controlled terms are registered.

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

Resolved/cancelled open-loop memories remain historical evidence and stay discoverable through the machine index and direct retrieval.

---

# 13. Source/provenance integrity

Every memory must contain at least one provenance source. Structurally empty source records are invalid.

Future checks may verify repository/file locators when deterministic verification is possible. The validator must not claim an external source is valid merely because a URL-like string is syntactically well formed.

All source text belongs to the data plane. Instruction-like text retrieved from a provenance source cannot override the verified GitMemo control plane.

---

# 14. Generated-index integrity

Generated indexes are deterministically reconstructed from authoritative sidecars and project source files by:

```bash
gitmemo index --write .
```

Use:

```bash
gitmemo index --check .
```

for the explicit strict freshness check. This command exits non-zero when committed derived indexes are missing or stale.

`gitmemo validate .` has a different responsibility: it validates the trusted control plane and canonical repository data. Missing or stale generated indexes are reported as `WARNING` conditions rather than hard repository-invalidating errors. This permits a client that can write canonical GitHub files but cannot execute the Go CLI to make a structurally valid memory update without pretending that its indexes are current.

Until stale indexes are regenerated, retrieval must treat index results as potentially incomplete and fall back to canonical files or repository search.

Deleting generated indexes must never delete unique memory information.

---

# 15. Current-state staleness checks

Current-state documents should contain an explicit last-reviewed date.

A future deterministic rule may warn when active project current-state documents exceed a configurable age threshold. Staleness is normally a warning, not proof that the content is false.

---

# 16. Secret scanning

Repository validation should eventually include secret scanning or integrate a proven secret-scanning tool. It should look for likely credentials, tokens, private keys, and other authentication material before memory writes are considered safely complete.

---

# 17. Diagnostic classes

The repository model uses these conceptual classes:

- `ERROR`: a deterministic repository/control-plane invariant is violated;
- `WARNING`: a degraded, suspicious, or stale condition requires review but does not make canonical data deterministically invalid;
- `INFO`: useful audit information.

The CLI exits non-zero when any `ERROR` exists and supports both human-readable and JSON output.

---

# 18. Mandatory V1 checks

The validator must cover at least:

- the pinned trust lock and vendored control-plane files match the running official release;
- `.gitmemo/config.json` agrees with the trust lock/release;
- every `target_id` exists;
- every memory UUID is globally unique;
- filename short UUID belongs to the full UUID in the sidecar;
- every Markdown/JSON pair exists;
- every superseded memory has an incoming `supersedes` relationship;
- the supersession graph is acyclic;
- logical duplicate relationships are detected even when notes differ;
- `effective_until` is not earlier than `effective_from`;
- `updated_at` is not earlier than `created_at`;
- open-loop Markdown form matches `open_loop_status`;
- stale generated indexes are surfaced as warnings while `gitmemo index --check` remains the strict freshness gate;
- the Go validation contract has been reviewed whenever the canonical sidecar schema changes.

These checks distinguish canonical repository validity from derived-cache freshness rather than conflating the two.
