# Repository Validation Specification — V1

## Purpose

`schema/memory-item.schema.json` defines the structural contract for one JSON sidecar. Repository validation also enforces invariants that require looking across files, memories, relationships, generated indexes, and time.

The canonical V1 implementation is the Go CLI:

```text
cmd/gitmemo
internal/memory
internal/indexer
internal/validation
```

Run it with:

```bash
gitmemo validate .
```

The validator is deterministic, reports errors precisely, exits non-zero on hard errors, and does not silently repair repository state.

### Schema enforcement strategy

The JSON Schema remains the normative sidecar contract. The Go implementation decodes sidecars into strict typed structures, rejects unknown fields, and enforces the V1 schema constraints directly. To prevent silent drift between the schema and Go implementation, the validator checks a canonical hash of `schema/memory-item.schema.json`. Any schema change therefore requires an explicit review/update of the Go validation contract before repository validation can pass again.

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

---

# 14. Generated-index integrity

Generated indexes are deterministically reconstructed from authoritative sidecars and project source files by:

```bash
gitmemo index --write .
```

Validation compares the generated representation with committed indexes and reports missing or stale generated files as errors.

Use:

```bash
gitmemo index --check .
```

for an explicit freshness-only check.

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

The repository model reserves these conceptual classes:

- `ERROR`: repository invariant is violated;
- `WARNING`: suspicious or stale condition requires review but is not deterministically invalid;
- `INFO`: useful audit information.

V1 exits non-zero when any `ERROR` exists and supports both human-readable and JSON output.

---

# 18. Mandatory V1 checks

The validator must cover at least:

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
- committed generated indexes are current;
- the Go validation contract has been reviewed whenever the canonical sidecar schema changes.

These checks are mandatory repository invariants, not optional ideas.
