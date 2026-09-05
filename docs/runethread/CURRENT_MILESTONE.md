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

The goal is to give remote semantic callers a permanent deterministic delivery path that invokes the existing MemoryService, keeps unaudited candidates off the authorized canonical Git ref, serializes hosted mutation work per repository, and performs each distinct correctness gate once. This milestone does not create a second memory implementation and does not turn the Core development pipeline into a reduced-safety fast mode.

The governing architectural decisions are:

- ADR-012 — audited candidate promotion for external memory delivery;
- ADR-013 — per-repository serialized mutation-delivery queue;
- ADR-014 — Cloud-hosted Phase 2.6 memory-delivery control plane.

Phase 2.6 implementation priorities are:

1. define the authenticated, transport-neutral hosted delivery request/status/cancel boundary plus an explicit authorization binding between the caller and an authorized GitHub App installation/repository;
2. bind each authorized repository to immutable repository identity + App installation + an explicit canonical branch ref (normally discovered from the default branch at adoption) + last accepted revision; never silently follow a default-branch rename/change/deletion/transfer;
3. store sealed private request bodies as short-retention private content-addressed/no-overwrite objects and keep only opaque references/digests in Durable Object, log, and status state;
4. keep Core idempotency identity separate from hosted operation-attempt identity; hosted attempt identity binds immutable repository + canonical-ref identity + exact sealed-request digest while Core remains authoritative for semantic equivalence/conflict;
5. implement one stateful coordinator per immutable repository identity using a Cloudflare Durable Object, with transactional SQLite state for bounded queue, one active operation, phase/execution generation, retry/backoff/deadlines, evidence references, canonical-ref binding, and lane states including `OPEN`, `SUSPENDED`, `MAINTENANCE`, and reconciliation;
6. use the DO as the sole hosted lane/operation-state authority and drive the active operation through idempotent `drive()` logic plus one at-least-once alarm; explicitly reschedule prolonged retryable failures rather than relying only on finite provider alarm retries;
7. treat DO async interleaving as real: before every Container/R2/GitHub external action, atomically claim phase + execution generation in SQLite, persist it, perform external I/O without long `blockConcurrencyWhile()`, then compare active operation/phase/generation before accepting the result; stale-generation outputs never advance state;
8. report `ACCEPTED` only after durable operation/request-reference state and a recoverable due alarm are established; exact resubmission/status/recovery must repair stored work with a missing alarm;
9. serialize the entire hosted finalization/audit/publication operation per repository in v1, but preserve ADR-003 ordering: canonical Core/Git committed-idempotency lookup happens before stale classification, so stale work stops before candidate construction/Index write/package/audit, not necessarily before cold Container/source acquisition;
10. isolate the GitHub App private key in a private internal GitHub gateway/publisher Worker; the public API has no publication binding, the baseline runtime App requests minimum Contents/Metadata authority, not Administration or Workflows permission, and finalizer/auditor receive only narrow short-lived read tokens when required;
11. run the existing deterministic Runethread Go/Core + Git finalizer in the repository runtime's attached Cloudflare Container; cold finalization targets at most one GitHub source clone/fetch and warm clone reuse is only an untrusted disposable cache after exact refresh/reset to the directly observed canonical ref/revision;
12. never retry `ApplyMutation` against a surviving local unpromoted candidate branch: every fresh finalization invocation starts from direct remote canonical state because local candidate `C` already contains operation metadata and must not be mistaken for canonical committed evidence;
13. make hosted finalization idempotent: persist complete exact candidate evidence first and create an immutable attempt/generation-bound finalization receipt last; create-if-absent receipt/evidence rules select the authoritative result while a retry with no valid receipt resets/reconstructs from remote canonical state;
14. let `ApplyMutation` preserve committed-idempotency-before-stale ordering and perform candidate construction, one Index v2 write, hard validation, commit creation, and local-only fast-forward rather than recreating those semantics in provider code; transport hashes do not replace Core's semantic request fingerprint;
15. treat `NO_OP` as a Core-validated terminal outcome that skips candidate/audit/publication, not a provider-side shortcut or invented Git commit;
16. distinguish request-local mutation failure from canonical repository/trust/compatibility/ref-binding failure: an unhealthy accepted canonical base suspends/reconciles the lane instead of producing endless unrelated `FINALIZATION_FAILED` operations;
17. export exact candidate `C` plus repository/canonical-ref/operation/base/runtime/contract bindings as private content-addressed, digest-verified, no-overwrite evidence; receipt/evidence mismatch is integrity failure and orphan data without a valid receipt is nonauthoritative;
18. independently audit exact `C` in a separate fresh reduced-privilege Container/DO context with hard validation, strict Index v2 freshness, candidate/request/runtime/ref binding, and expected-diff checks, using only minimum exact-base read needed to reconstruct `C`;
19. make audit completion generation-bound/idempotent through immutable audit evidence/receipt; deterministic audit failure becomes durable lane suspension or conservative reconciliation before the active lane can release;
20. return exact audit evidence to the repository DO; only that lane authority may atomically transition `AUDITED -> PUBLISHING` after rechecking cancellation, current generation, lane state, evidence, repository/App/canonical-ref authorization, direct canonical-ref state, and control-plane barrier;
21. publish only exact DO-authorized audited `C` through the private GitHub gateway and atomic expected-old-revision CAS against the **bound canonical ref**; prototype clone-free Git-object publication but use exact-candidate push fallback if exact commit identity cannot be reproduced;
22. once `PUBLISHING` wins, cancellation is no longer a correctness mechanism; after crash/response loss resolve exact bound-ref state (`C`, `H0`, or unexpected/reconciliation) and retry only the same exact authorized publication where appropriate;
23. after successful publication, confirm bound canonical ref == `C` cheaply rather than running another full synchronous clone/validation cycle;
24. use signed GitHub `push` webhooks only for fast observation; every relevant webhook triggers direct read of the bound canonical ref and stale/out-of-order payloads never directly mutate accepted state;
25. distinguish proven uncommitted stale work (`NEEDS_REPREPARE`) from unexpected bound-ref movement during the one active hosted operation (`RECONCILIATION_REQUIRED`);
26. keep GitHub Free and paid private repositories on the same hosted architecture, using paid branch/ruleset protection only as optional defense-in-depth;
27. preserve singleton operations, exact committed retry, cancellation-before-publication, audit-failure suspension, explicit recovery, and exclusive control-plane/canonical-ref-binding barriers;
28. version the hosted delivery protocol/release and treat breaking Worker/Container/control-path changes as barriers; provider rollout is not assumed atomic, so incompatible changes require draining/maintenance or versioned blue/green isolation;
29. enforce explicit hosted request/repository/artifact/runtime/retry/operation-history limits plus private-data/log/retention controls so quota/provider failures leave canonical Git unchanged;
30. keep project `current-state.md`/orientation prose outside the atomic memory dual-write transaction; future refresh is a separate materialized-view/projection concern;
31. remove push-on-every-normal-memory full GitHub Actions validation from the hosted data-plane path through proper managed rollout, retaining Actions only where health/recovery/migration/control-plane checks prove a distinct invariant;
32. roll the mechanism through required release/downstream gates and a real private memory repo, then measure cold/warm acquisition, bytes, idempotency/stale preflight, finalization, packaging, audit, publication, alarm/retry/interleaving overhead, and end-to-end latency/cost separately.

