# Runethread user command interface

This document defines the small user-facing command convention for asking an AI assistant to use a Runethread memory repository.

These commands are natural-language control phrases, not parser-sensitive shell commands. Assistants SHOULD tolerate ordinary capitalization and punctuation variations while preserving the intent described here.

## Repository discovery

When a Runethread command is issued and the active memory repository is not already known, the assistant SHOULD locate the user's private Runethread repository before acting.

A native repository is identified by at least:

- `.runethread/config.json`
- `.runethread/lock.json`
- `MEMORY_PROTOCOL.md`
- `memories/`
- `index/`

A repository named `runethread-memory` is only a strong candidate when it satisfies those markers. The public `runethread/core` implementation and `runethread/memory-template` MUST NOT be mistaken for the user's private memory repository.

If multiple plausible repositories exist and the target cannot be established safely, ask the user which repository to use. If repository access is unavailable, say so rather than pretending the command completed.

## `Runethread: store <content>`

This is an explicit durable memory-write request.

The assistant MUST:

1. locate and read the active private repository's verified `MEMORY_PROTOCOL.md`;
2. follow schema, content-format, taxonomy, security, stale-write, and validation rules;
3. search existing memories before creating anything new;
4. decide whether to create, update, supersede, correct, resolve, or leave an existing memory unchanged;
5. set `provenance.explicit_memory_request` to `true` on a newly created or materially updated atomic memory when applicable;
6. regenerate affected indexes when tooling is available, or mark `index/STALE` when the client can write files but cannot execute the indexer;
7. run available repository validation before claiming success;
8. commit the completed change when repository write access is available; and
9. report what was actually stored or changed, including relevant memory IDs and the commit/result actually verified.

The command does not override security rules. Forbidden secrets or credentials MUST NOT be stored merely because the user used `Runethread: store`.

Example:

```text
Runethread: store that for coding tasks I want actual outputs verified before claiming success.
```

## `Runethread: search <query>`

This is an explicit retrieval-only request.

The assistant MUST NOT create, modify, regenerate, commit, resolve, supersede, or withdraw memories merely because this command was used.

The assistant SHOULD:

1. locate and read the active private repository's verified operating contract;
2. determine whether the generated index is known current;
3. use the narrowest Index v2 entry point when usable: exact UUID shard, direct metadata index, or deterministic term index / `runethread search`;
4. fall back to repository search and canonical sidecars when index freshness is stale, unknown, missing, or unsupported;
5. retrieve the smallest canonical evidence set needed;
6. follow relevant correction, supersession, dependency, or conflict edges; and
7. distinguish stored memory from live project or external verification performed in addition to memory retrieval.

An index hit is discovery metadata, not sufficient factual evidence by itself. Read the selected canonical Markdown/JSON pair before relying on substantive content.

If nothing relevant is stored, say that the Runethread search found no relevant memory rather than inventing one. If index freshness is unknown, do not claim a complete negative search based only on the index.

Examples:

```text
Runethread: search what we decided about Go versus Python.
```

```text
Runethread: search for unresolved memory-system work.
```

## Why `store`, not `remember`

`remember` is intentionally not the canonical write command because ordinary conversation uses it for both storage and recall. `store` is unambiguous: persist this in Runethread.

An assistant MUST NOT interpret `Runethread: remember ...` as an explicit durable write merely from the word `remember`. If intent is clear from additional wording, follow that intent; otherwise prefer the canonical `store` and `search` commands.

## Ordinary use

A user may also say something like `use Runethread` without an explicit command. That means the assistant may consult Runethread when materially useful, but it is not by itself an explicit request to create durable memory.

The two primary commands are intentionally small and stable:

```text
Runethread: store ...
Runethread: search ...
```
