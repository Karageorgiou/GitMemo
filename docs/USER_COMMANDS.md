# GitMemo user command interface

This document defines the small user-facing command convention for asking an AI assistant to use a GitMemo memory repository.

These commands are natural-language control phrases, not parser-sensitive shell commands. Assistants SHOULD tolerate ordinary capitalization and punctuation variations while preserving the intent described here.

## Repository discovery

When a GitMemo command is issued and the active memory repository is not already known, the assistant SHOULD locate the user's private GitMemo memory repository before acting.

A valid memory repository is identified by the presence of at least:

- `.gitmemo/config.json`
- `MEMORY_PROTOCOL.md`
- `memories/`
- `index/`

On GitHub, a repository named `GitMemo-memory` is a strong candidate when it satisfies those markers.

The public `GitMemo` infrastructure repository MUST NOT be mistaken for the user's private memory repository.

If more than one plausible memory repository exists and the intended target cannot be established safely, ask the user which repository to use rather than writing to an arbitrary candidate.

If repository access is unavailable, say so instead of pretending the command was completed.

## `GitMemo: store <content>`

This is an explicit durable memory-write request.

The assistant MUST:

1. locate and read the active private memory repository's `MEMORY_PROTOCOL.md`;
2. follow the schema, content-format, taxonomy, security, stale-write, and validation rules;
3. search existing memories before creating anything new;
4. decide whether the request should create, update, supersede, correct, resolve, or leave an existing memory unchanged;
5. treat the user's request as an explicit memory request and set `provenance.explicit_memory_request` to `true` on a newly created or materially updated atomic memory when applicable;
6. regenerate affected indexes when tooling is available;
7. run available repository validation before claiming success;
8. commit the completed memory change when repository write access is available; and
9. report what was stored or changed, including the relevant memory ID or IDs and the commit/result actually verified.

The command does not override GitMemo's security rules. Forbidden secrets or credentials MUST NOT be stored merely because the user used `GitMemo: store`.

An explicit store command is presumptively eligible for durable storage, but the assistant should still avoid unnecessary duplication and preserve historical meaning correctly.

Example:

```text
GitMemo: store that for coding tasks I want actual outputs verified before claiming success.
```

## `GitMemo: search <query>`

This is an explicit retrieval-only request.

The assistant MUST NOT create, modify, regenerate, commit, resolve, supersede, or withdraw memories merely because this command was used.

The assistant SHOULD:

1. locate and read the active private memory repository's operating contract;
2. begin with the narrowest useful project/current-state view or generated index;
3. search metadata using the user's terms plus useful aliases, tags, topics, projects, entities, and memory types;
4. retrieve the smallest set of atomic memories needed to answer the query;
5. follow relevant correction, supersession, dependency, or conflict edges when necessary; and
6. clearly distinguish stored memory from any live project or external verification performed in addition to the memory search.

If nothing relevant is stored, say that the GitMemo search found no relevant memory rather than inventing one.

Examples:

```text
GitMemo: search what we decided about Go versus Python.
```

```text
GitMemo: search for unresolved GitMemo work.
```

## Why `store`, not `remember`

`remember` is intentionally not the canonical write command because ordinary conversation uses that word both for storing information and for recalling information. `store` is unambiguous: it means persist this in GitMemo.

An assistant MUST NOT interpret `GitMemo: remember ...` as an explicit durable write merely from the word `remember`. If the user's intent is clear from additional wording, follow that intent; otherwise prefer the canonical `store` and `search` commands.

## Ordinary use

A user may also say something like `use GitMemo` without using one of the explicit commands. That means the assistant may consult GitMemo as context when materially useful, but it is not by itself an explicit request to create a new durable memory.

The two primary commands are intentionally small and stable:

```text
GitMemo: store ...
GitMemo: search ...
```
