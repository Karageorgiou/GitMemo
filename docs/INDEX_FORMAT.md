# Runethread Index Format

## Purpose

Runethread indexes are deterministic, disposable discovery accelerators generated from canonical memory sidecars and project files. They are not independent sources of truth and MUST NOT contain unique knowledge.

Index format v2 replaces the old monolithic `index/memories.jsonl` machine index with bounded, sharded lookup structures. The goals are fast targeted reads, smaller Git diffs, lower write contention between independent conversations or agents, and a layout that can scale without requiring an LLM to load one global catalog.

The canonical memory remains the Markdown + JSON pair under `memories/`. If an index is stale, missing, damaged, unsupported, or stored through unsafe repository filesystem objects, operators MUST fall back to valid canonical source files or repository search.

---

## Versioning

`index/catalog.json` contains:

- `index_version` — committed index-format version;
- `record_count` — number of indexed atomic memory sidecars;
- `memory_source_sha256` — deterministic SHA-256 digest of sorted memory-sidecar paths and bytes used to build the machine indexes;
- sharding parameters;
- ID-list and posting chunk sizes plus maximum committed postings per term;
- indexed term fields;
- a description of the generated layout.

Index-format versioning is separate from the memory sidecar schema and from runtime/distribution release identity. A runtime release may change while continuing to embed the same contract release and Index v2 bytes.

