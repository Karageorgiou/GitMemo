# GitMemo

Git-backed persistent external memory for AI assistants.

GitMemo separates **public infrastructure** from **private user memory**:

- this public repository contains the Go CLI, validator, deterministic indexer, canonical operational contract, schemas, templates, tests, and CI infrastructure;
- each user keeps actual memories in a separate private Git repository.

A memory repository is self-describing: it vendors the operational contract that tells an unfamiliar LLM how to retrieve, interpret, and modify its contents safely, while the GitMemo implementation remains here.

> **AI / LLM OPERATORS working on a memory repository:** read and follow `MEMORY_PROTOCOL.md` and `docs/USER_COMMANDS.md` before retrieving from or modifying memories.

## Quick start

Install GitMemo with Go:

```bash
go install github.com/Karageorgiou/GitMemo/cmd/gitmemo@v0.2.0
```

Or download the binary for your platform from the GitHub release.

On Windows, double-clicking the v0.2.0 executable launches a small first-run wizard. It asks where to create `GitMemo-memory`, initializes the GitMemo repository skeleton, initializes local Git when Git is available, and prints the remaining private-remote/AI-access steps.

From a terminal, create your own private memory repository explicitly:

```bash
mkdir GitMemo-memory
cd GitMemo-memory
git init
gitmemo init .
```

Then commit it and push it to a **private** Git repository that your AI assistant can access.

The complete end-to-end setup, including GitHub CLI commands, is in [`docs/GETTING_STARTED.md`](docs/GETTING_STARTED.md).

## User command convention

GitMemo defines two primary natural-language commands for use with an AI assistant:

```text
GitMemo: store ...
GitMemo: search ...
```

`GitMemo: store ...` is an explicit durable memory-write request. The assistant must search before creating, preserve provenance/history, regenerate indexes, validate, and report the verified result.

`GitMemo: search ...` is retrieval-only. It searches the user's private memory repository and must not modify or commit memory merely because a search was requested.

`remember` is intentionally not the canonical write command because it can mean either storing or recalling information.

The full command contract, including repository discovery and failure behavior, is in [`docs/USER_COMMANDS.md`](docs/USER_COMMANDS.md).

## CLI

GitMemo is a small Go CLI with no third-party Go modules.

```bash
go test ./...
go vet ./...
go build -o gitmemo ./cmd/gitmemo
```

Commands:

```bash
gitmemo init ./my-memory
gitmemo upgrade ./my-memory
gitmemo validate ./my-memory
gitmemo index --check ./my-memory
gitmemo index --write ./my-memory
gitmemo version
```

`gitmemo init` refuses to overwrite a non-empty target. It may initialize a freshly created Git repository whose only existing entry is `.git`.

`gitmemo upgrade` upgrades the GitMemo-managed contract/config/validation files of an existing memory repository to the contract embedded in the running release, regenerates indexes, validates the result, and rolls back its managed changes if repository validation fails. It does not rewrite user memories or project data.

Existing memory repositories remain pinned to their installed GitMemo release until an explicit upgrade is performed.

## Categories and schema evolution

GitMemo intentionally allows flexible retrieval categories without allowing arbitrary schema drift.

Projects, topics, tags, aliases, and entities can grow with the user's needs. A request such as “store this under topic `home-automation`” normally needs no GitMemo system change.

The eight core memory `type` values are different: adding a ninth type can affect schema, templates, validation, indexing, and migrations. Normal memory-repository assistants must not invent custom core types locally. See [`docs/EXTENDING_GITMEMO.md`](docs/EXTENDING_GITMEMO.md).

## Operational contract

The CLI embeds and vendors these files into each initialized memory repository:

- `MEMORY_PROTOCOL.md`
- `schema/memory-item.schema.json`
- `docs/MEMORY_SCHEMA.md`
- `docs/MEMORY_CONTENT_FORMAT.md`
- `docs/TAXONOMY.md`
- `docs/REPOSITORY_VALIDATION.md`
- `docs/USER_COMMANDS.md`
- `docs/EXTENDING_GITMEMO.md`
- `templates/`

This makes each private memory repository understandable without network access to this repository.

See [`docs/REPOSITORY_ROLES.md`](docs/REPOSITORY_ROLES.md) for the infrastructure/data boundary.

## Privacy model

GitMemo does not require a server. The CLI runs where the memory repository already exists. User memories do not need to be uploaded to a GitMemo service.

A memory repository should normally be private. GitMemo is not a secrets vault; authentication secrets must not be stored as memories.

## License

GitMemo is released under the [MIT License](LICENSE).

## Status

GitMemo v0.2.0 adds explicit repository upgrades and a first-run executable wizard while keeping the V1 memory schema conservative.
