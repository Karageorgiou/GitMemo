# GitMemo Index Format

## Purpose

GitMemo indexes are deterministic, disposable discovery accelerators generated from canonical memory sidecars and project files. They are not independent sources of truth and MUST NOT contain unique knowledge.

Index format v2 replaces the monolithic `index/memories.jsonl` machine index with bounded, sharded lookup structures. The goals are fast targeted reads, smaller Git diffs, lower write contention between independent conversations or agents, and a layout that can scale to substantially larger repositories without requiring an LLM to load one global catalog.

The canonical memory remains the Markdown + JSON pair under `memories/`. If an index is stale, missing, damaged, or unsupported, operators MUST fall back to canonical source files or repository search.

---

## Versioning

`index/catalog.json` contains:

- `index_version` — the committed index-format version;
- `record_count` — number of indexed atomic memory sidecars;
- `memory_source_sha256` — deterministic SHA-256 digest of the sorted memory-sidecar paths and bytes used to build the machine indexes;
- sharding parameters;
- ID-list and posting chunk sizes plus the maximum committed postings per term;
- indexed term fields;
- a human-readable description of the generated layout.

Index-format versioning is separate from the memory sidecar schema. A GitMemo release may change only the generated index format while preserving the canonical memory schema.

The index format is controlled by the pinned GitMemo release. Do not invent repository-local variants.

---

## Generated tree

Index v2 may contain:

```text
index/
├── catalog.json
├── by-id/
│   ├── 00/
│   │   ├── 0.json
│   │   ├── 1.json
│   │   └── ...
│   ├── 01/
│   └── ...
├── by-project/
│   └── <project-slug>.json
├── by-topic/
│   └── <topic-slug>.json
├── by-tag/
│   └── <tag-slug>.json
├── by-type/
│   └── <memory-type>.json
├── by-lifecycle/
│   └── <lifecycle>.json
├── by-open-loop-status/
│   └── <status>.json
├── terms/
│   └── <sha256-term-prefix>.json
├── projects.md
├── open-loops.md
└── preferences.md
```

Only non-empty machine shards are created. Therefore a clean repository with no memories normally contains `catalog.json` plus the human navigation indexes, but no `by-id`, taxonomy, or term shard files.

`index/STALE`, when present, is an explicit dirty marker rather than a generated current-index file. It tells readers that the committed discovery indexes MUST be treated as incomplete until regeneration succeeds.

The entire `index/` directory is generated output. `gitmemo index --write` is allowed to replace its generated contents and remove obsolete files from older index formats.

---

## Exact ID lookup

A UUID is assigned to an ID shard by computing SHA-256 over the lowercase full UUID and using the first three hexadecimal hash characters (12 bits). The first two hash characters select a directory and the third selects the shard file:

```text
<uuid> -> SHA-256(lowercase uuid) -> index/by-id/<first-two-hash-hex>/<third-hash-hex>.json
```

Each ID shard is a JSON object whose keys are full memory UUIDs and whose values contain retrieval metadata such as title, type, lifecycle, summary, taxonomy, provenance basis, relationships, content path, and sensitivity.

At most 4096 ID shard files are possible with the current three-hex-character / 12-bit SHA-256 prefix, distributed beneath at most 256 first-level directories. Hashing the complete UUID keeps shard distribution uniform even if a valid UUID producer has biased or sequential visible prefixes. A reader that already knows a UUID therefore opens one deterministic shard instead of scanning a repository-wide catalog.

With respect to total repository size, lookup requires a constant number of shard-path calculations and file reads. At one million uniformly distributed UUIDv4 memories, the expected average is about 244 records per ID shard instead of about 3906 with an 8-bit/256-shard layout. Actual latency still depends on filesystem, Git provider, network, record size, and distribution.

---

## Direct metadata indexes

The following indexes map a normalized schema-controlled or slug value to a sorted list of matching memory UUIDs:

- `by-project/<project-slug>.json`
- `by-topic/<topic-slug>.json`
- `by-tag/<tag-slug>.json`
- `by-type/<memory-type>.json`
- `by-lifecycle/<lifecycle>.json`
- `by-open-loop-status/<status>.json`

A direct index file has the form:

```json
{"ids":["<uuid>","<uuid>"]}
```

IDs are sorted deterministically.

These files allow project/topic/tag/type/status filtering without scanning every memory entry. A list of up to 1,024 UUIDs is stored inline in its descriptor. Larger lists are deterministically chunked into 1,024-ID files beneath a sibling directory named for the key, so a very large project, tag, lifecycle, or type never becomes one unbounded JSON blob. Reading the complete category remains O(number of matching IDs), which is unavoidable for an API that returns every match.

---

## Natural-language term index

GitMemo v2 includes a deterministic inverted term index for discovery from ordinary language.

### Tokenization

Indexed text is tokenized by:

1. treating Unicode letters and digits as term characters;
2. lowercasing Unicode letters;
3. treating punctuation and other characters as separators;
4. indexing whole resulting tokens.

