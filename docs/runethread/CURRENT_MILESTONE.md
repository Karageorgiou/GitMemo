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
- ADR-015 records the next-contract decision that project current-state/overview documents become asynchronous orientation/materialized projections; contract v8 remains immutable and still requires relevant current-state synchronization until explicit migration.
- ADR-016 hardens the hosted trust boundary: normal hosted writes require the ADR-015 contract semantics, finalizer/auditor evidence-write capabilities are separated, live evidence cannot expire while referenced, and exact publication has a concrete minimal Git publisher executor unless a future GitHub API proves true expected-old ref CAS.

## Immediate milestone — Phase 2.6 Memory Write Delivery Pipeline

Phase 2.6 is the current engineering milestone. **Phase 3 MCP implementation is blocked until Phase 2.6 satisfies its exit criteria in issue #20.**

The goal is to give remote semantic callers a permanent deterministic delivery path that invokes the existing MemoryService, keeps unaudited candidates off the authorized canonical Git ref, serializes hosted mutation work per repository, and performs each distinct correctness gate once. This milestone does not create a second memory implementation and does not turn the Core development pipeline into a reduced-safety fast mode.

The governing architectural decisions are:

- ADR-012 — audited candidate promotion for external memory delivery;
- ADR-013 — per-repository serialized mutation-delivery queue;
- ADR-014 — Cloud-hosted Phase 2.6 memory-delivery control plane;
- ADR-015 — project current-state documents become asynchronous orientation projections in the next operational contract;
- ADR-016 — hosted contract eligibility, evidence capability/retention, and exact publication hardening.

### Mandatory contract-v9 prerequisite

The released contract v8 explicitly requires relevant project current-state synchronization when a memory materially changes present project state and treats affected current-state synchronization as part of memory-write completion. Phase 2.6 therefore MUST NOT rely on asynchronously stale project views for a contract-v8 repository.

After the architecture-freeze gate passes, the **first implementation slice** is the contract change implementing ADR-015:

1. create the next operational contract version/release (target contract v9) with the project-view completion/freshness semantics defined by ADR-015;
2. add exact contract-v8 historical fixture/migration/compatibility/trust/bootstrap coverage;
3. publish and independently verify the immutable contract/runtime release anchor;
4. migrate and permanently validate `runethread/memory-template`;
5. migrate and permanently validate the known private memory repository, preserving canonical atomic-memory identities/content/provenance/relationships and existing project-view bytes unless the reviewed contract change explicitly requires otherwise;
6. only after a repository is explicitly migrated may hosted Phase 2.6 treat project current-state/overview prose as an asynchronously refreshed projection rather than an atomic-memory completion dependency.

A contract-v8 repository continues to obey the v8 synchronization rule. Hosted delivery MUST NOT silently reinterpret v8 as v9. Under ADR-016, **normal hosted Phase 2.6 memory mutation is not admitted for contract-v8 repositories at all**. A v8 repository may be read, verified, reconciled, and upgraded, but normal hosted write admission requires the explicit projection-capable contract migration (target v9) or a later contract explicitly supported by the hosted release.

Phase 2.6 hosted implementation priorities after/alongside that prerequisite are:

