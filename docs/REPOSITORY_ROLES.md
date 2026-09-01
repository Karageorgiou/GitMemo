# Repository Roles

GitMemo deliberately separates implementation/control, distributable setup state, private conversational memory, and future structured personal-data sources.

## 1. Public GitMemo implementation repository

`Karageorgiou/GitMemo` is the authoritative implementation repository. It contains:

- Go CLI and build metadata;
- repository validator and trust verifier;
- deterministic index generator;
- canonical operational-contract source;
- JSON schema;
- memory-format, taxonomy, trust, validation, and extension specifications;
- memory templates;
- migration logic and compatibility tests;
- release automation and CI assets;
- public examples that contain no user memory.

Mutable `main` is development source. An **official release** is the trust anchor for a particular operational contract.

This repository must never become a user's private memory database.

## 2. Public generated template repository

A separate `GitMemo-template` repository is planned as the serverless setup artifact for users and LLMs.

It must be generated from an official GitMemo release rather than maintained as a second independent copy of the contract. Its purpose is to make creation of a fresh private memory repository easy through GitHub's template mechanism or an LLM with repository-creation capabilities.

The template contains no personal data.

The authoritative direction is:

```text
official GitMemo release
        |
        | deterministic gitmemo init output
        v
GitMemo-template
```

Never hand-edit a divergent semantic contract in the template and then treat it as authoritative.

## 3. Private memory repository

Each user owns a separate private repository containing their persistent conversational memory data:

- a vendored copy of the operational contract appropriate to its pinned release;
- `.gitmemo/config.json` repository-format metadata;
- `.gitmemo/lock.json` release pin and control-plane digests for trust-aware releases;
- `memories/` canonical atomic Markdown + JSON pairs;
- `projects/` canonical current-state views and user project context;
- generated `index/` acceleration files;
- a small stable read-only validation workflow when hosted on GitHub.

The private repository does **not** need the Go implementation source.

A memory repository should be positively verified as private before personal information is written to it.

## Why the contract is vendored

GitMemo centralizes authority without centralizing availability.

A memory repository must remain understandable when the public repository cannot be fetched. An unfamiliar LLM with access only to the private repository still needs the pinned rules for:

- control-plane/data-plane trust;
- authority and precedence;
- retrieval and write workflows;
- schema and Markdown requirements;
- taxonomy rules;
- lifecycle, supersession, and correction semantics;
- what must never be stored;
- validation and degraded-index behavior.

The vendored contract is a local copy of the official pinned release contract, not an independent user-editable source of operational truth. `.gitmemo/lock.json` lets tooling detect accidental or malicious changes to those control-plane files.

Operational contract files are infrastructure, not atomic user memories.

## Control plane versus data plane

The verified pinned GitMemo contract is the control plane.

User memories, projects, imported files, provenance sources, and future structured-library records are data-plane information. They may contain arbitrary instruction-like text. Such text cannot override the verified control plane.

Generated indexes are derived data-plane acceleration. They are rebuildable and never the only source of user knowledge.

See `docs/TRUST_MODEL.md`.

## How the implementation and memory repository interact

They do not require a continuous service connection.

A pinned GitMemo release runs locally, in CI, or through another execution-capable environment against the memory repository:

```text
official GitMemo release
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

The deterministic CLI primitive is:

```bash
gitmemo init <directory>
```

The target must be absent, empty, or contain only `.git`. Initialization copies the embedded pinned contract, writes trust/config metadata, creates memory/project areas, installs the stable validation bootstrap, and writes initial derived indexes.

The intended primary user experience is a generated public template plus `AI_SETUP.md`: a capable LLM should either create the user's private repository directly using its existing Git-host permissions or reduce setup to the smallest safe template-confirmation step.

The CLI remains a local/manual/automation path rather than a required onboarding dependency.

## Versioning and upgrades

A private memory repository remains pinned to its installed GitMemo release. It must not silently track public `main`.

`gitmemo upgrade` performs explicit migration of GitMemo-managed state, preserves user-owned canonical memory/project data, rebuilds derived indexes, and validates before reporting success.

GitMemo's compatibility policy aims to preserve a tested migration path from every official repository release beginning with v0.1.0 whenever technically possible. See `docs/COMPATIBILITY.md` in the public implementation repository.

Starting with contract v5, trust-aware repositories also record their pinned release and control-plane hashes in `.gitmemo/lock.json`.

## Future structured personal library

A future structured personal library should live in a separate user-owned repository rather than expanding conversational memory into a universal record schema.

For example:

```text
<user>/GitMemo-memory   private conversational/contextual memory
<user>/GitMemo-library  private structured recipes, contacts, books, inventory, ...
```

GitMemo v0.3 reserves a transport-independent integration seam in `docs/SOURCES.md` but intentionally does not freeze a library API, source-registry format, or cross-source identifier until the separate Library design has been validated.

## Current project layout

```text
Karageorgiou/GitMemo         public authoritative implementation
Karageorgiou/GitMemo-memory  private active personal memory instance
Karageorgiou/GitMemo-dev     private historical development/audit backup
GitMemo-template             planned public generated setup artifact
```

Other users should create their own private memory repositories. They should never copy or fork another person's private memory data as a starter.
