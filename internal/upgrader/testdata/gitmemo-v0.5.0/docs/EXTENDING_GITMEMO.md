# Extending GitMemo safely

GitMemo deliberately separates **retrieval taxonomy** from **core memory semantics**.

This lets users organize memories flexibly without allowing each repository to drift into an incompatible private schema.

## Flexible categories: use projects, topics, tags, aliases, and entities

Most requests to add a new "category" do **not** require a GitMemo schema change.

Examples:

- `health`
- `university`
- `pokemon`
- `finance`
- `home-automation`
- a new project name
- a new technology or organization entity

These belong in the existing retrieval metadata:

- `projects` for durable bodies of work;
- `topics` for broader stable conceptual categories;
- `tags` for narrower filters;
- `aliases` for natural-language search synonyms;
- `entities` for structured named things.

An AI assistant MAY introduce a new project/topic/tag/entity when the existing taxonomy has no semantically equivalent term, subject to `docs/TAXONOMY.md`. It SHOULD search existing terms first to avoid near-duplicate vocabulary.

This kind of extension does not change repository format, schema version, contract version, validator behavior, or compatibility with other GitMemo installations.

## Core memory types are different

The canonical V1 memory types are:

- `fact`
- `preference`
- `decision`
- `state`
- `open_loop`
- `correction`
- `milestone`
- `reference`

A new core memory type changes semantic behavior. It may require:

- a JSON Schema change;
- Go validator changes;
- a new Markdown authoring template;
- indexer behavior changes;
- lifecycle/status rules;
- protocol documentation;
- migration logic for existing repositories;
- a schema and/or contract version bump.

Therefore an AI assistant operating a normal user memory repository MUST NOT invent a new value for the core `type` field or edit the local schema merely to satisfy a one-off organization request.

If the user requests a genuinely new semantic memory type, the assistant SHOULD first determine whether an existing type plus project/topic/tag metadata already models the need. If not, treat the request as a GitMemo system-design change rather than an ordinary memory write.

## Talking to an LLM about categories

Users may use normal language. No special CLI command is required for ordinary taxonomy extension.

Examples:

```text
GitMemo: store this under the topic home-automation.
```

```text
GitMemo: store this as part of project pepper-museum.
```

The assistant should reuse existing taxonomy terms when possible and add a new term only when it represents a genuinely new stable concept.

A request such as:

```text
Add a new GitMemo memory type called hypothesis with its own lifecycle rules.
```

is not an ordinary `store` operation. It is a proposal to evolve GitMemo itself and should go through the public implementation's design, tests, validation, versioning, release, and repository-upgrade process.

## Why core types are intentionally closed

Allowing arbitrary per-repository memory types would weaken several properties GitMemo is designed to preserve:

- deterministic validation;
- predictable retrieval behavior across assistants;
- portable memory repositories;
- deterministic indexes;
- meaningful upgrades;
- a stable public schema;
- the ability for an unfamiliar LLM to understand a repository from the vendored contract alone.

GitMemo may gain an explicit extension mechanism in the future if real usage demonstrates a need, but V1 does not treat arbitrary custom core types as a supported repository-local feature.
