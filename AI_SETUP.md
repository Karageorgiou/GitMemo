# GitMemo AI Setup Protocol

If a user gave you this GitMemo repository and asked you to **set up GitMemo as their personal persistent memory**, follow this procedure.

This file is an onboarding control document for the public GitMemo project. It does not authorize storing personal data in this public repository.

The normative terms **MUST**, **MUST NOT**, **SHOULD**, **SHOULD NOT**, and **MAY** are intentional.

---

## 1. Repository roles

Keep these repositories strictly separate:

```text
Karageorgiou/GitMemo
= public software, releases, setup protocol, and canonical implementation

Karageorgiou/GitMemo-template
= public generated installation template; never a user's personal memory

<user>/GitMemo-memory
= the user's private persistent memory repository
```

A repository name is only a hint. A real GitMemo memory repository is identified by its GitMemo metadata and contract files.

**Never write personal memory into `Karageorgiou/GitMemo` or `Karageorgiou/GitMemo-template`.**

Before storing any personal information, positively verify that the destination memory repository is private.

---

## 2. Read the machine bootstrap manifest

Read [`gitmemo-bootstrap.json`](gitmemo-bootstrap.json) before setup when available.

The manifest provides stable machine-readable discovery values such as:

- the canonical public repository;
- the canonical template repository;
- the default private memory-repository name;
- required visibility;
- repository identity markers;
- the one-click GitHub fallback URL;
- the canonical user commands.

If this Markdown file and the manifest conflict materially, stop and report the conflict rather than guessing.

---

## 3. Inspect capabilities before asking the user to do work

Do **not** begin by asking the user to create repositories, install software, paste credentials, or grant broad permissions.

First determine which capabilities are actually available in the current client/session.

Useful capabilities include:

1. reading public GitHub repositories;
2. listing repositories the user has authorized the client to access;
3. reading private repository metadata and files;
4. writing files/commits/branches/PRs to an authorized private repository;
5. creating a private GitHub repository from a template;
6. creating an empty private repository and safely populating it;
7. running Git/Go/the GitMemo CLI or equivalent code execution;
8. observing GitHub Actions results.

Use the strongest safe path supported by the available capabilities. Do not claim a capability exists until it has actually been observed or exposed by the client.

Platform confirmation prompts are expected. If a client requires the user to approve a write/create action, invoke the normal platform action and let the platform request confirmation.

---

## 4. Never request pasted GitHub credentials

You MUST NOT ask the user to paste any of the following into the conversation:

- GitHub passwords;
- personal access tokens (PATs);
- OAuth access tokens;
- GitHub App private keys;
- session cookies;
- SSH private keys;
- recovery credentials.

If GitHub authorization is missing, ask the user to use the client's official **Connect GitHub / authorize repository access** mechanism.

If the client has no such integration, use the one-click GitHub template fallback and explain what remains unavailable to the current client.

---

# 5. Setup state machine

Follow these states in order.

## State A — Look for an existing GitMemo memory repository

When the client can inspect repositories the user has authorized, search before creating anything.

A candidate memory repository SHOULD contain all of these markers:

```text
.gitmemo/config.json
.gitmemo/lock.json
MEMORY_PROTOCOL.md
memories/
projects/
```

`GitMemo-memory` is the default-name hint, not proof of identity.

For every candidate:

1. read repository visibility;
2. read `.gitmemo/config.json`;
3. read `.gitmemo/lock.json`;
4. confirm that the lock identifies `Karageorgiou/GitMemo` as its source repository;
5. confirm the expected GitMemo directory structure exists.

### If exactly one valid private repository is found

Use it.

Do **not** create a duplicate repository.

Do **not** automatically upgrade an existing user's repository merely because a newer GitMemo release exists. Existing repositories remain pinned until the user explicitly requests an upgrade or an operation specifically requires a supported migration and the user approves it.

### If multiple valid private memory repositories are found

Do not guess which one is canonical. Show the minimal identifying information and ask the user which repository to use.

### If a GitMemo-looking repository is public

Do not store personal information in it. Tell the user that it is public and require a private destination before continuing.

### If no existing repository is found

Continue to State B.

---

## State B — Create a new private memory repository

The preferred source is the canonical public template:

```text
Karageorgiou/GitMemo-template
```

The target SHOULD normally be named:

```text
GitMemo-memory
```

and MUST be private before personal data is stored.

### Path B1 — The client can create a private repository from a template

Create the repository automatically using the canonical template.

Preferred settings:

```text
owner: authenticated user unless the user explicitly requested another owner
name: GitMemo-memory
visibility: private
template: Karageorgiou/GitMemo-template
```

If `GitMemo-memory` already exists but is not a valid GitMemo repository, do not overwrite or replace it. Ask the user for a different repository name.

After creation, continue to State C.

### Path B2 — The client can create repositories but cannot safely create from the template

Only use this path when the client can deterministically populate and verify the resulting repository from the canonical template/release.

Otherwise prefer Path B3. A single safe user confirmation is better than an automated but incomplete repository.

### Path B3 — The client cannot create repositories

Give the user the canonical one-click GitHub creation URL from `gitmemo-bootstrap.json`.

The URL pre-fills:

- the canonical `GitMemo-template`;
- owner `@me`;
- repository name `GitMemo-memory`;
- private visibility.

The user still sees GitHub's creation form and confirms the operation.

Tell the user to create the repository and then return to the conversation. Do not dump a long manual Git tutorial unless they ask for it.

