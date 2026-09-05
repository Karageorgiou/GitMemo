# Current Runethread engineering milestone

Last reviewed: 2026-09-05

This file is the concise current-work pointer for contributors and agents. The long-term sequence remains in `ROADMAP.md`; when an old roadmap paragraph conflicts with this file about the **immediate** next work, verify live repository state and follow this file plus the linked issue/accepted ADRs.

## Completed baseline

- Phase 0 architecture is merged.
- Phase 1 Runethread identity cutover is complete.
- Phase 2 deterministic MemoryService is merged and released.
- Phase 2.5 compatibility hardening is complete and issue #14 is closed.
- Runethread v0.8.0 is the current immutable release and introduces contract version 8 plus explicit runtime-release / contract-release separation.
- `runethread/memory-template` is migrated to contract v8 / v0.8.0 and passes permanent bootstrap validation.
- The known private `runethread-memory` repository is migrated to contract v8 / v0.8.0 with canonical memory, project, schema, template, and Index v2 bytes preserved where the migration contract required preservation.
- ADR-012 and ADR-013 are accepted and define audited candidate promotion plus the per-repository mutation-delivery queue for Phase 2.6.

## Immediate milestone — Phase 2.6 Memory Write Delivery Pipeline

Phase 2.6 is the current engineering milestone. **Phase 3 local MCP implementation is blocked until Phase 2.6 satisfies its exit criteria in issue #20.**

The goal is to give GitHub/cloud-only semantic callers a permanent deterministic delivery path that invokes the existing MemoryService, keeps unaudited candidates off canonical `main`, and performs each distinct correctness gate once. This milestone does not create a second memory implementation and does not turn the Core development pipeline into a reduced-safety fast mode.

The governing architectural decisions are:

- ADR-012 — audited candidate promotion for external memory delivery;
- ADR-013 — per-repository serialized mutation-delivery queue.

Phase 2.6 implementation priorities are:

1. define the sealed GitHub workflow-dispatch request shape for one complete MemoryService-compatible mutation;
2. run finalization through the existing deterministic MemoryService/Core boundary from the exact expected Git revision;
3. keep the resulting candidate noncanonical while rebuilding Index v2 and applying hard validation;
4. independently audit the exact candidate on a fresh read-only runner/job with strict index freshness and applicable trust/control-plane verification;
5. publish only the exact audited candidate through an expected-revision, non-force fast-forward compare-and-swap;
6. use a dedicated least-privilege Runethread GitHub App as the canonical publisher for user-authorized memory repositories, with managed repository policy preventing routine writers from bypassing the audited path;
7. expose deterministic operation states including queued/finalizing/audit/publishing/committed outcomes, park stale work as `NEEDS_REPREPARE`, and preserve idempotent crash/lost-response recovery;
8. implement write-lane suspension for audit disagreement and explicit barriers for contract/schema/trust/repository-format/bootstrap/workflow/migration changes;
9. roll the finished mechanism through `runethread/memory-template` and then a real private memory repository under the normal downstream gates;
10. measure end-to-end latency and verify that the normal memory-delivery path removes the previous duplicated synchronous validation ceremony without weakening correctness.

Phase 2.6 v1 deliberately uses singleton operations. The following remain deferred:

- semantic memory/read dependency quantification;
- neighboring-operation batching/coalescing;
- automatic semantic re-preparation;
- Phase 3 MCP transport implementation;
- Orchestrator runtime/database work;
- hosted multi-tenant/cloud infrastructure beyond what is strictly required to prove this GitHub-backed delivery profile.

GitHub Actions is the first replaceable execution adapter, not the architectural queue authority. Correctness must depend on MemoryService invariants, exact Git revisions, immutable candidate identity, independent audit, and fail-closed publication rather than workflow execution order. A later hosted executor such as Cloudflare may replace the execution backend without changing canonical memory format solely for queue state.

## Next phase after Phase 2.6

After issue #20 exits green and the Phase 2.6 implementation/rollout is independently verified, Phase 3 may begin as a thin local MCP adapter over the same MemoryService application boundary and the delivery lifecycle established here. MCP must remain transport rather than a second owner of storage, lifecycle, provenance, trust, indexing, idempotency, concurrency, or Git transaction behavior.

## Engineering procedure

All substantive changes follow `docs/runethread/ENGINEERING_PROCESS.md` and `docs/runethread/DEVELOPMENT_PIPELINE.md`. Validation CI is read-only and must never patch/push source.

## Historical / superseded work

- PR #13 (`superseded: decouple runtime and contract releases`) remains closed as design-forensics history and must not be revived or used as an implementation base.
- Phase 2.5 design and completion evidence remain recorded in issue #14 and ADR-011.
- The 2026-09-04 milestone pointer that made Phase 3 immediately current is superseded by accepted ADR-012/ADR-013 and the Phase 2.6 sequencing decision recorded here and in issue #20.
