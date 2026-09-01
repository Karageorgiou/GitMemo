# Memory Sidecar Schema — V1

## Purpose

Each atomic memory has a human-readable Markdown file and a strict JSON sidecar.

The JSON sidecar exists for deterministic retrieval, indexing, provenance tracking, lifecycle management, relationships, validation, deduplication, and future tooling.

The authoritative machine schema is `schema/memory-item.schema.json`.

This document explains field semantics. If this document and the JSON Schema conflict structurally, the JSON Schema controls what is valid.

## Core fields

- `schema_version`: integer schema version; V1 is `1`.
- `id`: immutable lowercase UUIDv4; never encode mutable meaning in it.
- `title`: concise canonical human-readable title.
- `type`: one of `fact`, `preference`, `decision`, `state`, `open_loop`, `correction`, `milestone`, `reference`.
- `lifecycle`: one of `active`, `superseded`, `withdrawn`.
- `summary`: compact retrieval summary; do not copy the whole Markdown body.
- `projects`: canonical project slugs; may be empty or contain multiple projects.
- `topics`: durable conceptual categories.
- `tags`: lightweight retrieval/filter terms.
- `aliases`: natural-language search phrases and synonyms.
- `entities`: structured `{kind, name}` named entities.
- `importance`: `normal`, `high`, or `critical`.
- `temporal`: memory timestamps plus optional real-world effective boundaries.
- `provenance`: epistemic basis, confidence, explicit-memory flag, and sources.
- `relationships`: canonical outgoing graph edges.
- `content_path`: repository-relative path to the paired Markdown file.
- `sensitivity`: `routine`, `private`, or `sensitive`.
- `open_loop_status`: conditional field only for `open_loop` memories.

## Provenance

`basis` is one of `user_stated`, `project_verified`, `external_verified`, `derived`, `inferred`, `migrated`.

`confidence` is separately `high`, `medium`, or `low`.

At least one source is mandatory. Each source has `kind`, non-empty `locator`, optional `revision`, and optional `note`.

## Relationships

Allowed V1 types are `related_to`, `depends_on`, `supersedes`, `corrects`, and `conflicts_with`.

Reverse relationships are derived. Do not store inverse duplicates.

## Required empty arrays

Arrays such as `projects`, `topics`, `tags`, `aliases`, `entities`, and `relationships` remain required even when empty. This deliberately removes ambiguity between “none” and “field forgotten.”

## Schema versus repository validation

The JSON Schema validates one sidecar in isolation. It cannot determine repository-wide facts such as whether UUIDs are globally unique, relationship targets exist, Markdown pairs exist, supersession is acyclic, or timestamps satisfy semantic ordering rules.

Those invariants are defined in `docs/REPOSITORY_VALIDATION.md`.
