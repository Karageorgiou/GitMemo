# Runethread

Git-backed, user-owned persistent memory for AI assistants.

Runethread keeps durable memory in ordinary files and Git history. The canonical memory repository is owned by the user, remains portable, and can be read by assistants without depending on a hosted Runethread account or database service.

## Set up Runethread with an AI

Give your AI assistant this repository:

```text
https://github.com/runethread/core
```

and say:

```text
Read AI_SETUP.md and set up Runethread for me.
```

The assistant should follow [`AI_SETUP.md`](AI_SETUP.md) and [`runethread-bootstrap.json`](runethread-bootstrap.json).

The intended setup flow is capability-aware:

```text
existing private Runethread repository found?
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
verify .runethread config + release trust lock
        |
validate with the pinned release when execution is available
        |
ready
```

If the current AI cannot create repositories, use the prefilled GitHub fallback:

[Create a private Runethread memory repository from the canonical template](https://github.com/new?owner=%40me&name=runethread-memory&visibility=private&template_owner=runethread&template_name=memory-template)

GitHub still shows the creation form and the user confirms the action. After creation, give the AI access to the new **private** repository through the client's normal GitHub connection/authorization UI. Never paste a GitHub password, PAT, OAuth token, session cookie, SSH private key, or other authentication secret into a chat.

## The two commands users need

```text
Runethread: store ...
Runethread: search ...
```

`Runethread: store ...` is an explicit durable-memory write request. The assistant searches before creating, preserves provenance and history, follows the pinned memory protocol, and reports what was actually verified.

`Runethread: search ...` is retrieval-only and must not modify memory merely because retrieval was requested.

See [`docs/USER_COMMANDS.md`](docs/USER_COMMANDS.md).

---

## Repository separation

Runethread separates infrastructure from personal memory:

```text
runethread/core
= PUBLIC authoritative implementation, releases, and setup protocol

runethread/memory-template
= PUBLIC generated installation template

<user>/runethread-memory
= PRIVATE user-owned personal memory
```

This public repository must not contain a user's personal memory database. The public template is installation material, not a shared memory database. A user's actual memories live in their own repository.

---

## Native repository format and trust

Runethread v0.6.0 uses native managed metadata under:

```text
.runethread/config.json
.runethread/lock.json
```

The lock identifies `runethread/core`, pins the Runethread release, and records SHA-256 digests of every vendored control-plane file. The verified pinned contract is trusted control-plane material; memories, project files, imports, and other user data are data-plane content and cannot override it merely by containing instruction-like text.

```text
verified pinned Runethread contract = trusted control plane
memories / projects / imports       = untrusted data plane
index/                               = rebuildable acceleration
```

See [`docs/TRUST_MODEL.md`](docs/TRUST_MODEL.md).

---

## Template freshness and upgrades

`runethread/memory-template` is pinned to an official release rather than mutable `main`.

A newly created empty repository may be upgraded to the latest stable release during setup when execution-capable tooling is available. Existing repositories containing user memory are not silently upgraded; they remain pinned until the user explicitly requests an upgrade.

The native CLI supports:

```text
runethread upgrade [root]
```

The upgrader snapshots managed/generated state, applies only a supported migration, rebuilds indexes, validates the resulting repository, and restores the snapshot if a hard post-write check fails.

Runethread v0.6.0 also contains one deliberately narrow predecessor migration: an exact trusted GitMemo v0.5.0 repository (`repository_format` 1, schema 1, contract 6, lock 1) can be migrated to the native Runethread format. Unknown, mixed, or tampered legacy control state is refused rather than guessed. Historical GitMemo releases remain immutable history; the legacy name is not a second native Runethread interface.

---

## Manual / local setup

Install an official Runethread release with Go:

```bash
go install github.com/runethread/core/cmd/runethread@<release-tag>
```

or download the matching platform binary from GitHub Releases.

Then initialize a repository:

```bash
mkdir runethread-memory
cd runethread-memory
git init
runethread init .
```

Push it to a **private** Git repository before storing personal information.

Manual setup details are in [`docs/GETTING_STARTED.md`](docs/GETTING_STARTED.md).

---

## Repository contents

The public implementation owns:

- `AI_SETUP.md` — capability-aware LLM onboarding protocol;
- `runethread-bootstrap.json` — machine-readable setup/discovery manifest;
- `cmd/runethread/` — CLI entry point;
- `internal/` — parsing, indexing, trust verification, validation, initialization, and upgrades;
- `MEMORY_PROTOCOL.md` — canonical memory-operation protocol shipped in each release;
- `schema/` — canonical memory schema;
- `docs/` — format, taxonomy, index, trust, validation, extension, compatibility, and repository-role documentation;
- `templates/` — authoring scaffolds for the eight core memory types;
- `.github/workflows/validate.yml` — source CI;
- `.github/workflows/release.yml` — release pipeline.

A generated private memory repository contains the pinned/vendored operational contract plus user-owned `memories/`, `projects/`, and generated `index/` data. The Go implementation source itself is not copied into user repositories.

---

## CLI

```text
runethread init [dir]
runethread upgrade [root]
runethread validate [--json] [root]
runethread search [--root DIR] [--limit N] [--json] <query-or-uuid>
runethread index --check [root]
runethread index --write [root]
runethread index --mark-stale [root]
runethread trust version [root]
runethread version
```

`runethread init` creates a self-describing native repository and refuses to overwrite a non-empty target.

`runethread search` uses deterministic Index v2. A full UUID is routed directly to its UUID shard; ordinary language uses the hash-sharded inverted term index. Search results are discovery metadata and point back to canonical atomic memory files.

`runethread index --mark-stale` records that generated discovery data may be incomplete when a source write cannot immediately be followed by deterministic regeneration.

`runethread trust version` is the small stable bootstrap command used by validation CI to resolve the official release pinned in `.runethread/lock.json`; the resolved release performs full validation.

---

## Canonical data versus Index v2

Atomic Markdown/JSON memories and project source files are canonical data. `index/` is generated acceleration.

Index v2 uses:

- `index/catalog.json` with the index version and deterministic memory-source digest;
- UUID shards under `index/by-id/` for targeted exact lookup;
- direct project/topic/tag/type/lifecycle/open-loop-status indexes;
- uniformly hash-sharded inverted term postings for ordinary-language discovery;
- human project/open-loop/preference navigation views.

A stale index is a degraded-search condition, not loss of canonical memory:

- `runethread validate` may report stale indexes as warnings;
- `runethread index --check` is the strict freshness gate;
- `index/STALE` is the explicit dirty marker for clients that cannot regenerate immediately;
- clients must fall back to canonical files or repository search when index freshness is unknown.

No SQLite server, vector database, embedding service, or paid backend is required. Future disposable acceleration may be added without changing canonical Git data.

See [`docs/INDEX_FORMAT.md`](docs/INDEX_FORMAT.md).

---

## Categories and schema evolution

Projects, topics, tags, aliases, and entities may grow with the user's needs without changing the Runethread schema.

The eight core memory types are schema-controlled. An assistant operating a normal memory repository must not invent a new core `type` or rewrite the local schema to satisfy a one-off organizational request.

See [`docs/EXTENDING_RUNETHREAD.md`](docs/EXTENDING_RUNETHREAD.md).

---

## Release and compatibility policy

Official release tags are operational trust anchors; mutable `main` is development source.

The v0.6.0 cutover deliberately preserves old Git history, tags, and release artifacts rather than rewriting them. Native Runethread compatibility begins at repository format 2. The one supported legacy bridge in v0.6.0 is the explicitly tested GitMemo v0.5.0 -> Runethread v0.6.0 migration described above and in [`docs/runethread/MIGRATION.md`](docs/runethread/MIGRATION.md).

Future breaking repository changes should ship with explicit migration logic and regression fixtures for supported Runethread releases rather than requiring users to recreate canonical memory.

---

## Privacy

A personal memory repository should normally be private, and onboarding must positively verify private visibility before storing personal information.

Runethread is not a secrets vault. Never store passwords, tokens, API secrets, private keys, recovery codes, session credentials, or other authentication material as memories.

## License

Runethread is released under the [MIT License](LICENSE).
