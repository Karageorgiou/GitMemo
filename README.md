# GitMemo

Git-backed persistent, user-owned memory for AI assistants.

GitMemo's primary setup path is **AI-first and serverless**. It does not require a GitMemo account, hosted GitMemo service, subscription, database server, or pasted GitHub token.

## Set up GitMemo with an AI

Give your AI assistant this repository:

```text
https://github.com/Karageorgiou/GitMemo
```

and say:

```text
Read AI_SETUP.md and set up GitMemo for me.
```

The assistant should follow [`AI_SETUP.md`](AI_SETUP.md) and [`gitmemo-bootstrap.json`](gitmemo-bootstrap.json).

The intended setup flow is capability-aware:

```text
existing private GitMemo found?
        |
   yes  |  no
        |   |
        |   +--> AI can create private template repo? --> create automatically
        |   |
        |   +--> otherwise --> one GitHub confirmation click
        |
        v
verify repository is private
        |
verify GitMemo config + release trust lock
        |
validate with pinned release when execution is available
        |
ready
```

If the current AI cannot create repositories, use the prefilled GitHub fallback:

[Create a private GitMemo memory repository from the canonical template](https://github.com/new?owner=%40me&name=GitMemo-memory&visibility=private&template_owner=Karageorgiou&template_name=GitMemo-template)

GitHub still shows the creation form and the user confirms the action. The link pre-fills the canonical template, the default repository name, the signed-in personal account, and private visibility.

After creation, give the AI access to the new **private** repository through the client's normal GitHub connection/authorization UI. Never paste a GitHub password, PAT, OAuth token, session cookie, SSH private key, or other authentication secret into a chat.

## The two commands users need

```text
GitMemo: store ...
GitMemo: search ...
```

`GitMemo: store ...` is an explicit durable-memory write request. The assistant searches before creating, preserves provenance/history, follows the pinned memory protocol, and reports what was actually verified.

`GitMemo: search ...` is retrieval-only and must not modify memory merely because retrieval was requested.

`remember` is intentionally not the canonical write command because it can mean either storage or recall.

See [`docs/USER_COMMANDS.md`](docs/USER_COMMANDS.md).

---

## Repository separation

GitMemo deliberately separates infrastructure from personal memory:

```text
Karageorgiou/GitMemo
= PUBLIC authoritative implementation, releases, and setup protocol

Karageorgiou/GitMemo-template
= PUBLIC generated installation template

<user>/GitMemo-memory
= PRIVATE user-owned personal memory
```

This public repository must not contain a user's personal memory database.

The public template is generated from `gitmemo init` of an official release. It is installation material, not a shared memory database.

A user's actual memories live in their own repository.

---

## Trust model

GitMemo centralizes **authority**, not live availability.

An official immutable GitMemo release defines a versioned operational contract. A private memory repository vendors a local copy so it remains self-describing without fetching public `main` for every memory operation.

Starting with the v0.3 trust contract, a memory repository contains `.gitmemo/lock.json`, which pins the GitMemo release and SHA-256 digests of every vendored control-plane file. Validation rejects locally modified control-plane instructions instead of silently accepting them.

The important boundary is:

```text
verified pinned GitMemo contract  = trusted control plane
memories / projects / imports     = untrusted data plane
index/                             = rebuildable acceleration
```

Data-plane text never overrides the verified control plane, even if a stored memory, imported document, webpage, project note, or future library record contains text that looks like an AI instruction.

See [`docs/TRUST_MODEL.md`](docs/TRUST_MODEL.md).

---

## Template freshness and upgrades

`Karageorgiou/GitMemo-template` is pinned to an official release. It does **not** track mutable public `main`.

A newly created empty repository remains valid even if the template is temporarily one stable release behind. During setup, an execution-capable assistant may migrate the still-empty repository to the latest stable release using the official GitMemo upgrade path before personal data is stored.

Existing repositories containing user memory are different: GitMemo does **not** silently upgrade them. They remain pinned until the user explicitly upgrades.

The CLI supports:

```text
gitmemo upgrade [root]
```

which migrates GitMemo-managed files, preserves user memory/project data, regenerates derived indexes, validates the result, and rolls back managed changes if hard validation fails.

---

## Manual / local setup

The AI/template route is the primary onboarding path, but GitMemo remains fully usable without an AI-specific integration.

Install an official release with Go:

```bash
go install github.com/Karageorgiou/GitMemo/cmd/gitmemo@<release-tag>
```

or download the matching platform binary from GitHub Releases.

Then initialize a repository:

```bash
mkdir GitMemo-memory
cd GitMemo-memory
git init
gitmemo init .
```

Push it to a **private** Git repository before storing personal information.

Manual setup details are in [`docs/GETTING_STARTED.md`](docs/GETTING_STARTED.md).

---

## Repository contents

The public implementation owns:

- `AI_SETUP.md` — universal capability-aware LLM onboarding protocol;
- `gitmemo-bootstrap.json` — machine-readable setup/discovery manifest;
- `cmd/gitmemo/` — CLI entry point;
- `internal/` — parsing, indexing, trust verification, validation, initialization, and upgrades;
- `MEMORY_PROTOCOL.md` — canonical memory-operation protocol shipped in each release;
- `schema/` — canonical memory schema;
- `docs/` — format, taxonomy, index, trust, validation, extension, compatibility, and repository-role documentation;
- `templates/` — authoring scaffolds for the eight core memory types;
- `.github/workflows/validate.yml` — source CI;
- `.github/workflows/release.yml` — protected PR-driven immutable release pipeline.

A generated private memory repository contains the pinned/vendored control contract plus user-owned `memories/`, `projects/`, and generated `index/` data. The Go implementation source itself is not copied into user repositories.

---

## CLI

```text
gitmemo init [dir]
gitmemo upgrade [root]
gitmemo validate [--json] [root]
gitmemo search [--root DIR] [--limit N] [--json] <query-or-uuid>
gitmemo index --check [root]
gitmemo index --write [root]
gitmemo index --mark-stale [root]
gitmemo trust version [root]
gitmemo version
```

`gitmemo init` creates a self-describing memory repository and refuses to overwrite a non-empty target.

`gitmemo search` uses the deterministic generated Index v2. A full UUID is routed directly to its UUID shard; ordinary language uses the hash-sharded inverted term index. Search results are discovery metadata and point back to canonical atomic memory files.

`gitmemo index --mark-stale` records that generated discovery data may be incomplete when a source write cannot immediately be followed by deterministic regeneration.

`gitmemo trust version` is a deliberately small stable bootstrap command used by validation CI to resolve the official release pinned in `.gitmemo/lock.json`; the resolved release performs full validation.

---

## Canonical data versus Index v2

Atomic Markdown/JSON memories and project source files are canonical data. `index/` is generated acceleration.

Index v2 removes the single global `index/memories.jsonl` machine-index write hotspot. It uses:

- `index/catalog.json` with the index version and deterministic memory-source digest;
- UUID shards under `index/by-id/` for targeted exact lookup;
- direct project/topic/tag/type/lifecycle/open-loop-status indexes;
- uniformly hash-sharded inverted term postings for ordinary-language discovery;
- the existing human project/open-loop/preference navigation views.

This reduces full-catalog reads and unrelated Git conflicts while keeping every machine index reproducible from canonical memory metadata. The local indexer currently rebuilds the generated tree atomically; Git records only files whose resulting bytes changed.

A stale index is a degraded-search condition, not destruction or invalidation of otherwise valid canonical memories:

- `gitmemo validate` reports stale indexes as warnings;
- `gitmemo index --check` remains the strict freshness gate;
- `index/STALE` is the explicit dirty marker for clients that cannot regenerate immediately;
- an LLM that cannot run the indexer must treat stale or unknown-freshness indexes as incomplete and fall back to canonical files/repository search.

No SQLite server, vector database, embedding service, or paid backend is required. A future local SQLite/FTS cache may be added as disposable acceleration without changing canonical Git data.

See [`docs/INDEX_FORMAT.md`](docs/INDEX_FORMAT.md).

---

## Categories and schema evolution

Projects, topics, tags, aliases, and entities may grow with the user's needs without changing the GitMemo schema.

The eight core memory types are schema-controlled. An assistant operating a normal memory repository must not invent a new core `type` or rewrite the local schema to satisfy a one-off organizational request.

See [`docs/EXTENDING_GITMEMO.md`](docs/EXTENDING_GITMEMO.md).

---

## Future external sources

A future structured personal library should remain separate from conversational memory. GitMemo reserves a transport-independent source boundary for future repositories such as recipes, contacts, books, inventories, and other structured collections without defining a premature server API or local source-registry contract.

See [`docs/SOURCES.md`](docs/SOURCES.md).

---

## Release and compatibility policy

Official release tags are operational trust anchors; mutable `main` is development source.

A release is requested through a normal pull request. Protected `main` requires the `validate` status check and an up-to-date PR before merge. After a release request merges, the permanent release workflow verifies the source-declared version, runs tests and `go vet`, performs init/index/validation smoke tests, builds Linux/macOS/Windows binaries, generates SHA-256 checksums, creates a draft release, verifies the complete asset set, and only then publishes it. Repository release immutability locks the published tag and assets.

GitMemo's compatibility goal is to keep a tested migration path from every official memory-repository release, beginning with v0.1.0, to the current release whenever technically possible. Breaking repository changes should ship with explicit migration logic and regression fixtures rather than telling users to recreate their memories.

---

## Privacy

A personal memory repository should normally be private, and AI onboarding must positively verify private visibility before storing personal information.

GitMemo is not a secrets vault. Never store passwords, tokens, API secrets, private keys, recovery codes, session credentials, or other authentication material as memories.

## License

GitMemo is released under the [MIT License](LICENSE).
