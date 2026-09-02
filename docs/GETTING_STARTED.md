# Getting started with Runethread

Runethread gives an AI assistant durable, user-owned memory backed by ordinary Git files.

The public implementation is `runethread/core`. Actual personal memory belongs in a separate private repository, conventionally named `runethread-memory`.

For most users, the preferred setup is the AI-first flow in `README.md` and `AI_SETUP.md`. The CLI remains the deterministic local initialization, validation, search, index, and migration tool.

## Requirements

For a prebuilt release binary:

- Git is recommended;
- a private Git remote such as GitHub/GitLab is recommended for durable remote storage;
- the AI client needs authorized access to the private memory repository.

Go is not required for a prebuilt binary. Go is required only when installing/building from source. GitHub CLI (`gh`) is optional.

## 1. Install Runethread

Download an official release binary and verify it against `SHA256SUMS.txt` when practical.

On Windows, running `runethread-windows-amd64.exe` without arguments launches the first-run wizard. It asks for a target directory (default `runethread-memory`), creates a native repository, initializes Git when available, and prints the remaining private-remote/AI-access steps.

Install with Go from an immutable release tag:

```bash
go install github.com/runethread/core/cmd/runethread@<release-tag>
```

For development only:

```bash
git clone https://github.com/runethread/core.git
cd core
go build -o runethread ./cmd/runethread
```

## 2. Initialize a private memory repository

```bash
mkdir runethread-memory
cd runethread-memory
git init
runethread init .
```

`runethread init` refuses to overwrite a non-empty directory; a directory containing only `.git` is allowed.

A native repository contains:

- `MEMORY_PROTOCOL.md` — mandatory operating rules;
- `docs/USER_COMMANDS.md` — the `store/search` contract;
- `docs/EXTENDING_RUNETHREAD.md` — taxonomy/schema-extension rules;
- `docs/INDEX_FORMAT.md` — deterministic Index v2 layout;
- `schema/` and `templates/` — memory structure;
- `memories/` — canonical atomic memories;
- `projects/` — concise project state views;
- `index/` — generated retrieval acceleration;
- `.runethread/config.json` — compatibility/version metadata;
- `.runethread/lock.json` — immutable release pin and control-plane digests;
- `.github/workflows/validate.yml` — read-only trust/bootstrap validation workflow.

## 3. Create a private remote

Using GitHub CLI:

```bash
git add .
git commit -m "Initialize Runethread memory repository"
gh repo create runethread-memory --private --source=. --remote=origin --push
```

Or create an empty **private** repository in the Git host UI and push normally.

Keep the repository private unless you intentionally choose to publish its contents.

## 4. Give the AI authorized repository access

Use the AI client's official Git-provider connection/authorization UI so it can read and, when desired, write the private repository.

Never paste GitHub passwords, PATs, OAuth tokens, session cookies, SSH private keys, or equivalent credentials into chat.

Runethread itself does not require a hosted server and does not receive the user's memory data.

## 5. Use Runethread from conversations

The two primary user commands are:

```text
Runethread: store ...
Runethread: search ...
```

Examples:

```text
Runethread: store that I prefer verified outputs before claims of coding success.
Runethread: search for my standing coding preferences.
```

`store` means durable write. `search` is retrieval-only.

Deterministic local search is also available:

```bash
runethread search --root . "standing coding preferences"
```

A full UUID uses the exact-ID shard route; ordinary text uses Index v2's inverted term index.

## 6. Organize memories without changing the schema

Most categories belong in flexible retrieval metadata: projects, topics, tags, aliases, and entities.

```text
Runethread: store this under the topic home-automation.
```

Adding a core memory `type` is different because it changes semantic behavior and may require schema/validator/migration work. Normal memory operators must not invent a ninth core type or edit the pinned schema as a routine write. See `docs/EXTENDING_RUNETHREAD.md`.

## 7. Upgrade or migrate a repository

Native repositories remain pinned until an explicit upgrade:

```bash
runethread upgrade .
```

Runethread v0.6.0 additionally recognizes exactly one trusted predecessor state: GitMemo v0.5.0 / format 1 / schema 1 / contract 6 / lock 1. That cutover is verified before any native files are written.

The v0.6 upgrader:

1. detects native versus supported legacy metadata and refuses mixed state;
2. verifies the exact supported source state;
3. snapshots managed/generated paths;
4. writes native `.runethread` metadata and pinned contract;
5. preserves canonical `memories/`, `projects/`, and unrelated user files;
6. rebuilds Index v2;
7. validates the resulting repository; and
8. restores the snapshot on a hard post-write failure.

A custom validation workflow is not overwritten silently.

## 8. Index freshness and validation

```bash
runethread index --check .
runethread validate .
```

After a write affecting indexed metadata:

```bash
runethread index --write .
runethread index --check .
runethread validate .
```

If a write-capable client cannot run the indexer, preserve or create the standard stale marker:

```bash
runethread index --mark-stale .
```

A stale index is degraded discovery, not loss of canonical memory. Use repository search/canonical sidecars until deterministic regeneration succeeds.

## Privacy and security

Runethread is not a secrets vault. Do not store passwords, API tokens, private keys, recovery codes, session credentials, banking credentials, or other authentication secrets.

The user's memory repository is user-owned plain Git data. The public project contains tooling and release contracts, not personal memories.
