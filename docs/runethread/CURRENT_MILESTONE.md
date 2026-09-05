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
- ADR-012 and ADR-013 define audited candidate promotion plus the per-repository mutation-delivery queue for Phase 2.6.
- ADR-014 defines the accepted Cloudflare-hosted primary execution/control-plane profile and amends the earlier GitHub-Actions-first implementation assumptions without changing the core ADR-012/ADR-013 safety invariants.

## Immediate milestone — Phase 2.6 Memory Write Delivery Pipeline

Phase 2.6 is the current engineering milestone. **Phase 3 MCP implementation is blocked until Phase 2.6 satisfies its exit criteria in issue #20.**

The goal is to give remote semantic callers a permanent deterministic delivery path that invokes the existing MemoryService, keeps unaudited candidates off canonical GitHub `main`, serializes hosted mutation work per repository, and performs each distinct correctness gate once. This milestone does not create a second memory implementation and does not turn the Core development pipeline into a reduced-safety fast mode.

The governing architectural decisions are:

- ADR-012 — audited candidate promotion for external memory delivery;
- ADR-013 — per-repository serialized mutation-delivery queue;
- ADR-014 — Cloud-hosted Phase 2.6 memory-delivery control plane.

Phase 2.6 implementation priorities are:

1. define the authenticated, transport-neutral hosted delivery request/status/cancel boundary plus an explicit authorization binding between the caller and an authorized GitHub App installation/repository;
2. implement one stateful coordinator per immutable GitHub repository identity using a Cloudflare Durable Object, with explicit lane states including `OPEN`, `SUSPENDED`, `MAINTENANCE`, and reconciliation;
3. serialize the entire heavy finalization/audit/publication workflow per repository in v1, so queued operations made stale by a preceding hosted commit become `NEEDS_REPREPARE` before starting a Container;
4. use one Cloudflare Workflow per admitted operation for durable, idempotent execution checkpoints/retries while keeping the Durable Object as the sole repository-lane authority;
5. isolate the GitHub App private key in a private internal GitHub gateway/publisher Worker reached only through an internal service binding; the public API Worker and finalizer/auditor containers never receive that long-lived key;
6. run the existing deterministic Runethread Go/Core + Git finalizer in a Cloudflare Container; a cold finalizer targets at most one GitHub source clone/fetch and a warm clone may be reused only as a disposable cache after clean exact refresh;
7. let `ApplyMutation` perform its existing candidate construction, Index v2 write, hard validation, and local-only fast-forward exactly once rather than recreating those semantics in provider code;
8. export exact candidate `C` plus its operation/base/runtime/contract bindings as private immutable/content-addressed candidate evidence; optimize candidate transport for total transferred work, allowing either a full package or a delta over exact `H0`;
9. independently audit exact `C` in a fresh reduced-privilege container with hard validation, strict Index v2 freshness, candidate/request/runtime binding, and expected-diff checks, using only the minimum read-only base fetch needed to reconstruct `C` rather than enforcing an arbitrary zero-clone rule;
10. publish only exact audited `C` through the least-privilege Runethread GitHub App and an atomic expected-old-revision compare-and-swap; prototype clone-free Git-object publication but fall back to exact candidate push if exact commit identity cannot be reproduced;
11. after successful publication, confirm `main == C` cheaply rather than running another full synchronous clone/validation cycle;
12. subscribe the GitHub App to signed `push` webhooks for fast canonical-movement observation while continuing to re-read exact canonical state directly for correctness at admission and publication;
13. distinguish known queued staleness (`NEEDS_REPREPARE`) from unexpected movement during the one active hosted workflow (`RECONCILIATION_REQUIRED`);
14. keep GitHub Free and paid private repositories on the same normal hosted mutation architecture, using paid branch/ruleset protection only as optional defense-in-depth;
15. preserve singleton operations, exact idempotent retry, cancellation-before-publication, audit-failure suspension, and exclusive control-plane barriers;
16. enforce explicit hosted request/repository/artifact/runtime/retry limits plus private-data/log/retention controls so quota or provider failures leave canonical Git unchanged;
17. keep project `current-state.md`/orientation prose outside the atomic memory dual-write transaction; future refresh of those project views is a separate materialized-view/projection concern;
18. remove push-on-every-normal-memory full GitHub Actions validation from the hosted data-plane path through the proper managed workflow/control-plane rollout, while retaining Actions where health/recovery/migration/control-plane checks prove a distinct invariant;
19. roll the finished mechanism through the required release/downstream gates and a real private memory repository, then measure cold/warm source acquisition, bytes transferred, finalization, packaging, audit, publication, and end-to-end latency/cost separately.

Phase 2.6 v1 deliberately uses singleton operations. The following remain deferred:

- semantic memory/read dependency quantification;
- neighboring-operation batching/coalescing;
- automatic semantic re-preparation;
- general AI task orchestration/provider execution;
- project-orientation projection refresh beyond what is needed to preserve current correctness;
- Phase 3 MCP transport implementation itself.

Cloudflare is the primary hosted execution/control-plane provider for Phase 2.6, but Core remains provider-independent and local/offline MemoryService operation must continue to work without Cloudflare. The hosted service account requires Workers Paid because Containers are needed for the real Linux/Go/Git runtime; end users do not require Cloudflare accounts or paid Cloudflare plans.

GitHub remains the user-owned canonical repository. GitHub Actions is no longer the primary normal write executor/queue. It remains useful for independent repository-health, migration, recovery, and control-plane validation where those checks prove a distinct invariant rather than merely repeating the exact candidate audit after every memory write.

## Next phase after Phase 2.6

After issue #20 exits green and the Phase 2.6 implementation/rollout is independently verified, Phase 3 may begin as a thin MCP adapter over the same MemoryService application boundary and hosted delivery lifecycle established here. Remote MCP may use the hosted delivery API; local MCP may invoke local MemoryService directly. MCP must remain transport rather than a second owner of storage, lifecycle, provenance, trust, indexing, idempotency, concurrency, audit, or Git transaction behavior.

## Engineering procedure

All substantive changes follow `docs/runethread/ENGINEERING_PROCESS.md` and `docs/runethread/DEVELOPMENT_PIPELINE.md`. Validation CI is read-only and must never patch/push source.

## Historical / superseded work

- PR #13 (`superseded: decouple runtime and contract releases`) remains closed as design-forensics history and must not be revived or used as an implementation base.
- Phase 2.5 design and completion evidence remain recorded in issue #14 and ADR-011.
- The 2026-09-04 milestone pointer that made Phase 3 immediately current is superseded by the Phase 2.6 sequencing decision.
- The original ADR-012/ADR-013 GitHub-Actions-first execution profile is amended by ADR-014; their core audited-candidate and per-repository-serialization invariants remain accepted.
