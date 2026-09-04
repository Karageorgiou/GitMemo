# Current Runethread engineering milestone

Last reviewed: 2026-09-04

This file is the concise current-work pointer for contributors and agents. The long-term sequence remains in `ROADMAP.md`; when an old roadmap paragraph conflicts with this file about the **immediate** next work, verify live repository state and follow this file plus the linked issue/accepted ADRs.

## Completed baseline

- Phase 0 architecture is merged.
- Phase 1 Runethread identity cutover is complete.
- Phase 2 deterministic MemoryService is merged and released as v0.7.0.
- `runethread/memory-template` and the known private memory repository have been repinned to v0.7.0 with canonical memory bytes preserved and post-migration validation passing.

## Immediate milestone — Phase 2.5 compatibility hardening

Phase 3 MCP **must not start yet**.

The current blocker is release-pin coupling discovered after Phase 2. Published v0.7 uses one release identity for both the running executable and the immutable repository operational contract. A superseded exploratory PR (#13) attempted to separate them without a contract change; review proved that would reinterpret published v0.7 normative semantics.

The replacement work is tracked by issue #14:

> Introduce runtime-release / contract-release separation through an explicit new contract migration, with an exact frozen v0.7 fixture and forward tests proving later runtime-only releases can advance without repository churn.

Required high-level order:

1. merge engineering-process hardening;
2. start Phase 2.5 from freshly verified `main`;
3. audit published v0.7 contract/bootstrap/compatibility semantics;
4. freeze exact v0.7 historical repository fixture;
5. design and implement the explicit new contract transition and migration;
6. release and independently verify it;
7. migrate/verify public template, then private memory repository;
8. only after Phase 2.5 closes, re-evaluate current Go/MCP SDK requirements and begin Phase 3 on a fresh branch.

## Engineering procedure

All substantive changes follow `docs/runethread/ENGINEERING_PROCESS.md` and the repository PR checklist. Validation CI is read-only and must never patch/push source.

## Known superseded work

PR #13 (`superseded: decouple runtime and contract releases`) is intentionally closed and retained only as design-forensics history. Do not merge, revive, or use it as an implementation base.
