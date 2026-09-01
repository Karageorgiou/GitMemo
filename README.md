# GitMemo

Git-backed persistent external memory for AI assistants.

GitMemo separates **public infrastructure** from **private user memory**:

- this public repository contains the Go CLI, validator, deterministic indexer, canonical operational contract, schemas, templates, tests, and CI infrastructure;
- each user keeps actual memories in a separate private Git repository.

A memory repository is self-describing: it vendors the operational contract that tells an unfamiliar LLM how to retrieve, interpret, and modify its contents safely, while the GitMemo implementation remains here.

> **AI / LLM OPERATORS working on a memory repository:** read and follow `MEMORY_PROTOCOL.md` before retrieving from or modifying memories.

See [`docs/REPOSITORY_ROLES.md`](docs/REPOSITORY_ROLES.md) for the infrastructure/data boundary.

## CLI

GitMemo is a small Go CLI with no third-party Go modules.

```bash
go test ./...
go vet ./...
go build -o gitmemo ./cmd/gitmemo
```

Create a private memory-repository skeleton:

```bash
./gitmemo init ./my-memory
```

Then validate or rebuild its generated indexes:

```bash
./gitmemo validate ./my-memory
./gitmemo index --check ./my-memory
./gitmemo index --write ./my-memory
```

`gitmemo init` refuses to overwrite a non-empty target. It may also initialize a freshly created Git repository whose only existing entry is `.git`.

## Operational contract

The CLI embeds and vendors these files into each initialized memory repository:

- `MEMORY_PROTOCOL.md`
- `schema/memory-item.schema.json`
- `docs/MEMORY_SCHEMA.md`
- `docs/MEMORY_CONTENT_FORMAT.md`
- `docs/TAXONOMY.md`
- `docs/REPOSITORY_VALIDATION.md`
- `templates/`

This makes each private memory repository understandable without network access to this repository.

## Privacy model

GitMemo does not require a server. The CLI runs where the memory repository already exists. User memories do not need to be uploaded to a GitMemo service.

## Status

GitMemo is currently pre-1.0. Repository format and release/versioning workflows are still being hardened before the first public release.