The index format is controlled by the repository's pinned Runethread **contract release**. Do not invent repository-local variants, and do not repin an unchanged repository merely because a newer compatible runtime exists.

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
├── term-postings/
│   └── <sha256-prefix>/<full-sha256>/<chunk>.json
├── projects.md
├── open-loops.md
└── preferences.md
```

Only non-empty machine shards are created. A clean repository with no memories normally contains `catalog.json` plus human navigation indexes but no UUID, taxonomy, or term shards.

`index/STALE`, when present, is an explicit dirty marker rather than a generated current-index file. It tells readers that committed discovery indexes MUST be treated as incomplete until regeneration succeeds.

The entire `index/` directory is generated output. `runethread index --write` may replace its generated contents and remove obsolete files from older index formats.

For contract v8, the `index/` directory and every generated shard read as repository state MUST be reached through a real repository root and real ancestor directories. Index files MUST be regular files. Symbolic links and unsupported special filesystem objects in the generated index tree are not valid shortcuts to matching bytes and MUST be rejected rather than followed.

---

## Exact ID lookup

A UUID is assigned to an ID shard by computing SHA-256 over the lowercase full UUID and using the first three hexadecimal hash characters (12 bits). The first two characters select a directory and the third selects a shard file:

```text
<uuid> -> SHA-256(lowercase uuid) -> index/by-id/<first-two-hash-hex>/<third-hash-hex>.json
```

Each ID shard is a JSON object whose keys are full memory UUIDs and whose values contain retrieval metadata such as title, type, lifecycle, summary, taxonomy, provenance basis, relationships, content path, and sensitivity.

At most 4096 ID shard files are possible with the current 12-bit SHA-256 prefix, distributed beneath at most 256 first-level directories. Hashing the complete UUID keeps distribution uniform even when a UUID producer has biased or sequential visible prefixes.

A reader that already knows a UUID therefore opens one deterministic shard instead of scanning a repository-wide catalog.

---

## Direct metadata indexes

These indexes map a normalized schema-controlled or slug value to a sorted list of matching UUIDs:

- `by-project/<project-slug>.json`
- `by-topic/<topic-slug>.json`
- `by-tag/<tag-slug>.json`
- `by-type/<memory-type>.json`
- `by-lifecycle/<lifecycle>.json`
- `by-open-loop-status/<status>.json`

A direct index descriptor has the form:

```json
{"ids":["<uuid>","<uuid>"]}
```

IDs are deterministic and sorted. Lists up to 1,024 UUIDs are stored inline. Larger lists are deterministically chunked into 1,024-ID files beneath a sibling directory for that key, preventing one unbounded category blob.

---

## Natural-language term index

Index v2 includes a deterministic inverted term index for ordinary-language discovery.

### Tokenization

Indexed text is tokenized by:

1. treating Unicode letters and digits as term characters;
2. lowercasing Unicode letters;
3. treating punctuation and other characters as separators;
4. indexing whole resulting tokens.

The committed index does not perform stemming, embeddings, semantic-vector search, or language-specific synonym expansion. Retrieval aliases, topics, tags, entities, and good memory titles therefore remain important.

### Indexed fields and weights

Term postings are generated from:

- title: weight 8
- aliases: weight 6
- projects: weight 5
- topics: weight 5
- tags: weight 5
- entity kinds and names: weight 4
- type, lifecycle, and open-loop status: weight 3
- summary: weight 2

Weights are discovery heuristics only. They do not express truth, confidence, authority, or importance.

A term receives a field group's weight at most once per memory even if repeated within that group.

### Hash sharding

A normalized term is assigned from a prefix of:

```text
SHA-256(UTF-8 normalized term)
```

The current prefix length is recorded in `catalog.json`; v2 currently uses three hexadecimal characters. Hash sharding avoids concentrating common linguistic prefixes into a few files.

Each term shard maps normalized terms to descriptors containing exact `document_frequency` and either inline postings, chunk metadata, or `suppressed: true`. A posting has the form:

```json
{"id":"<uuid>","score":8}
```

Postings are sorted by score descending and UUID ascending. Up to 1,024 postings are stored inline. Larger posting lists are split into deterministic 1,024-posting files under `index/term-postings/<sha256-prefix>/<full-sha256>/`.

A term matching more than 32,768 memories retains exact document frequency but suppresses committed postings. Such a term is too broad to be a useful Git-native candidate set and the query must include a more specific term. This places a hard bound on committed postings for any one term.

For a query, the reader computes the shard for each normalized query term, reads only those shard files, combines postings, then resolves highest-ranked UUIDs through relevant `by-id` shards.

---

## Search ranking

Built-in deterministic term search ranks candidates by:

1. number of distinct query terms matched, descending;
2. sum of field weights for matched terms, descending;
3. UUID ascending as the final deterministic tie-breaker.

This is a transparent first-stage discovery mechanism, not a semantic reasoning engine. An LLM SHOULD read canonical atomic memories for selected results before relying on factual content.

---

## Canonical source filesystem integrity

Index generation is only deterministic over valid canonical repository objects. For contract v8, source discovery MUST reject symbolic links or unsupported special objects in repository-owned canonical source trees rather than hash or index the bytes reached through them.

In particular:

- the repository root and traversed ancestors for `memories/` and `projects/` MUST be real directories;
- canonical memory sidecars/Markdown and project source files used during deterministic generation MUST be regular files; and
- path traversal or volume escape MUST NOT be accepted as an indexed source path.

This object-integrity rule is part of source determinism. Two files with identical bytes are not equivalent index sources when one is reached through an unsupported symbolic-link path.

---

## Freshness and `index/STALE`

Canonical source data and generated indexes have different authority.

If an execution-capable client changes indexed metadata, it SHOULD run:

```bash
runethread index --write .
runethread index --check .
```

If a client can write repository files but cannot execute the Runethread indexer, it SHOULD create or preserve:

```text
index/STALE
```

using the standard stale-marker text from the pinned contract release. The marker MUST NOT be removed merely because an operator hopes indexes are current. Successful deterministic regeneration removes it by replacing the generated index tree.

Absence of `index/STALE` is useful but not cryptographic proof that no out-of-band source edit occurred. `runethread index --check` is the strict check because it regenerates expected output from canonical source state and compares the complete generated tree.

When freshness is unknown, treat index results as potentially incomplete and fall back to canonical files or repository search where completeness matters.

---

## Source digest

`memory_source_sha256` in `catalog.json` is computed deterministically over every sorted canonical memory sidecar path and its exact bytes after the source tree has satisfied the contract-v8 filesystem-object rules.

The digest makes the source state used to produce a committed machine index explicit and audit-friendly. Recomputing it requires reading canonical sidecars, so fast queries do not recompute it on every lookup.

The digest is not a replacement for filesystem-object validation or `runethread index --check`.

---

## Write and concurrency model

The old monolithic machine index caused every memory change to rewrite one global file. Index v2 removes that hotspot.

Logically, a memory affects only relevant ID, taxonomy, type/status, and term shards. The current Go indexer rebuilds the complete generated tree in a temporary directory and atomically swaps it into place for correctness and simplicity; Git records only files whose resulting bytes changed.

This reduces unrelated Git conflicts even though local regeneration remains deterministic and full-tree.

Human convenience indexes such as `open-loops.md` may still be shared files and can conflict when two writers change the same logical category. Canonical atomic memories remain merge authority; generated files should be regenerated rather than manually reconciled as facts.

---

## Failure behavior

If index generation or reading fails:

- do not rewrite canonical memories to satisfy the indexer;
- do not claim indexes are current;
- reject unsafe symbolic-link/special-object repository paths instead of following them;
- preserve or create the stale marker when possible;
- diagnose the canonical validation error, filesystem-object error, or indexer failure;
- retry only after the source problem is understood.

A validator may report stale indexes as warnings while canonical validation still passes. Unsafe authoritative filesystem objects are not a stale-cache condition and must be treated according to repository validation rules.

`runethread index --check` is intentionally strict and fails when expected files are missing, bytes differ, obsolete generated files remain, `index/STALE` exists, or the authoritative source/generated index tree is unsafe.

---

## Optional local acceleration

A future Runethread client may build an uncommitted local SQLite/FTS or similar cache from canonical Git/index data for faster local retrieval. Such a cache would be disposable implementation state, not committed repository format and not an authority source.

Index v2 does not require SQLite, a vector database, embeddings, a server, or any paid service.
