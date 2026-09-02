# Getting started with GitMemo

GitMemo gives an AI assistant a durable, user-owned memory repository backed by plain Git files.

The public `GitMemo` repository contains the tooling and operational contract. Your actual memories belong in a separate private repository, conventionally named `GitMemo-memory`.

For most users, the preferred setup is now the AI-first flow described by the public `README.md` and `AI_SETUP.md`. The local CLI remains the deterministic initialization, validation, search, and upgrade tool.

## Requirements

For the downloadable release binary:

- Git is recommended and is initialized automatically by the first-run wizard when available;
- a Git host such as GitHub, GitLab, or another private remote is recommended for durable remote storage;
- an AI assistant must be able to access your private memory repository when you ask it to use GitMemo.

Go is **not required** when you use a prebuilt release binary. Go 1.23 or newer is only required when installing/building GitMemo from source.

GitHub CLI (`gh`) is optional but makes private-repository creation easier.

## 1. Install GitMemo

### Download a release binary

Download the binary for your platform from the latest stable GitHub release and verify it against `SHA256SUMS.txt` when practical.

On Windows, double-clicking `gitmemo-windows-amd64.exe` with no command-line arguments launches the first-run wizard. The wizard:

1. asks for a target directory, defaulting to `GitMemo-memory`;
2. creates the self-describing GitMemo memory-repository skeleton;
3. initializes a local Git repository on branch `main` when Git is installed; and
4. prints the remaining steps for creating a private remote and connecting an AI assistant.

The wizard does **not** create an online GitHub/GitLab account or private remote for you because GitMemo does not require any specific hosting provider or credentials.

### Install with Go

```bash
go install github.com/Karageorgiou/GitMemo/cmd/gitmemo@<release-tag>
```

Use an official immutable release tag such as `v0.5.0`, not mutable `main`, for a user memory repository.

Make sure your Go binary directory is on `PATH`.

You can also clone this repository and build locally for development:

```bash
git clone https://github.com/Karageorgiou/GitMemo.git
cd GitMemo
go build -o gitmemo ./cmd/gitmemo
```

## 2. Initialize your private memory repository

If you are using a terminal instead of the first-run wizard:

```bash
mkdir GitMemo-memory
cd GitMemo-memory
git init
gitmemo init .
```

`gitmemo init` refuses to overwrite a non-empty directory. A directory containing only `.git` is allowed.

The generated repository contains:

- `MEMORY_PROTOCOL.md` — mandatory operating rules for AI assistants;
- `docs/USER_COMMANDS.md` — the `store/search` command contract;
- `docs/EXTENDING_GITMEMO.md` — category/schema-extension rules;
- `docs/INDEX_FORMAT.md` — deterministic generated-index layout and freshness/fallback rules;
- `schema/` and `templates/` — memory structure;
- `memories/` — atomic durable memories;
- `projects/` — concise project state views;
- `index/` — generated retrieval acceleration;
- `.gitmemo/config.json` — repository/schema/contract/tool version metadata;
- `.gitmemo/lock.json` — immutable release pin and control-plane digests;
- `.github/workflows/validate.yml` — stable read-only trust bootstrap that resolves and validates with the release pinned in the lock.

## 3. Create the private remote repository

### With GitHub CLI

From inside `GitMemo-memory`:

```bash
git add .
git commit -m "Initialize GitMemo memory repository"
gh repo create GitMemo-memory --private --source=. --remote=origin --push
```

### With the GitHub website

Create a new **private** empty repository, for example `GitMemo-memory`, without a generated README, license, or `.gitignore`.

Then from your local memory directory:

```bash
git add .
git commit -m "Initialize GitMemo memory repository"
git branch -M main
git remote add origin https://github.com/YOUR-USERNAME/GitMemo-memory.git
git push -u origin main
```

Keep this repository private unless you intentionally want to publish your memories.

## 4. Give your AI assistant repository access

Connect or authorize the Git provider you use so the assistant can read your private `GitMemo-memory` repository. The exact connection mechanism depends on the assistant and platform.

GitMemo itself does not require a server and does not receive your memory data.

## 5. Use GitMemo from any conversation

Use two explicit natural-language commands:

```text
GitMemo: store ...
GitMemo: search ...
```

Examples:

```text
GitMemo: store that I prefer verified outputs before claims of coding success.
```

```text
GitMemo: search for my standing coding preferences.
```

`store` means durable write. `search` means retrieval-only.

A fresh assistant should locate the private memory repository, verify/read the pinned operating contract, then follow the repository's rules rather than improvising its own memory format.

When the local CLI is available, deterministic retrieval is also exposed as:

```bash
gitmemo search --root . "standing coding preferences"
```

A full UUID passed as the query uses the exact-ID shard path. Ordinary text uses Index v2's inverted term index.

## 6. Organize memories without changing the schema

Most new categories belong in GitMemo's flexible retrieval taxonomy. Projects, topics, tags, aliases, and entities may be added when a genuinely new stable concept is needed.

For example:

```text
GitMemo: store this under the topic home-automation.
```

Adding a new core memory `type` is different because it can change schema and validation semantics. Normal assistants must not edit the local schema or invent a ninth core type as an ordinary memory write. See `docs/EXTENDING_GITMEMO.md`.

## 7. Upgrade an existing memory repository

Existing repositories remain pinned to their current GitMemo release. Publishing a newer GitMemo release does not silently modify them.

Install/download the newer GitMemo executable, then run it against the existing repository:

```bash
gitmemo upgrade .
```

The upgrader:

1. reads `.gitmemo/config.json`;
2. refuses repository/schema/contract versions newer than the running binary understands;
3. updates GitMemo-managed operational contract/configuration files;
4. preserves `memories/`, `projects/`, and unrelated user files;
5. regenerates the complete deterministic index tree for the new release, removing obsolete generated index files from older formats;
6. runs full repository validation; and
7. rolls back managed/generated changes if validation fails.

If `.github/workflows/validate.yml` no longer looks GitMemo-managed, the upgrader refuses to overwrite it rather than destroying a custom workflow.

Commit and push the successful upgrade like any other repository change.

## 8. Index freshness and local validation

From the memory repository:

```bash
gitmemo index --check .
gitmemo validate .
```

After a write that changes indexed memory metadata:

```bash
gitmemo index --write .
gitmemo index --check .
gitmemo validate .
```

If a repository-writing client cannot execute the indexer, the generated index should be treated as potentially incomplete. When possible, explicitly mark that state:

```bash
gitmemo index --mark-stale .
```

or create/preserve the standard `index/STALE` marker according to the pinned contract. Do not hand-edit generated shards to imitate a fresh index.

A stale index does not invalidate otherwise valid canonical memories. Repository search and canonical sidecars remain the fallback discovery path until regeneration succeeds.

See `docs/INDEX_FORMAT.md` for the exact Index v2 layout.

## Privacy and security

GitMemo is not a secrets vault. Do not store passwords, API tokens, private keys, recovery codes, session credentials, banking credentials, or other authentication secrets.

The memory repository is user-owned plain Git data. The public GitMemo project contains the tooling, not your personal memories.
