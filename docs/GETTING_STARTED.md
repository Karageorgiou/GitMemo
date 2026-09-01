# Getting started with GitMemo

GitMemo gives an AI assistant a durable, user-owned memory repository backed by plain Git files.

The public `GitMemo` repository contains the tooling and operational contract. Your actual memories belong in a separate private repository, conventionally named `GitMemo-memory`.

## Requirements

- Git
- Go 1.23 or newer
- a Git host such as GitHub
- an AI assistant that can access your private memory repository when you ask it to use GitMemo

GitHub CLI (`gh`) is optional but makes private-repository creation easier.

## 1. Install GitMemo

After the first public release, install the CLI with:

```bash
go install github.com/Karageorgiou/GitMemo/cmd/gitmemo@v0.1.0
```

Make sure your Go binary directory is on `PATH`.

You can also clone this repository and build locally:

```bash
git clone https://github.com/Karageorgiou/GitMemo.git
cd GitMemo
go build -o gitmemo ./cmd/gitmemo
```

## 2. Initialize your private memory repository

Create a fresh directory and initialize Git first:

```bash
mkdir GitMemo-memory
cd GitMemo-memory
git init
gitmemo init .
```

`gitmemo init` refuses to overwrite a non-empty directory. A directory containing only `.git` is allowed.

The generated repository contains:

- `MEMORY_PROTOCOL.md` — mandatory operating rules for AI assistants
- `docs/USER_COMMANDS.md` — the `store/search` command contract
- `schema/` and `templates/` — memory structure
- `memories/` — atomic durable memories
- `projects/` — concise project state views
- `index/` — generated retrieval indexes
- `.gitmemo/config.json` — repository/contract version metadata
- `.github/workflows/validate.yml` — validation CI pinned to GitMemo v0.1.0

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

Use two explicit commands:

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

A fresh assistant should locate the private memory repository, read `MEMORY_PROTOCOL.md` and `docs/USER_COMMANDS.md`, then follow the repository's rules rather than improvising its own memory format.

## 6. Validate locally

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

The generated GitHub Actions workflow runs index freshness and repository validation on pushes and pull requests.

## Privacy and security

GitMemo is not a secrets vault. Do not store passwords, API tokens, private keys, recovery codes, session credentials, banking credentials, or other authentication secrets.

The memory repository is user-owned plain Git data. The public GitMemo project contains the tooling, not your personal memories.
