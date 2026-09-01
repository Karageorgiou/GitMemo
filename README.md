# GitMemo

Git-backed persistent external memory for AI assistants.

GitMemo separates **public infrastructure** from **private user memory**:

- this public repository contains the Go CLI, validator, deterministic indexer, canonical operational contract, schemas, templates, tests, and release infrastructure;
- each user keeps actual memories in a separate private Git repository.

The long-term goal is simple: a user should be able to give a capable LLM this public repository and say **“set up GitMemo for me”**. A generated public template and explicit AI setup protocol will provide that serverless onboarding path. GitMemo itself does not require an account, hosted service, database server, or paid backend.

## Trust model

GitMemo centralizes **authority**, not availability.

An official GitMemo release defines one immutable operational contract. A private memory repository vendors a local copy so it remains self-describing and usable without fetching public `main` on every operation.

Starting with the v0.3 trust contract, a memory repository also contains `.gitmemo/lock.json`, which pins the GitMemo release and SHA-256 digests of every vendored control-plane file. Validation rejects locally modified control-plane instructions instead of silently accepting them.

The important boundary is:

```text
verified pinned GitMemo contract  = control plane
memories / projects / imports     = data plane
index/                             = rebuildable acceleration
```

Data-plane text is never allowed to override the verified control plane, even when a stored memory, imported document, webpage, project note, or future library record contains text that looks like an AI instruction. See [`docs/TRUST_MODEL.md`](docs/TRUST_MODEL.md).

## Quick start today

The current stable public release remains available from GitHub Releases. The source tree is preparing the v0.3 trust foundation used by the upcoming universal AI/template onboarding work.

Install a stable release with Go:

```bash
go install github.com/Karageorgiou/GitMemo/cmd/gitmemo@<release-tag>
```

or download a platform binary from the matching GitHub release.

Create a memory repository:

```bash
mkdir GitMemo-memory
cd GitMemo-memory
git init
gitmemo init .
```

Push it to a **private** Git repository that your AI assistant can access. Do not store personal information until repository visibility has been positively verified as private.

The current manual setup documentation is in [`docs/GETTING_STARTED.md`](docs/GETTING_STARTED.md). AI-first setup will become the primary path once the generated `GitMemo-template` repository exists.

## User command convention

GitMemo defines two primary natural-language commands:

```text
GitMemo: store ...
GitMemo: search ...
```

`GitMemo: store ...` is an explicit durable memory-write request. The assistant searches before creating, preserves provenance and history, validates canonical data as far as its tools permit, and reports what was actually verified.

`GitMemo: search ...` is retrieval-only and must not modify memory merely because retrieval was requested.

`remember` is intentionally not the canonical write command because it can mean either storing or recalling information.

See [`docs/USER_COMMANDS.md`](docs/USER_COMMANDS.md).

## Repository contents

The public implementation owns:

- `cmd/gitmemo/` — CLI entry point;
- `internal/` — memory parsing, indexing, trust verification, validation, initialization, and upgrades;
- `MEMORY_PROTOCOL.md` — canonical LLM operating protocol;
- `schema/` — canonical memory schema;
- `docs/` — format, taxonomy, trust, validation, extension, and repository-role contracts;
- `templates/` — authoring scaffolds for the eight core memory types;
- `.github/workflows/validate.yml` — source CI;
- `.github/workflows/release.yml` — permanent tag-driven release pipeline.

A generated private memory repository contains the pinned/vendored control contract plus user-owned `memories/`, `projects/`, and generated `index/` data. The implementation source itself is not copied into user repositories.

## CLI

```text
gitmemo init [dir]
gitmemo upgrade [root]
gitmemo validate [--json] [root]
gitmemo index --check [root]
gitmemo index --write [root]
gitmemo trust version [root]
gitmemo version
```

`gitmemo init` creates a self-describing memory repository and refuses to overwrite a non-empty target.

`gitmemo upgrade` explicitly migrates GitMemo-managed repository files to the contract embedded in the running release, preserves user memory/project data, regenerates derived indexes, validates the result, and rolls back its managed changes if hard validation fails.

Existing repositories remain pinned until explicitly upgraded. GitMemo does not silently track public `main`.

`gitmemo trust version` is a deliberately small stable bootstrap command used by validation CI to resolve the release pinned in `.gitmemo/lock.json`; the resolved release performs full validation.

## Canonical data versus indexes

Atomic Markdown/JSON memories and project source files are canonical data. `index/` is generated acceleration.

A stale index is therefore a degraded-search condition, not destruction or invalidation of otherwise valid canonical memories:

- `gitmemo validate` reports stale indexes as warnings;
- `gitmemo index --check` remains the strict freshness gate;
- an LLM that cannot run the indexer must treat stale indexes as incomplete, fall back to canonical files/repository search, and report that regeneration remains pending.

This distinction allows GitMemo to work with LLM clients that can read/write Git repositories but cannot execute local binaries.

## Categories and schema evolution

Projects, topics, tags, aliases, and entities may grow with the user's needs without changing the GitMemo schema.

The eight core memory types are schema-controlled. An assistant operating a normal memory repository must not invent a new core `type` or rewrite the local schema to satisfy a one-off organizational request. See [`docs/EXTENDING_GITMEMO.md`](docs/EXTENDING_GITMEMO.md).

## Future external sources

A future structured personal library should remain separate from conversational memory. GitMemo v0.3 reserves a transport-independent source boundary for future repositories such as recipes, contacts, books, inventories, and other structured collections without defining a premature server API or local source-registry contract. See [`docs/SOURCES.md`](docs/SOURCES.md).

## Release and compatibility policy

Release tags are the operational trust anchors; mutable `main` is development source.

The permanent release workflow verifies that a tag matches the source-declared version, runs tests and `go vet`, performs an init/index/validation smoke test, builds binaries for Linux/macOS/Windows, generates SHA-256 checksums, and publishes the release.

GitMemo's compatibility goal is to keep a tested migration path from every official memory-repository release, beginning with v0.1.0, to the current release whenever technically possible. Breaking repository changes must therefore ship with explicit migration logic and regression fixtures rather than telling users to recreate their memories.

## Privacy

GitMemo is not a secrets vault. Never store passwords, tokens, API secrets, private keys, recovery codes, session credentials, or other authentication material as memories.

A personal memory repository should normally be private.

## License

GitMemo is released under the [MIT License](LICENSE).
