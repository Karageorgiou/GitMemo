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
2. store sealed private request bodies as short-retention private content-addressed objects and keep only opaque references/digests in Durable Object, log, and status state;
3. keep Core idempotency identity separate from hosted operation-attempt identity; hosted attempt identity binds immutable repository identity plus the exact sealed-request digest so byte-different envelopes sharing one Core key cannot alias one hosted attempt, while Core remains authoritative for semantic equivalence/conflict;
4. implement one stateful coordinator per immutable GitHub repository identity using a Cloudflare Durable Object, with transactional SQLite state for the bounded queue, one active operation, retry/backoff deadlines, evidence references, and lane states including `OPEN`, `SUSPENDED`, `MAINTENANCE`, and reconciliation;
5. use the Durable Object as the sole hosted lane/operation-state authority and drive the one active operation through an idempotent `drive()` state machine; use one alarm as an at-least-once scheduler/recovery wakeup and explicitly reschedule retry/backoff rather than depending only on the platform's finite automatic alarm retries;
6. serialize the entire hosted finalization/audit/publication operation per repository in v1, but preserve ADR-003 ordering: canonical Core/Git committed-idempotency lookup happens before stale classification, so stale work is guaranteed to stop before candidate construction/Index write/package/audit, not necessarily before cold Container/source acquisition;
7. isolate the GitHub App private key in a private internal GitHub gateway/publisher Worker; the public API has no publication binding, the baseline runtime App requests only minimum Contents/Metadata authority, not Administration or Workflows permission, and finalizer/auditor receive only narrow short-lived read tokens when required;
8. run the existing deterministic Runethread Go/Core + Git finalizer in the repository runtime's attached Cloudflare Container; a cold finalizer targets at most one GitHub source clone/fetch and a warm clone may be reused only as an untrusted disposable cache after exact refresh/reset to direct remote canonical state;
9. never retry `ApplyMutation` against a surviving local unpromoted candidate branch: every fresh finalization invocation starts from direct remote canonical state because local candidate `C` already contains operation metadata and must not be mistaken for canonical committed evidence;
10. make hosted finalization idempotent: after `ApplyMutation` succeeds, persist complete exact candidate evidence first and create an immutable attempt-bound finalization receipt last; create-if-absent receipt/evidence rules select the authoritative candidate if overlapping retries race, while a retry with no valid receipt resets/reconstructs from remote canonical state;
11. let `ApplyMutation` preserve its existing committed-idempotency-before-stale ordering and perform candidate construction, one Index v2 write, hard validation, commit creation, and local-only fast-forward rather than recreating those semantics in provider code; transport hashes bind artifacts but do not replace Core's semantic request fingerprint;
12. treat `NO_OP` as a Core-validated terminal outcome that skips candidate/audit/publication, not as a provider-side shortcut or invented Git commit;
13. distinguish request-local mutation failure from canonical repository/trust/compatibility failure: an unhealthy accepted canonical base suspends/reconciles the lane instead of producing endless unrelated `FINALIZATION_FAILED` operations;
14. export exact candidate `C` plus its operation/base/runtime/contract bindings as private content-addressed, digest-verified, no-overwrite candidate evidence; receipt/evidence mismatch is an integrity failure, while orphan package data without a valid receipt is nonauthoritative and may be garbage-collected;
15. independently audit exact `C` in a separate fresh reduced-privilege Container/DO context with hard validation, strict Index v2 freshness, candidate/request/runtime binding, and expected-diff checks, using only the minimum read-only exact-base source acquisition needed to reconstruct `C`;
16. make audit completion idempotent through immutable audit evidence/receipt; deterministic audit failure must become durable lane suspension or conservative reconciliation before the active lane can be released;
17. return exact audit evidence to the repository Durable Object; only that lane authority may atomically transition the active operation from `AUDITED` to `PUBLISHING` after rechecking cancellation, suspension, maintenance, operation identity, evidence binding, authorization, and directly observed canonical state;
18. publish only the exact DO-authorized audited `C` through the private GitHub gateway and an atomic expected-old-revision compare-and-swap; prototype clone-free Git-object publication but fall back to an exact candidate push if exact commit identity cannot be reproduced;
19. once `PUBLISHING` wins, cancellation is no longer a correctness mechanism; after crash/response loss, resolve exact Git state (`C`, `H0`, or unexpected/reconciliation) and retry only the same exact authorized publication when appropriate;
20. after successful publication, confirm `main == C` cheaply rather than running another full synchronous clone/validation cycle;
21. subscribe the GitHub App to signed `push` webhooks only for fast observation; every webhook is a hint that triggers a direct canonical-ref read, and stale/out-of-order webhook payloads never directly mutate accepted repository state;
22. distinguish proven uncommitted stale work (`NEEDS_REPREPARE`) from unexpected canonical movement during the one active hosted operation (`RECONCILIATION_REQUIRED`);
23. keep GitHub Free and paid private repositories on the same normal hosted mutation architecture, using paid branch/ruleset protection only as optional defense-in-depth;
24. preserve singleton operations, exact committed retry, cancellation-before-publication, audit-failure suspension, explicit recovery, and exclusive control-plane barriers;
25. version the hosted delivery protocol/release and treat breaking Worker/Container/control-path changes as control-plane barriers; provider rollout is not assumed atomic, so incompatible changes require draining/maintenance or versioned blue/green isolation;
26. enforce explicit hosted request/repository/artifact/runtime/retry/operation-history limits plus private-data/log/retention controls so quota or provider failures leave canonical Git unchanged;
27. keep project `current-state.md`/orientation prose outside the atomic memory dual-write transaction; future refresh of those project views is a separate materialized-view/projection concern;
28. remove push-on-every-normal-memory full GitHub Actions validation from the hosted data-plane path through the proper managed workflow/control-plane rollout, while retaining Actions where health/recovery/migration/control-plane checks prove a distinct invariant;
29. roll the finished mechanism through the required release/downstream gates and a real private memory repository, then measure cold/warm source acquisition, bytes transferred, idempotency/stale preflight, finalization, packaging, audit, publication, alarm/retry overhead, and end-to-end latency/cost separately.

### Pre-implementation architecture-freeze gate

Phase 2.6 implementation code MUST NOT begin merely because ADR-014 has been drafted or CI is green. Before the first implementation branch/repository work begins, the exact current ADR/planning state must complete a fresh adversarial architecture review covering correctness, state ownership, concurrency, crash/retry behavior, privilege boundaries, deployment/version skew, privacy/resource limits, and avoidable latency/duplication.

The review is considered a pass only when it produces **zero required architecture or planning edits**. If the review identifies a material correction, simplification, missing invariant, or changed implementation boundary, record that change first and repeat the full review against the new exact head. A previous review does not count after the architecture/planning head changes.

Only a zero-edit adversarial review of the exact architecture/planning head unlocks Phase 2.6 implementation. Prototype questions already explicitly delegated to implementation by the accepted ADRs may remain open when they have a safe invariant-preserving fallback and do not require choosing architecture by guesswork.

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