### Pre-implementation architecture-freeze gate

Phase 2.6 implementation code MUST NOT begin merely because ADR-014 is drafted or CI is green. The exact current ADR/planning state must complete a fresh adversarial architecture review covering correctness, state ownership, concurrency/interleaving, crash/retry behavior, privilege boundaries, deployment/version skew, privacy/resource limits, canonical-ref behavior, and avoidable latency/duplication.

The review passes only when it produces **zero required architecture or planning edits**. Any material correction, simplification, missing invariant, or changed implementation boundary is recorded first and resets the gate; the full review then repeats against the new exact head. A previous review does not count after the planning head changes.

Prototype questions may remain only when accepted ADRs already provide a safe invariant-preserving fallback and architecture does not depend on guessing the result.

Phase 2.6 v1 deliberately uses singleton operations. Deferred work includes semantic dependency quantification, neighboring-operation batching/coalescing, automatic semantic re-preparation, general AI task orchestration/provider execution, project-orientation projection refresh beyond current correctness needs, and Phase 3 MCP transport implementation itself.

Cloudflare is the primary hosted execution/control-plane provider, but Core remains provider-independent and local/offline MemoryService operation continues without Cloudflare. The hosted service account requires Workers Paid because Containers are needed for Linux/Go/Git execution; end users do not require Cloudflare accounts or paid Cloudflare plans.

GitHub remains the user-owned canonical repository. GitHub Actions is no longer the primary normal write executor/queue and remains useful only where independent repository-health, PR, migration, recovery, or control-plane validation proves a distinct invariant.

## Next phase after Phase 2.6

After issue #20 exits green and Phase 2.6 implementation/rollout is independently verified, Phase 3 may begin as a thin MCP adapter over the same MemoryService application boundary and hosted delivery lifecycle. Remote MCP may use hosted delivery; local MCP may invoke local MemoryService directly. MCP remains transport rather than a second owner of storage, lifecycle, provenance, trust, indexing, idempotency, concurrency, audit, or Git transaction behavior.

## Engineering procedure

All substantive changes follow `docs/runethread/ENGINEERING_PROCESS.md` and `docs/runethread/DEVELOPMENT_PIPELINE.md`. Validation CI is read-only and must never patch/push source.

## Historical / superseded work

- PR #13 (`superseded: decouple runtime and contract releases`) remains closed as design-forensics history and must not be revived or used as an implementation base.
- Phase 2.5 design and completion evidence remain recorded in issue #14 and ADR-011.
- The 2026-09-04 milestone pointer that made Phase 3 immediately current is superseded by the Phase 2.6 sequencing decision.
- The original ADR-012/ADR-013 GitHub-Actions-first execution profile is amended by ADR-014; their core audited-candidate and per-repository-serialization invariants remain accepted.