1. define the authenticated, transport-neutral hosted delivery request/status/cancel boundary plus explicit authorization binding between caller and GitHub App installation/repository;
2. bind each authorized repository to immutable repository identity + App installation + explicit canonical branch ref (normally discovered from default branch at adoption) + last accepted revision; never silently follow default-branch rename/change/deletion/transfer;
3. store sealed private request bodies as short-retention private content-addressed/no-overwrite objects and keep only opaque references/digests in Durable Object, log, and status state;
4. mediate private request/candidate/finalization/audit object creation through role-, artifact-class-, repository-, operation-, phase-, generation-, key-, and digest-bound capabilities; finalizer cannot create authoritative audit receipts, auditor cannot create/replace finalization evidence, and neither receives publication authority;
5. keep Core idempotency identity separate from hosted operation-attempt identity; hosted attempt identity binds immutable repository + canonical-ref identity + exact sealed-request digest while Core remains authoritative for semantic equivalence/conflict;
6. implement one stateful coordinator per immutable repository identity using a Cloudflare Durable Object, with transactional SQLite for bounded queue, one active operation, phase/execution generation, retry/backoff/deadlines, evidence references, canonical-ref binding, and lane states including `OPEN`, `SUSPENDED`, `MAINTENANCE`, and reconciliation;
7. use the DO as sole hosted lane/operation-state authority and drive the active operation through idempotent `drive()` logic plus one at-least-once alarm; explicitly reschedule prolonged retryable failures rather than relying only on finite provider alarm retries;
8. treat DO async interleaving as real: before every Container/object-store/GitHub external action, atomically claim phase + execution generation, persist it, perform external I/O without long `blockConcurrencyWhile()`, then compare active operation/phase/generation before accepting result; stale-generation outputs never advance state;
9. report `ACCEPTED` only after durable operation/request-reference state and a recoverable due alarm are established; exact resubmission/status/recovery repairs stored work with missing alarm;
10. serialize the entire hosted finalization/audit/publication operation per repository in v1, but preserve ADR-003 ordering: canonical Core/Git committed-idempotency lookup happens before stale classification, so stale work stops before candidate construction/Index write/package/audit, not necessarily before cold Container/source acquisition;
11. isolate the long-lived GitHub App private key in a private internal GitHub gateway Worker; public API has no publication binding, runtime App requests minimum Contents/Metadata authority rather than Administration/Workflows, and finalizer/auditor receive only narrow short-lived read tokens when required;
12. run existing deterministic Runethread Go/Core + Git finalizer in repository runtime's attached Container; cold finalization targets at most one GitHub source clone/fetch and warm clone reuse is only untrusted disposable cache after exact refresh/reset to directly observed canonical ref/revision;
13. never retry `ApplyMutation` against surviving local unpromoted candidate history; every fresh finalization invocation starts from direct remote canonical state;
14. make hosted finalization idempotent: persist complete exact candidate evidence first and create immutable attempt/generation-bound finalization receipt last; valid receipt selects authoritative result while retry without receipt resets/reconstructs from remote canonical state;
15. let `ApplyMutation` preserve committed-idempotency-before-stale ordering and perform candidate construction, one Index v2 write, hard validation, commit creation, and local-only fast-forward rather than recreating those semantics in provider code;
16. treat `NO_OP` as a Core-validated terminal outcome that skips candidate/audit/publication, not provider shortcut or invented Git commit;
17. distinguish request-local mutation failure from canonical repository/trust/compatibility/ref-binding failure; unhealthy accepted canonical base suspends/reconciles lane;
18. export exact candidate C plus repository/canonical-ref/operation/base/runtime/contract bindings as private content-addressed, digest-verified, no-overwrite evidence through the finalizer's restricted evidence capability;
19. independently audit exact C in separate fresh reduced-privilege Container/DO context with hard validation, strict Index v2 freshness, binding/scope checks, only minimum exact-base read needed, and only the audit-specific evidence capability;
20. make audit completion generation-bound/idempotent through immutable audit evidence/receipt; deterministic audit failure becomes durable lane suspension/reconciliation before release;
21. only repository DO may atomically transition `AUDITED -> PUBLISHING` after rechecking cancellation, generation, lane, evidence, repository/App/canonical-ref authorization, direct canonical-ref state, and barrier;
22. after durable `PUBLISHING` authorization, have the private App gateway mint a short-lived one-repository minimum Contents-write installation token for a minimal trusted Git publisher executor/Container; that executor imports exact audited candidate Git objects, performs no semantic work and no source clone, and attempts only exact bound-ref `H0 -> C` publication;
23. require a real expected-old Git ref update. The currently documented GitHub REST ref-update fast-forward operation alone is not exact CAS and is not the v1 correctness mechanism; a future clone-free API path may replace the publisher executor only after authoritative documentation and integration tests prove exact candidate identity **and** atomic expected-old-`H0` semantics;
24. once `PUBLISHING` wins, cancellation is no longer a correctness mechanism; ambiguous publication resolves by exact bound-ref state (`C` committed, `H0` same exact retry allowed by policy, anything else reconciliation);
25. after success confirm bound canonical ref == C cheaply rather than another full clone/validation cycle;
26. use signed GitHub push webhooks only for fast observation; every relevant webhook triggers direct bound-ref read and payload never directly mutates accepted state;
27. distinguish proven-uncommitted stale work (`NEEDS_REPREPARE`) from unexpected bound-ref movement (`RECONCILIATION_REQUIRED`);
28. keep GitHub Free and paid private repositories on same hosted architecture, paid branch/ruleset protection optional defense-in-depth;
29. preserve singleton operations, exact committed retry, cancellation-before-publication, audit-failure suspension, explicit recovery, and control-plane/canonical-ref-binding barriers;
30. version hosted delivery protocol/release and treat breaking Worker/DO/Container/evidence/publisher/control-path changes as barriers; incompatible rollout uses drain/maintenance or versioned blue/green isolation and an operation never crosses incompatible generations;
31. enforce explicit hosted request/repository/artifact/runtime/retry/operation-history/private-data/log/retention limits so quota/provider failures leave canonical Git unchanged;
32. make evidence retention reference-aware: queued/active/retrying/audited/publishing/reconciling operations keep every object still required for progress or exact recovery; only unreferenced/orphan or safely terminal evidence after its bounded recovery/incident window is GC-eligible, and automatic provider TTL must never undercut that liveness rule;
33. after contract-v9 migration, keep project current-state/overview prose outside atomic memory dual-write transaction; refresh remains separate projection concern under ADR-015;
34. remove push-on-every-normal-memory full GitHub Actions validation through proper managed rollout, retaining Actions only for distinct health/PR/recovery/migration/control-plane checks;
35. roll mechanism through release/downstream gates and real private memory repo, then measure cold/warm acquisition, bytes, idempotency/stale preflight, finalization, packaging, audit, publication, publisher-executor startup, alarm/interleaving/provider startup, and end-to-end latency/cost.

