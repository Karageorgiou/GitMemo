# Repository Roles

GitMemo uses two deliberately different repository roles.

## 1. Public infrastructure repository

The public GitMemo repository contains the implementation and canonical format contract:

- Go CLI and build metadata;
- repository validator;
- deterministic index generator;
- canonical `MEMORY_PROTOCOL.md`;
- JSON schema;
- memory-format, taxonomy, and validation specifications;
- memory templates;
- tests;
- release automation and CI assets;
- public examples that contain no user memory.

It must not contain a user's private memory database.

Architecture and development history belong in normal project documentation or ADRs, not in user-memory storage.

## 2. Private memory repository

Each user owns a separate private repository containing their persistent memory data:

- a vendored copy of the operational contract appropriate to that repository version;
- `.gitmemo/config.json` repository-format metadata;
- `memories/` atomic Markdown + JSON pairs;
- `projects/` current-state views;
- generated `index/` files;
- a small CI workflow when hosted on GitHub.

The private repository does **not** need the Go implementation source.

## Why the contract is vendored

A memory repository must be self-describing. An unfamiliar LLM with access only to that private repository still needs to know:

- authority and precedence rules;
- retrieval and write workflows;
- schema and Markdown requirements;
- taxonomy rules;
- lifecycle, supersession, and correction semantics;
- what must never be stored;
- when validation is required.

Therefore the operational contract lives with the memory instance even though the implementation lives elsewhere.

Operational contract files are infrastructure, not atomic user memories.

## How the repositories interact

They do not maintain a continuous connection.

The GitMemo CLI runs locally or inside CI **against the checked-out private memory repository**:

```text
public GitMemo release
        |
        | executable/tooling
        v
private memory checkout
        |
        +-- validate
        +-- index --check
        +-- index --write
```

The public project does not need to receive or store the private memory contents.

## Creating a memory repository

The CLI supports:

```bash
gitmemo init <directory>
```

The target must be absent, empty, or contain only `.git`. Initialization copies the embedded operational contract, creates the memory/project areas and repository metadata, and writes deterministic empty indexes.

GitHub users may also use a dedicated starter/template flow once the release workflow is finalized.

## Versioning

A private memory repository should use a pinned GitMemo release for CI and upgrades. It should not silently track the public repository's `main` branch.

The initial repository-format metadata is intentionally small:

```json
{
  "repository_format": 1,
  "schema_version": 1,
  "contract_version": 1
}
```

A future explicit upgrade command may update the vendored contract, migrate repository data when necessary, regenerate indexes, and validate before changes are considered complete.

## Current layout

The canonical hosted layout is:

```text
Karageorgiou/GitMemo         public infrastructure
Karageorgiou/GitMemo-memory  private personal memory instance
```

Other users should create their own private memory repositories rather than forking a user's private memory data.
