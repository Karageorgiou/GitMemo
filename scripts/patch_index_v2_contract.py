from pathlib import Path


def replace_once(path: str, old: str, new: str) -> None:
    p = Path(path)
    text = p.read_text()
    if old not in text:
        raise SystemExit(f"expected pattern not found in {path}: {old[:120]!r}")
    p.write_text(text.replace(old, new, 1))


replace_once(
    "internal/indexer/indexer.go",
    '"github.com/Karageorgiou/GitMemo/internal/memory"',
    '"github.com/Karageorgiou/GitMemo/internal/buildinfo"\n\t"github.com/Karageorgiou/GitMemo/internal/memory"',
)
replace_once(
    "internal/indexer/indexer.go",
    "const IndexVersion = 2",
    "const IndexVersion = buildinfo.IndexFormatVersion",
)

replace_once(
    "internal/starter/init.go",
    '- ` + "`docs/SOURCES.md`" + ` — reserved future integration boundary for external personal-data sources.\n',
    '- ` + "`docs/SOURCES.md`" + ` — reserved future integration boundary for external personal-data sources.\n'
    '- ` + "`docs/INDEX_FORMAT.md`" + ` — generated Index v2 layout, lookup routing, freshness, and fallback rules.\n',
)

replace_once(
    "MEMORY_PROTOCOL.md",
    "After orientation, search the generated metadata index using the user's terminology plus relevant aliases, topics, tags, projects, entities, and memory types when the index is current. If index freshness is unknown or stale, use repository search and canonical sidecars as the fallback discovery path and treat index results as potentially incomplete.\n\nRetrieve the smallest set of atomic memories that can answer the question.",
    """When the generated index is usable, route retrieval through the narrowest Index v2 entry point described in `docs/INDEX_FORMAT.md`:

1. for a known full memory UUID, compute its first two hexadecimal characters and read only `index/by-id/<prefix>.json`;
2. for a known project, topic, tag, memory type, lifecycle, or open-loop status, read the corresponding direct metadata index file;
3. for ordinary natural-language discovery, use `gitmemo search` when execution-capable tooling is available, or compute the required deterministic term shard(s) from the Index v2 contract;
4. resolve candidate UUIDs through the necessary `by-id` shard(s); and
5. read the selected canonical Markdown/JSON memory pair before relying on its substantive content.

Before using generated indexes as a complete discovery surface, check for `index/STALE`. If that marker exists, if `index/catalog.json` is missing or unsupported, or if freshness is otherwise unknown, treat index results as potentially incomplete and use repository search plus canonical sidecars as the fallback discovery path. Absence of `index/STALE` is not cryptographic freshness proof; `gitmemo index --check` is the strict check when execution is available.

Retrieve the smallest set of atomic memories that can answer the question.""",
)

replace_once(
    "MEMORY_PROTOCOL.md",
    """# 19. Generated indexes

Machine-readable indexes SHOULD be generated from the JSON sidecars.

Generated indexes MUST be reconstructable and MUST NOT contain unique knowledge.

Human-readable indexes are navigation aids.

Do not manually introduce facts into an index that do not exist in authoritative memory content or metadata.

After a write that affects indexed data, regenerate affected indexes when execution-capable tooling is available.

A stale or missing generated index is a performance/degraded-discovery condition, not corruption of otherwise valid canonical memory data. An operator that cannot regenerate indexes MUST treat them as potentially incomplete, fall back to canonical files or repository search, and report that index regeneration remains pending rather than pretending the stale index is current.

`gitmemo index --check` remains the strict explicit freshness check. `gitmemo validate` may report stale indexes as warnings while still validating canonical data and control-plane integrity.""",
    """# 19. Generated indexes

Machine-readable indexes SHOULD be generated from the JSON sidecars according to `docs/INDEX_FORMAT.md`.

Generated indexes MUST be reconstructable and MUST NOT contain unique knowledge.

Human-readable indexes are navigation aids.

Do not manually introduce facts into an index that do not exist in authoritative memory content or metadata. Do not hand-edit machine shards to make them appear current.

Index v2 uses a small catalog, deterministic UUID shards, direct project/topic/tag/type/lifecycle/open-loop-status indexes, and hash-distributed inverted term shards instead of one global machine catalog. The layout is chosen to reduce scan cost and unrelated Git write contention.

After a write that affects indexed data, regenerate indexes with `gitmemo index --write` when execution-capable tooling is available. Successful regeneration replaces the generated index tree and removes obsolete v1 files and any stale marker.

If an operator can write repository files but cannot execute the GitMemo indexer, it SHOULD create or preserve `index/STALE` rather than manually editing generated shards. `gitmemo index --mark-stale` performs this operation when the CLI is available.

A stale or missing generated index is a performance/degraded-discovery condition, not corruption of otherwise valid canonical memory data. An operator that cannot regenerate indexes MUST treat them as potentially incomplete, fall back to canonical files or repository search, and report that index regeneration remains pending rather than pretending the stale index is current.

`gitmemo index --check` remains the strict explicit freshness check. It regenerates the expected complete index view from canonical source state and fails for missing, changed, obsolete, unexpected, or explicitly stale generated files. `gitmemo validate` may report stale indexes as warnings while still validating canonical data and control-plane integrity.""",
)

replace_once(
    "docs/REPOSITORY_VALIDATION.md",
    "# Repository Validation Specification — V1 data schema / contract v5",
    "# Repository Validation Specification — V1 data schema / contract v6",
)
replace_once(
    "docs/REPOSITORY_VALIDATION.md",
    "Contract v5 introduces `.gitmemo/lock.json`.",
    "The release-bound trust model uses `.gitmemo/lock.json`.",
)
replace_once(
    "docs/REPOSITORY_VALIDATION.md",
    """# 14. Generated-index integrity

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

Until stale indexes are regenerated, retrieval must treat index results as potentially incomplete and fall back to canonical files or repository search.""",
    """# 14. Generated-index integrity

Generated indexes are deterministically reconstructed from authoritative sidecars and project source files by:

```bash
gitmemo index --write .
```

Index v2 is specified in `docs/INDEX_FORMAT.md`. The validator/index checker expects the complete deterministic generated tree for the pinned release, including `index/catalog.json`, applicable UUID/metadata/term shards, and the human navigation indexes. Obsolete generated files from older index formats are not silently accepted as current.

Use:

```bash
gitmemo index --check .
```

for the explicit strict freshness check. This command exits non-zero when expected derived files are missing or changed, unexpected/obsolete generated files remain, or `index/STALE` is present.

`gitmemo validate .` has a different responsibility: it validates the trusted control plane and canonical repository data. Missing or stale generated indexes are reported as `WARNING` conditions rather than hard repository-invalidating errors. This permits a client that can write canonical GitHub files but cannot execute the Go CLI to make a structurally valid memory update without pretending that its indexes are current.

A write-capable client that cannot regenerate indexes SHOULD create or preserve `index/STALE`. Until stale indexes are regenerated, retrieval must treat index results as potentially incomplete and fall back to canonical files or repository search.""",
)