The committed index does not perform stemming, embeddings, semantic-vector search, or language-specific synonym expansion. Retrieval aliases, topics, tags, entities, and good memory titles remain important for durable recall.

### Indexed fields and weights

Term postings are generated from these fields:

- title: weight 8
- aliases: weight 6
- projects: weight 5
- topics: weight 5
- tags: weight 5
- entity kinds and names: weight 4
- type, lifecycle, and open-loop status: weight 3
- summary: weight 2

Weights are discovery heuristics only. They do not express truth, confidence, authority, or importance.

A term receives a field's weight at most once per memory even if that term occurs repeatedly within the same weighted field group. This prevents repeated prose from artificially dominating retrieval.

### Hash sharding

A normalized term is deterministically assigned to a term shard from a prefix of:

```text
SHA-256(UTF-8 normalized term)
```

The current prefix length is recorded in `catalog.json`. Hash sharding is used instead of visible-letter sharding so common linguistic prefixes do not concentrate most terms into a few files.

Each term shard is a JSON object from normalized term to a descriptor containing its exact `document_frequency` and either inline postings, chunk metadata, or `suppressed: true`. A posting contains:

```json
{"id":"<uuid>","score":8}
```

Postings are sorted by score descending and UUID ascending. Up to 1,024 postings are stored inline. Larger posting lists are split into deterministic 1,024-posting files under `index/term-postings/<sha256-prefix>/<full-sha256>/`. A term with more than 32,768 matching memories retains its exact document frequency but its postings are intentionally suppressed: such a term is too broad to be a useful Git-native candidate set and the query must include a more specific term. This is language-independent and places a hard bound on committed postings for any one term.

For a query, the reader computes the shard for each normalized query term, reads only those shard files, combines postings, then resolves the highest-ranked UUIDs through the relevant `by-id` shard(s).

The query cost is therefore driven primarily by the number of query terms and their posting lists rather than by the total number of memory records.

---

## Search ranking

The built-in deterministic term search ranks candidate memories by:

1. number of distinct query terms matched, descending;
2. sum of field weights for the matched terms, descending;
3. UUID, ascending, as a deterministic final tie-breaker.

This is deliberately simple and inspectable. It is a first-stage discovery mechanism, not a semantic reasoning engine.

An LLM SHOULD read the canonical atomic memories for the selected results before relying on their factual content.

---

## Freshness and `index/STALE`

Canonical source data and generated indexes have different authority.

If an execution-capable client changes data that affects indexed metadata, it SHOULD run:

```bash
gitmemo index --write .
gitmemo index --check .
```

If a client can write repository files but cannot execute the GitMemo indexer, it SHOULD create or preserve:

```text
index/STALE
```

using the standard stale-marker text documented by the pinned GitMemo release. The marker MUST NOT be removed merely because an operator hopes the indexes are current. Successful deterministic regeneration removes it because the entire generated index tree is replaced.

Absence of `index/STALE` is a useful operational signal but is not, by itself, cryptographic proof that no out-of-band source edit occurred. `gitmemo index --check` remains the strict freshness check because it regenerates the expected index from canonical source state and compares the complete generated tree.

When freshness is unknown, an operator MUST treat index results as potentially incomplete and fall back to canonical files or repository search where completeness matters.

---

## Source digest

`memory_source_sha256` in `catalog.json` is computed deterministically over every sorted memory sidecar path and its exact bytes.

The digest makes the source state that produced a committed machine index explicit and audit-friendly. Recomputing the digest requires reading canonical sidecars, so normal fast queries do not recompute it on every lookup.

The digest is not a replacement for `gitmemo index --check`.

---

## Write and concurrency model

The v1 monolithic machine index caused every memory change to rewrite one global file. Index v2 removes that single machine-index write hotspot.

Logically, a memory affects only its relevant ID, taxonomy, type/status, and term shards. The current Go indexer rebuilds the complete generated tree in a temporary directory and atomically swaps it into place for correctness and simplicity; Git itself records only files whose resulting bytes changed.

This reduces unrelated Git conflicts even though regeneration remains deterministic and full-tree locally.

Human convenience indexes such as `open-loops.md` may still be shared files and can conflict if two writers modify the same logical category concurrently. Canonical atomic memories remain the merge authority; generated files should be regenerated rather than manually reconciled as facts.

---

## Failure behavior

If index generation fails:

- do not rewrite canonical memories to satisfy the indexer;
- do not claim indexes are current;
- preserve or create the stale marker when possible;
- diagnose the canonical validation error or indexer failure;
- retry regeneration only after the source problem is understood.

A validator may report stale generated indexes as warnings while canonical memory validation still passes.

`gitmemo index --check` is intentionally strict and fails when expected files are missing, bytes differ, obsolete generated files remain, or `index/STALE` exists.

---

## Optional local acceleration

A future GitMemo client may build an uncommitted local SQLite/FTS or similar cache from canonical Git/index data for even faster local retrieval. Such a cache would be disposable implementation state, not part of the committed GitMemo repository format and not an authority source.

Index v2 does not require SQLite, a vector database, embeddings, a server, or any paid service.
