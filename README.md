# GitMemo

Git-backed persistent external memory for AI assistants.

GitMemo separates **public infrastructure** from **private user memory**:

- this public repository contains the Go CLI, validator, deterministic indexer, canonical operational contract, schemas, templates, tests, and CI infrastructure;
- each user keeps actual memories in a separate private Git repository.

A memory repository is self-describing: it vendors the operational contract that tells an unfamiliar LLM how to retrieve, interpret, and modify its contents safely, while the GitMemo implementation remains here.

> **AI / LLM OPERATORS working on a memory repository:** read and follow `MEMORY_PROTOCOL.md` and `docs/USER_COMMANDS.md` before retrieving from or modifying memories.

## Quick start

Install the first public release:

```bash
go install github.com/Karageorgiou/GitMemo/cmd/gitmemo@v0.1.0
```

Create your own private memory repository:

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
gitmemo validate ./my-memory
gitmemo index --check ./my-memory
gitmemo index --write ./my-memory
```

`gitmemo init` refuses to overwrite a non-empty target. It may initialize a freshly created Git repository whose only existing entry is `.git`.

New memory repositories include a GitHub Actions validation workflow pinned to the public GitMemo v0.1.0 release.

## Operational contract

The CLI embeds and vendors these files into each initialized memory repository:

- `MEMORY_PROTOCOL.md`
- `schema/memory-item.schema.json`
- `docs/MEMORY_SCHEMA.md`
- `docs/MEMORY_CONTENT_FORMAT.md`
- `docs/TAXONOMY.md`
- `docs/REPOSITORY_VALIDATION.md`
- `docs/USER_COMMANDS.md`
- `templates/`

This makes each private memory repository understandable without network access to this repository.

See [`docs/REPOSITORY_ROLES.md`](docs/REPOSITORY_ROLES.md) for the infrastructure/data boundary.

## Privacy model

GitMemo does not require a server. The CLI runs where the memory repository already exists. User memories do not need to be uploaded to a GitMemo service.

A memory repository should normally be private. GitMemo is not a secrets vault; authentication secrets must not be stored as memories.

## License

GitMemo is released under the [MIT License](LICENSE).

## Status

GitMemo v0.1.0 is the first public preview. The V1 repository/schema format remains intentionally conservative while real-world usage is evaluated.
