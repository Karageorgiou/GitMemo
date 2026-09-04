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
verify .runethread config + contract-release trust lock
        |
validate with the pinned contract release when execution is available
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

Runethread uses native managed metadata under:

```text
.runethread/config.json
.runethread/lock.json
```

Under contract v8, the lock identifies `runethread/core`, pins the immutable **contract release** in `runethread_version`, and records SHA-256 digests of every vendored control-plane file. The runtime/distribution release is a separate identity. A newer runtime can operate against an unchanged repository only when it embeds the exact pinned contract release and all compatibility dimensions/digests match; the repository is not repinned merely to record that newer executable version.

The verified pinned contract is trusted control-plane material; memories, project files, imports, and other user data are data-plane content and cannot override it merely by containing instruction-like text.

```text
verified pinned Runethread contract = trusted control plane
memories / projects / imports       = untrusted data plane
index/                               = rebuildable acceleration
```

Contract v8 also treats repository-owned authoritative filesystem paths as fail-closed: canonical/control-plane/index-source directories must be real directories, authoritative files must be regular files, and symbolic links or unsupported special objects are rejected rather than followed for those inputs.

See [`docs/TRUST_MODEL.md`](docs/TRUST_MODEL.md).

---

## Template freshness and upgrades

`runethread/memory-template` is pinned to an official contract release rather than mutable `main`.

A newly created empty repository may be explicitly upgraded to the latest stable contract during setup when execution-capable tooling is available. Existing repositories containing user memory are not silently contract-upgraded; they remain pinned until the user explicitly requests an upgrade.

A newer runtime-only release that embeds the same contract requires no repository commit or lock repin.

The native CLI supports:

```text
runethread upgrade [root]
```

The upgrader snapshots managed/generated state, applies only a supported migration, rebuilds indexes, validates the resulting repository, and restores the snapshot if a hard post-write check fails.

Runethread v0.8.0 introduces contract version 8 and explicit runtime/contract-release separation. It recognizes the exact trusted native v0.6.0 and v0.7.0 contract-v7 source anchors and migrates them to the v0.8.0 contract while preserving canonical memory/project bytes. The migration also adds the managed root `.gitattributes` support file when absent and refuses to overwrite conflicting custom Git attributes. The deliberately narrow GitMemo predecessor bridge remains available for the exact trusted GitMemo v0.5.0 state. Unknown, mixed, newer-unknown, customized, tampered, or unsafe authoritative source state is refused rather than guessed.

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
- `MEMORY_PROTOCOL.md` — canonical memory-operation protocol shipped in each contract release;
- `schema/` — canonical memory schema;
- `docs/` — format, taxonomy, index, trust, validation, extension, compatibility, and repository-role documentation;
- `templates/` — authoring scaffolds for the eight core memory types;
- `.github/workflows/validate.yml` — source CI;
- `.github/workflows/release.yml` — release pipeline.

A generated private memory repository contains the pinned/vendored operational contract plus user-owned `memories/`, `projects/`, and generated `index/` data. Fresh v8 repositories also include a managed `.gitattributes` support file for byte-stable LF checkouts. The Go implementation source itself is not copied into user repositories.

---

## CLI

```text
runethread init [dir]
runethread upgrade [root]
runethread validate [--json] [root]
runethread search [--root DIR] [--limit N] [--json] <query-or-uuid>
runethread get [--root DIR] [--json] <uuid>
runethread prepare [--root DIR] [--json] [--request FILE|-]
runethread apply [--root DIR] [--json] [--request FILE|-]
runethread withdraw [--root DIR] [--json] [--request FILE|-]
runethread status [--root DIR] [--json]
runethread index --check [root]
runethread index --write [root]
runethread index --mark-stale [root]
runethread trust version [root]
runethread version
```

`runethread init` creates a self-describing native repository and refuses to overwrite a non-empty target.

`runethread search` uses deterministic Index v2. A full UUID is routed directly to its UUID shard; ordinary language uses the hash-sharded inverted term index. Search results are discovery metadata and point back to canonical atomic memory files.

The Phase 2 MemoryService commands provide the deterministic automation boundary for retrieval and writes. `prepare` is read-only and returns an expected Git revision; `apply` and `withdraw` validate in an isolated worktree and publish only if that revision is still current. See [`docs/runethread/MEMORY_SERVICE.md`](docs/runethread/MEMORY_SERVICE.md).

`runethread index --mark-stale` records that generated discovery data may be incomplete when a source write cannot immediately be followed by deterministic regeneration.

`runethread trust version` is the small stable bootstrap command used by validation CI to resolve the official **contract release** pinned in `.runethread/lock.json`; the resolved contract release performs full validation.

`runethread version` reports the runtime/distribution release. `runethread status` reports runtime and contract release identities separately.

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
- `runethread index --check` is the strict freshness/integrity gate;
- `index/STALE` is the explicit dirty marker for clients that cannot regenerate immediately;
- clients must fall back to valid canonical files or repository search when index freshness is unknown.

No SQLite server, vector database, embedding service, or paid backend is required. Future disposable acceleration may be added without changing canonical Git data.

See [`docs/INDEX_FORMAT.md`](docs/INDEX_FORMAT.md).

---

## Categories and schema evolution

Projects, topics, tags, aliases, and entities may grow with the user's needs without changing the Runethread schema.

The eight core memory types are schema-controlled. An assistant operating a normal memory repository must not invent a new core `type` or rewrite the local schema to satisfy a one-off organizational request.

See [`docs/EXTENDING_RUNETHREAD.md`](docs/EXTENDING_RUNETHREAD.md).

---

## Release and compatibility policy

Official contract-release tags are immutable operational trust anchors; mutable `main` is development source. Runtime releases are distribution identities and may advance independently when the embedded contract is unchanged.

The v0.6.0 cutover deliberately preserves old Git history, tags, and release artifacts rather than rewriting them. Native Runethread compatibility begins at repository format 2. Historical migration support is explicit and tested rather than inferred from mutable current generators.

Future breaking repository-contract changes should ship with explicit migration logic and regression fixtures for supported historical source anchors rather than requiring users to recreate canonical memory. Runtime-only releases should not churn unchanged memory repositories.

See [`docs/COMPATIBILITY.md`](docs/COMPATIBILITY.md).

---

## Privacy

A personal memory repository should normally be private, and onboarding must positively verify private visibility before storing personal information.

Runethread is not a secrets vault. Never store passwords, tokens, API secrets, private keys, recovery codes, session credentials, or other authentication material as memories.

## License

Runethread is released under the [MIT License](LICENSE).