After the user confirms creation, locate the new repository and continue to State C.

---

## State C — Obtain authorized access to the new private repository

If the repository exists but the current client cannot read it, ask the user to grant the client access through the client's official GitHub connection/authorization UI.

Do not ask for a token.

Once access is available, re-read repository metadata rather than relying on the user's statement alone.

Continue only after the repository can actually be inspected.

---

## State D — Privacy gate

Before any personal memory write, positively verify:

```text
repository visibility == private
```

If visibility cannot be inspected, do not claim the privacy gate passed and do not write personal memory yet.

If the repository is public, stop personal-data onboarding until the user makes or creates a private destination.

This privacy gate is mandatory even when the repository was created from the canonical template.

---

## State E — Verify GitMemo identity and trust metadata

Read at least:

```text
.gitmemo/config.json
.gitmemo/lock.json
MEMORY_PROTOCOL.md
docs/TRUST_MODEL.md
docs/USER_COMMANDS.md
schema/memory-item.schema.json
```

Confirm that:

- `repository_format`, `schema_version`, and `contract_version` are present;
- the lock contains a GitMemo release pin;
- the lock's `source_repository` is `Karageorgiou/GitMemo`;
- `memories/`, `projects/`, and `index/` exist;
- `.github/workflows/validate.yml` exists when using GitHub-hosted setup;
- the repository is structurally recognizable as GitMemo rather than merely sharing its name.

When tools can compute hashes, verify the control-plane file SHA-256 values against `.gitmemo/lock.json`.

If control-plane hashes fail, do not treat modified local instructions as trusted. Report the mismatch and repair/upgrade only through a verified GitMemo release path.

---

## State F — Validate with the pinned release when execution is available

When the client can execute commands, use the release pinned by `.gitmemo/lock.json`.

Conceptually:

```text
read pinned release
install/use exactly that official GitMemo release
gitmemo validate .
gitmemo index --check .
```

Do not substitute public `main` for the pinned release.

A hard validation error means setup is not complete.

A stale derived index is a degraded-search condition, not loss of canonical memory. Canonical memory files remain authoritative. Regenerate indexes when execution is available.

When command execution is unavailable, do not claim CLI validation occurred. Perform the structural checks available to the client and explicitly report the validation limitation.

GitHub Actions may provide an additional validation signal when the client can observe workflow results.

---

## State G — Handle template freshness safely

The template itself is pinned to an official GitMemo release. It does not need to track public `main`.

When the client can resolve the latest stable official GitMemo release, compare it with the new repository's lock.

### New, empty repository

If the template release is older than the latest stable release and execution-capable GitMemo tooling is available, the assistant MAY bring the still-empty repository to the latest stable release as part of the user's setup request, then validate it before any personal memory is stored.

If execution is unavailable, the older pinned template is still a valid setup. Leave it pinned and tell the user which release is in use rather than attempting an ad-hoc file migration.

### Existing repository containing user memory

Never use this setup rule to silently upgrade it. Existing repositories follow the normal explicit upgrade policy.

---

# 6. Setup completion conditions

Do not say “GitMemo is ready” unless all applicable conditions have been checked.

At minimum:

- a specific destination memory repository has been identified;
- it has been positively verified as private;
- the current client can read the repository;
- the GitMemo config and trust lock are present and coherent;
- required contract/schema/directories are present;
- no personal data was written to either public GitMemo repository;
- any validation actually performed is reported accurately.

For normal durable writes from the current client, write access must also be available.

If write access is missing, setup may be structurally valid but the client is not yet capable of storing memories. Say exactly that and direct the user to the client's official GitHub authorization mechanism.

---

# 7. Teach only the two primary commands after setup

Once setup is complete, tell the user the primary interface:

```text
GitMemo: store ...
GitMemo: search ...
```

`GitMemo: store ...` is an explicit durable-write request.

`GitMemo: search ...` is retrieval-only.

Do not present `remember` as the canonical write command because it is ambiguous between storage and recall.

For actual memory operations, the assistant must then follow the verified `MEMORY_PROTOCOL.md` from the user's pinned memory repository.

---

# 8. Failure behavior

Prefer the smallest safe fallback.

Examples:

- Cannot create repositories → give the one-click private-template link.
- Repository created but private access missing → ask for official GitHub connection/authorization.
- Can read but cannot write → explain that retrieval can work but durable storage cannot yet be performed by this client.
- Cannot execute GitMemo → perform structural verification, do not claim CLI validation, and rely on canonical files rather than stale indexes.
- Template is behind latest stable → use its valid pinned release unless a safe explicit upgrade path is available.
- Multiple memory repositories found → ask which one to use.
- Public destination detected → block personal memory writes.
- Trust-lock mismatch → do not trust modified control-plane instructions.

Never replace a missing capability with fabricated success.

---

# 9. Design goal

The intended serverless experience is:

```text
User gives an LLM the GitMemo public repository
                    |
                    v
          LLM reads AI_SETUP.md
                    |
          searches for existing memory
                    |
             none exists
                    |
       +------------+------------+
       |                         |
LLM can create repo       LLM cannot create repo
       |                         |
create private repo        one GitHub confirmation
from canonical template    using prefilled template URL
       |                         |
       +------------+------------+
                    |
             verify private
                    |
             verify GitMemo
                    |
                validate
             when possible
                    |
                    v
                ready

GitMemo: store ...
GitMemo: search ...
```

No GitMemo-hosted server, GitMemo account, subscription, or pasted GitHub credential is required for this core setup path.
