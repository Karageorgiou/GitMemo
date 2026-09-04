# Current Runethread engineering milestone

Last reviewed: 2026-09-04

This file is the concise current-work pointer for contributors and agents. The long-term sequence remains in `ROADMAP.md`; when an old roadmap paragraph conflicts with this file about the **immediate** next work, verify live repository state and follow this file plus the linked issue/accepted ADRs.

## Completed baseline

- Phase 0 architecture is merged.
- Phase 1 Runethread identity cutover is complete.
- Phase 2 deterministic MemoryService is merged and released.
- Phase 2.5 compatibility hardening is complete and issue #14 is closed.
- Runethread v0.8.0 is the current immutable release and introduces contract version 8 plus explicit runtime-release / contract-release separation.
- `runethread/memory-template` is migrated to contract v8 / v0.8.0 and passes permanent bootstrap validation.
- The known private `runethread-memory` repository is migrated to contract v8 / v0.8.0 with canonical memory, project, schema, template, and Index v2 bytes preserved where the migration contract required preservation.

## Immediate milestone — Phase 3 local MCP adapter

Phase 3 may now begin from a freshly verified `main`.

The goal is a thin local MCP transport over the already-implemented deterministic MemoryService, not a second implementation of memory behavior.

Before the first Phase 3 implementation write:

1. verify current `main`, current release, and permanent CI state;
2. re-check current authoritative MCP and Go SDK/toolchain requirements rather than relying on older research;
3. define the smallest MCP tool surface that maps cleanly to existing MemoryService operations;
4. keep storage, lifecycle, provenance, trust, indexing, idempotency, concurrency, and Git transaction rules inside Core rather than duplicating them in the adapter;
5. preserve CLI/native behavior and provider independence;
6. classify any proposed protocol/API/dependency change through the normal engineering-process gates before implementation.

Expected MemoryService-facing operations remain conceptually:

```text
Search
Get
PrepareMutation
ApplyMutation
Withdraw
Status
```

The first MCP milestone does not include the separate Runethread Orchestrator, hosted cloud infrastructure, worker routing, or unrelated repository-format/schema/contract changes unless new evidence independently requires them.

## Engineering procedure

All substantive changes follow `docs/runethread/ENGINEERING_PROCESS.md` and the repository PR checklist. Validation CI is read-only and must never patch/push source.

## Historical / superseded work

- PR #13 (`superseded: decouple runtime and contract releases`) remains closed as design-forensics history and must not be revived or used as an implementation base.
- Phase 2.5 design and completion evidence remain recorded in issue #14 and ADR-011.