### Pre-implementation architecture-freeze gate

Phase 2.6 implementation code MUST NOT begin merely because ADRs are drafted or CI is green. The exact current ADR/planning state must complete a fresh adversarial architecture review covering correctness, contract compatibility, state ownership, concurrency/interleaving, crash/retry behavior, privilege boundaries, evidence authority/retention, exact remote publication, deployment/version skew, privacy/resource limits, canonical-ref behavior, and avoidable latency/duplication.

The review passes only when it produces **zero required architecture or planning edits**. Any material correction, simplification, missing invariant, contract prerequisite, or changed implementation boundary is recorded first and resets the gate; the full review then repeats against the new exact head.

The attack review completed on 2026-09-05 against pre-amendment head `68549677e0fbb76b0018ce3aaa574c1d1ba4e1bb` found material edits and produced ADR-016 plus synchronized planning changes. **That review therefore failed the zero-edit gate and cannot unlock implementation.** Per the explicit user instruction for that review cycle, no fresh attack run is started immediately after these edits. A later implementation-unlocking review, if requested, must start from the new exact synchronized head and must itself require zero edits.

Prototype questions may remain only when accepted ADRs already provide a safe invariant-preserving fallback and architecture does not depend on guessing the result. The publisher now has a concrete Git-protocol fallback rather than depending on the unproven REST-ref-CAS prototype.

Phase 2.6 v1 deliberately uses singleton operations. Deferred work includes semantic dependency quantification, batching/coalescing, automatic semantic re-preparation, general AI task orchestration/provider execution, actual project-orientation projection-refresh machinery beyond contract-v9 correctness semantics, and Phase 3 MCP transport itself.

Cloudflare is the primary hosted provider, but Core remains provider-independent and local/offline MemoryService continues without Cloudflare. Hosted service account requires Workers Paid because Containers are needed for Linux/Go/Git execution; end users do not require Cloudflare accounts or paid Cloudflare plans.

GitHub remains user-owned canonical repository. GitHub Actions is no longer primary normal write executor/queue and remains useful only where independent health/PR/migration/recovery/control-plane validation proves a distinct invariant.

## Next phase after Phase 2.6

After issue #20 exits green and Phase 2.6 implementation/rollout is independently verified, Phase 3 may begin as thin MCP adapter over same MemoryService application boundary and hosted delivery lifecycle. Remote MCP may use hosted delivery; local MCP may invoke local MemoryService directly. MCP remains transport rather than second owner of storage, lifecycle, provenance, trust, indexing, idempotency, concurrency, audit, or Git transaction behavior.

## Engineering procedure

All substantive changes follow `docs/runethread/ENGINEERING_PROCESS.md` and `docs/runethread/DEVELOPMENT_PIPELINE.md`. Validation CI is read-only and must never patch/push source.

## Historical / superseded work

- PR #13 (`superseded: decouple runtime and contract releases`) remains closed as design-forensics history and must not be revived or used as implementation base.
- Phase 2.5 design/completion evidence remain in issue #14 and ADR-011.
- 2026-09-04 milestone pointer that made Phase 3 immediately current is superseded by Phase 2.6 sequencing.
- original ADR-012/013 GitHub-Actions-first profile is amended by ADR-014; core audited-candidate/per-repo invariants remain accepted.
- contract v8 remains immutable historical authority. ADR-015 is not retroactive; its asynchronous projection semantics begin only in the explicit next-contract release/migration.
- ADR-016 amends Phase 2.6 hosted admission/evidence/publication details without changing Core memory semantics or GitHub canonical-state ownership.