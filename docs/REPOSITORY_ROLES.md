# Repository Roles

Runethread separates implementation/control, distributable setup state, private conversational memory, and future structured personal-data sources.

## 1. Public implementation repository

`runethread/core` is the authoritative implementation repository. It contains the Go CLI, validator/trust verifier, deterministic index generator, canonical operational-contract source, JSON schema, templates, migration logic/tests, release automation, and public examples without user memory.

Mutable `main` is development source. An official immutable release is the trust anchor for a particular operational contract.

This repository must never become a user's private memory database.

## 2. Public generated template repository

`runethread/memory-template` is the intended serverless setup artifact for users and LLMs.

It must be generated from an official Runethread release rather than maintained as a divergent second contract. Its direction of authority is:

```text
official Runethread release
        |
        | deterministic runethread init output
        v
runethread/memory-template
```

The template contains no personal data and should not be hand-edited into an independent semantic contract.

## 3. Private memory repository

Each user owns a separate private repository containing:

- a vendored operational contract pinned to its official release;
- `.runethread/config.json` compatibility metadata;
- `.runethread/lock.json` release/source pin and control-plane digests;
- `memories/` canonical Markdown + JSON pairs;
- `projects/` canonical current-state views and user project context;
- generated `index/` acceleration files;
- a small read-only validation workflow when hosted on GitHub.

The private repository does not need the Go implementation source.

It should be positively verified as private before personal information is written.

## Why the contract is vendored

Runethread centralizes authority without centralizing availability. A private repository remains understandable when the public repository cannot be fetched.

The vendored contract preserves rules for trust boundaries, authority/precedence, retrieval/write workflows, schema/Markdown formats, taxonomy, lifecycle/correction semantics, forbidden secrets, validation, and degraded-index behavior.

It is a hash-verified local copy of the official pinned release contract, not an independent user-editable source of operational truth.

## Control plane versus data plane

The verified pinned Runethread contract is the control plane.

User memories, projects, imports, provenance sources, and future structured-library records are data-plane information. They may contain arbitrary instruction-like text but cannot override the verified control plane.

Generated indexes are rebuildable data-plane acceleration and never the sole source of user knowledge.

## Implementation interaction

No continuous Runethread service connection is required:

```text
official Runethread release
        |
        | executable/tooling
        v
private memory checkout
        |
        +-- trust verification
        +-- validate
        +-- index --check
        +-- index --write
        +-- upgrade
```

The public project does not need to receive or store private memory contents.

## Creating a memory repository

The deterministic primitive is:

```bash
runethread init <directory>
```

The target must be absent, empty, or contain only `.git`. Initialization copies the embedded pinned contract, writes `.runethread` trust/config metadata, creates memory/project areas, installs validation bootstrap, and writes initial derived indexes.

The intended primary user experience is the public template plus `AI_SETUP.md`; the CLI remains a local/manual/automation path.

## Versioning and migration

A native repository remains pinned to its installed Runethread release and does not silently track public `main`.

`runethread upgrade` performs explicit supported migration, preserves user-owned canonical data, rebuilds derived indexes, validates, and rolls back managed/generated state on hard failure.

Runethread v0.6.0 additionally provides the finite trusted GitMemo v0.5.0 predecessor bridge documented in `docs/runethread/MIGRATION.md`. Legacy `.gitmemo` metadata is migration input only, not a second native format.

See `docs/COMPATIBILITY.md`.

## Future structured personal library

A future structured personal library should live in a separate user-owned repository rather than expanding conversational memory into a universal schema. `docs/SOURCES.md` reserves a transport-independent integration seam but does not yet freeze a source-registry or cross-source identifier contract.

## Current target layout

```text
runethread/core             public authoritative implementation
runethread/memory-template  public generated setup artifact
<user>/runethread-memory    private user-owned memory instance
```

Other users should create their own private memory repositories; they should never use another person's private memory data as a starter.
