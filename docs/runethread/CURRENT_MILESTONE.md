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
- ADR-017 hardens reconciliation/privacy/publication lifetime: ordinary out-of-band adoption must preserve accepted Git history, private repository visibility is a live hosted-write eligibility condition, `PUBLISHING` remains fenced until issued publisher capabilities are quiesced, and the managed validation-workflow trigger change must ship through the released upgrader/downstream migration.
- ADR-018 hardens publication-history recovery: a candidate that is proven or possibly published remains a protected history anchor until its outcome/ancestry is resolved, so a later rewrite cannot erase committed-idempotency evidence by falling back to the older accepted base.
- ADR-019 hardens destructive hosted-state recovery and managed support: rollback-sensitive acceptance/cancellation/history/publication facts survive Durable Object rollback in a small append-only safety journal, PITR/recreation is an explicit recovery barrier, retained private-memory Actions use immutable full-SHA pins, and contract-v9 support prose is migrated without silently overwriting customized README/workflow state.
- ADR-020 hardens the prepublication trust boundary: the fresh auditor must independently prove that exact candidate `C` contains the Core-derived semantic result of the exact sealed request against exact `H0`; valid/index-fresh/in-scope bytes plus a copied request fingerprint are not sufficient.
- ADR-021 closes the remaining finalizer-result bypass: `NO_OP` and `ALREADY_COMMITTED` are successful terminal semantic claims and must be independently re-proven in a fresh reduced-privilege Core/repository context before becoming durable client-visible success or releasing the lane.

## Immediate milestone — Phase 2.6 Memory Write Delivery Pipeline

Phase 2.6 is the current engineering milestone. **Phase 3 MCP implementation is blocked until Phase 2.6 satisfies its exit criteria in issue #20.**

The goal is to give remote semantic callers a permanent deterministic delivery path that invokes the existing MemoryService, keeps unaudited candidates off the authorized canonical Git ref, serializes hosted mutation work per repository, and performs each distinct correctness gate once. This milestone does not create a second memory implementation and does not turn the Core development pipeline into a reduced-safety fast mode.

The governing architectural decisions are:

- ADR-012 — audited candidate promotion for external memory delivery;
- ADR-013 — per-repository serialized mutation-delivery queue;
- ADR-014 — Cloud-hosted Phase 2.6 memory-delivery control plane;
- ADR-015 — project current-state documents become asynchronous orientation projections in the next operational contract;
- ADR-016 — hosted contract eligibility, evidence capability/retention, and exact publication hardening;
- ADR-017 — accepted-history reconciliation, repository privacy eligibility, publisher-capability fencing, and managed validation-bootstrap rollout;
- ADR-018 — proven/possible publication-history preservation across ambiguous completion and out-of-band rewrites;
- ADR-019 — rollback-independent recovery evidence, restore barriers, immutable managed-workflow Action pins, and v9 managed-support authority alignment;
- ADR-020 — independent candidate-to-request conformance audit before publication;
- ADR-021 — independent verification of successful terminal no-candidate mutation results.

### Mandatory contract-v9 prerequisite

The released contract v8 explicitly requires relevant project current-state synchronization when a memory materially changes present project state and treats affected current-state synchronization as part of memory-write completion. Phase 2.6 therefore MUST NOT rely on asynchronously stale project views for a contract-v8 repository.

After the architecture-freeze gate passes, the **first implementation slice** is the released compatibility/migration transition implementing ADR-015 plus the managed support/bootstrap changes required by ADR-017/ADR-019:

1. create the next operational contract version/release (target contract v9) with the project-view completion/freshness semantics defined by ADR-015;
2. add exact contract-v8 historical fixture/migration/compatibility/trust/bootstrap coverage;
3. deliberately transition the Runethread-managed `.github/workflows/validate.yml` through starter/upgrader ownership logic so normal hosted canonical pushes no longer trigger redundant full validation, every retained external `uses:` action is pinned to a verified full-length commit SHA, the exact prior managed workflow remains a supported migration source, and customized/unrecognized workflow state is not silently overwritten;
4. update generated/current Runethread support text which still describes `projects/` current-state/overview prose as canonical, while preserving the actual user project-view bytes; automatic private-memory README replacement is limited to exact recognized prior managed README bytes/source state rather than the current broad heading/lock heuristic;
5. publish and independently verify the immutable contract/runtime release anchor;
6. migrate and permanently validate `runethread/memory-template` with the new contract and supported managed support/bootstrap state;
7. migrate and permanently validate the known private memory repository, preserving canonical atomic-memory identities/content/provenance/relationships and existing project-view bytes unless the reviewed contract change explicitly requires otherwise;
8. only after a repository is explicitly migrated may hosted Phase 2.6 treat project current-state/overview prose as an asynchronously refreshed projection rather than an atomic-memory completion dependency or admit that repository to normal hosted mutation.

A contract-v8 repository continues to obey the v8 synchronization rule. Hosted delivery MUST NOT silently reinterpret v8 as v9. Under ADR-016, **normal hosted Phase 2.6 memory mutation is not admitted for contract-v8 repositories at all**. A v8 repository may be read, verified, reconciled, and upgraded, but normal hosted write admission requires the explicit projection-capable contract migration (target v9) or a later contract explicitly supported by the hosted release.

Phase 2.6 hosted implementation priorities after/alongside that prerequisite are:

1. define the authenticated, transport-neutral hosted delivery request/status/cancel boundary plus explicit authorization binding between caller and GitHub App installation/repository;
2. bind each authorized repository to immutable repository identity + App installation + explicit canonical branch ref (normally discovered from default branch at adoption) + last accepted revision, with directly observed **private** repository visibility as a normal hosted-write eligibility condition; never silently follow default-branch rename/change/deletion/transfer or continue normal writes after observed non-private visibility;
3. store sealed private request bodies as short-retention private content-addressed/no-overwrite objects and keep only opaque references/digests in Durable Object, log, and status state;
4. mediate private request/candidate/finalization/audit object creation through role-, artifact-class-, repository-, operation-, phase-, generation-, key-, and digest-bound capabilities; finalizer cannot create authoritative audit receipts, auditor cannot create/replace finalization evidence, and neither receives publication authority;
5. keep Core idempotency identity separate from hosted operation-attempt identity; hosted attempt identity binds immutable repository + canonical-ref identity + exact sealed-request digest while Core remains authoritative for semantic equivalence/conflict;
6. implement one stateful coordinator per immutable repository identity using a Cloudflare Durable Object, with transactional SQLite for bounded queue, one active operation, phase/execution generation, retry/backoff/deadlines, evidence references, canonical-ref binding, and lane states including `OPEN`, `SUSPENDED`, `MAINTENANCE`, and reconciliation;
7. use the DO as sole hosted lane/operation-state authority and drive the active operation through idempotent `drive()` logic plus one at-least-once alarm; explicitly reschedule prolonged retryable failures rather than relying only on finite provider alarm retries;
8. treat DO async interleaving as real: before every Container/object-store/GitHub external action, atomically claim phase + execution generation, persist it, perform external I/O without long `blockConcurrencyWhile()`, then compare active operation/phase/generation before accepting result; stale-generation outputs never advance state;
9. report `ACCEPTED` only after durable operation/request-reference state, a recoverable due alarm, and ADR-019 rollback-independent acceptance evidence are established; exact resubmission/status/recovery repairs stored work with missing scheduling while destructive DO rollback is handled through the recovery barrier rather than assumed forward-only;
10. serialize the entire hosted finalization/audit/publication operation per repository in v1, but preserve ADR-003 ordering: canonical Core/Git committed-idempotency lookup happens before stale classification, so stale work stops before candidate construction/Index write/package/audit, not necessarily before cold Container/source acquisition;
11. isolate the long-lived GitHub App private key in a private internal GitHub gateway Worker; public API has no publication binding, runtime App requests minimum Contents/Metadata authority rather than Administration/Workflows, and finalizer/auditor receive only narrow short-lived read tokens when required;
12. run existing deterministic Runethread Go/Core + Git finalizer in repository runtime's attached Container; cold finalization targets at most one GitHub source clone/fetch and warm clone reuse is only untrusted disposable cache after exact refresh/reset to directly observed canonical ref/revision;
13. never retry `ApplyMutation` against surviving local unpromoted candidate history; every fresh finalization invocation starts from direct remote canonical state;
14. make hosted finalization idempotent: persist complete exact candidate evidence first and create immutable attempt/generation-bound finalization receipt last; valid receipt selects authoritative result while retry without receipt resets/reconstructs from remote canonical state;
15. let `ApplyMutation` preserve committed-idempotency-before-stale ordering and perform candidate construction, one Index v2 write, hard validation, commit creation, and local-only fast-forward rather than recreating those semantics in provider code;
16. treat `NO_OP` and `ALREADY_COMMITTED` as Core-owned successful terminal outcomes which skip new-candidate publication but do **not** skip independent trust verification: before either result becomes durable client-visible success or releases the lane, a fresh reduced-privilege verifier must prove the exact result from the immutable sealed request and canonical evidence under ADR-021; `NO_OP` replays/verifies exact request + exact H0, while `ALREADY_COMMITTED` independently proves reachable accepted canonical idempotency metadata/fingerprint before ordinary stale classification;
17. distinguish request-local mutation failure from canonical repository/trust/compatibility/ref-binding failure; unhealthy accepted canonical base suspends/reconciles lane;
18. export exact candidate C plus repository/canonical-ref/operation/base/runtime/contract bindings as private content-addressed, digest-verified, no-overwrite evidence through the finalizer's restricted evidence capability;
19. independently audit exact C in a separate fresh reduced-privilege Container/DO context: use Core-owned mutation semantics against the exact immutable sealed request + exact H0 to prove the candidate's authoritative memory bytes and Core mutation metadata are the expected result, then also require exact parent/binding/scope, hard validation, strict Index v2 freshness, absence of unauthorized unrelated/control-plane changes, only minimum exact-base read needed, and only the audit-specific evidence capability; a finalizer-provided expected diff/manifest is never conformance authority;
20. make audit completion generation-bound/idempotent through immutable audit evidence/receipt which also binds the independent ADR-020 conformance result; deterministic audit failure becomes durable lane suspension/reconciliation before release; terminal-success verification under ADR-021 likewise produces immutable role-separated evidence which the finalizer cannot author and which is bound to the same exact request/repository/generation/release state;
21. only repository DO may atomically transition `AUDITED -> PUBLISHING` after rechecking cancellation, generation, lane, evidence, repository/App/canonical-ref authorization, directly observed private visibility, direct canonical-ref state, and barrier;
22. before external publisher/token I/O, persist an exact publisher-attempt identity and make publisher startup/execution addressable by that identity; the publisher performs at most one authoritative Git push for that attempt and has no autonomous retry loop;
23. after durable `PUBLISHING` authorization, have the private App gateway mint a short-lived one-repository minimum Contents-write installation token for a minimal trusted Git publisher executor/Container; that executor imports exact audited candidate Git objects, performs no semantic work and no source clone, and attempts only exact bound-ref `H0 -> C` publication;
24. require a real expected-old Git ref update. The documented REST ref-update fast-forward operation alone is not exact CAS and is not the v1 correctness mechanism; the currently documented GraphQL `updateRefs` `beforeOid`/`afterOid` path is an eligible clone-free prototype, but it may replace the publisher executor only after integration tests also prove exact candidate-object identity and required GitHub App permission behavior for the supported baseline;
25. treat `PUBLISHING` as capability-bearing/in-doubt until every issued publisher executor/token for that logical publication is provably quiesced: clean completion/termination plus revocation policy, or on ambiguous loss executor stop/destroy plus conservative wait for any unconfirmed token expiry before the lane is released or another publisher attempt is issued;
26. classify a fenced publication outcome conservatively as proven-not-published, proven-published, or indeterminate under ADR-018; timeout/lost response/process loss or merely observing `ref != C` is never proof that C was not published, and every proven/possible C remains a protected history anchor with its required evidence pinned until resolved;
27. resolve indeterminate publication only under ADR-018: current `C` or a descendant containing `C` proves the operation committed at exact C; current `H0` may retry only the same exact `C`; any other current ref that excludes C remains reconciliation-required even when it descends from the older H0. A definitive success remains a durable committed fact even if a later owner rewrite excludes C;
28. after success and publisher-capability fencing confirm/reconcile the bound canonical ref cheaply rather than another full clone/validation cycle; a later descendant containing C may be separately adopted after normal trust/index/history checks, while a rewrite excluding proven C must preserve/restore that committed history before lane reopen;
29. use signed GitHub push webhooks only for fast observation; every relevant webhook triggers direct bound-ref read and payload never directly mutates accepted state or clears a protected publication anchor;
30. distinguish proven-uncommitted stale work (`NEEDS_REPREPARE`) from unexpected bound-ref movement (`RECONCILIATION_REQUIRED`);
31. allow ordinary out-of-band reconciliation to adopt a new canonical revision only when no unresolved ADR-018 protected publication anchor blocks adoption, the last accepted revision remains reachable as an ancestor, and the exact new revision passes trust/repository/index plus mutation/idempotency-history integrity checks; backward/sideways non-descendant rewrites or revisions excluding a proven/possible C remain reconciliation-required until ancestry-preserving recovery rather than being adopted from tree validity alone;
32. keep GitHub Free and paid private repositories on same hosted architecture, paid branch/ruleset protection optional defense-in-depth;
33. preserve singleton operations, exact committed retry, cancellation-before-publication, audit-failure suspension, explicit recovery, and control-plane/canonical-ref-binding barriers;
34. version hosted delivery protocol/release and treat breaking Worker/DO/Container/evidence/publisher/reconciliation/privacy/control-path changes as barriers; incompatible rollout uses drain/maintenance or versioned blue/green isolation and an operation never crosses incompatible generations;
35. enforce explicit hosted request/repository/artifact/runtime/retry/operation-history/private-data/log/retention limits so quota/provider failures leave canonical Git unchanged;
36. make evidence retention reference-aware: queued/active/retrying/audited/publishing/reconciling operations keep every object still required for progress or exact recovery, including ADR-018 proven/possible publication anchors; only unreferenced/orphan or safely terminal evidence after its bounded recovery/incident window is GC-eligible, and automatic provider TTL must never undercut that liveness rule;
37. after contract-v9 migration, keep project current-state/overview prose outside atomic memory dual-write transaction; refresh remains separate projection concern under ADR-015;
38. transition push-on-every-normal-memory full GitHub Actions validation through the released starter/upgrader/template/private-repository managed migration, retaining Actions only for distinct health/PR/recovery/migration/control-plane checks and never hand-editing/deleting the managed workflow as part of an ordinary hosted memory operation;
39. roll mechanism through release/downstream gates and real private memory repo, then measure cold/warm acquisition, bytes, idempotency/stale preflight, finalization, packaging, request-conformance audit, terminal-success verification, publication, publisher/API-path startup/fencing, alarm/interleaving/provider startup, and end-to-end latency/cost.

ADR-019 adds mandatory rollback/recreation handling around those priorities. The existing private evidence-storage boundary carries a minimized append-only repository safety journal; it is not a second queue or transition authority. Binding epochs, client-visible acceptance, cancellation wins, accepted canonical anchors, publication intents/outcomes, and required receipt references survive outside the DO PITR rollback domain. `PUBLISHING` starts no externally effective write before its publication intent is journaled. A restored/recreated DO compares its local journal checkpoint/epoch with rollback-independent evidence, enters maintenance on mismatch, fences possibly issued publication capabilities, reconstructs accepted/cancelled/protected-history state, reconciles exact Git, and repairs alarms only after safety state is re-established. Missing/corrupt required recovery evidence fails closed rather than silently creating a fresh empty runtime for an existing binding.

ADR-020 adds mandatory independent request-to-candidate conformance before publication. The auditor must derive the expected mutation from exact H0 + the immutable sealed request through the same pinned Core semantics (or a Core-owned equivalent verifier), prove candidate `C` contains exactly those authoritative semantic changes plus a strictly fresh derived index and no unauthorized extra changes, and bind that conformance result into the audit receipt. The auditor never substitutes a replay-created commit for exact `C` and provider code does not implement a second mutation engine.

ADR-021 extends that same independent-success boundary to no-candidate terminal success. A finalizer receipt cannot by itself make `NO_OP` or `ALREADY_COMMITTED` authoritative: fresh Core/repository verification must independently prove the result, immutable role-separated terminal-success evidence must bind the exact request/canonical state/release, and the minimal receipt reference/result identity is rollback-recoverable under ADR-019 before client-visible terminal success releases the lane. Unsuccessful/stale finalizer results remain fail-closed but are not independently replayed solely to defend against denial of service.

### Repository privacy boundary

Hosted Phase 2.6 v1 supports normal writes only while the repository is directly observed as private. A GitHub `public`/repository webhook is only an early warning; current repository metadata owns the observed eligibility decision. If the repository is observed as public/internal/non-private, new normal writes and publication fail closed and require explicit revalidation before any later resume.

This does not create an impossible cross-resource guarantee. Repository visibility and the Git ref cannot be atomically CASed together with the Contents-only publisher. An authorized repository owner/admin can make the entire repository public concurrently after the last visibility check and before an already-authorized push completes. That administrator action is itself the authority exposing the repository. Phase 2.6 must document this owner-controlled privacy boundary and may narrow the race with immediate prepublication metadata checks, but it MUST NOT claim that Runethread can lock visibility without broader GitHub Administration authority.

### Pre-implementation architecture-freeze gate

Phase 2.6 implementation code MUST NOT begin merely because ADRs are drafted or CI is green. The exact current ADR/planning state must complete a fresh adversarial architecture review covering correctness, contract compatibility, state ownership, concurrency/interleaving, crash/retry behavior, destructive control-plane rollback/recreation, privilege boundaries, evidence authority/retention, independent request-to-candidate conformance, independent terminal-success verification, publisher capability lifetime, exact remote publication, accepted-history reconciliation, indeterminate publication history, repository visibility/privacy, deployment/version skew, resource limits, canonical-ref behavior, managed bootstrap/support migration, workflow supply-chain immutability, and avoidable latency/duplication.

The review passes only when it produces **zero required architecture or planning edits**. Any material correction, simplification, missing invariant, contract prerequisite, or changed implementation boundary is recorded first and resets the gate; the full review then repeats against the new exact head.

The attack review completed on 2026-09-05 against pre-amendment head `68549677e0fbb76b0018ce3aaa574c1d1ba4e1bb` found material edits and produced ADR-016 plus synchronized planning changes. That review failed the zero-edit gate.

The next full attack review, explicitly started against synchronized head `0a1ea0b871105d6497754fbbee93a387cb2494b4`, also found material required edits and produced ADR-017. That review also failed the zero-edit gate.

The following full attack review, explicitly started against synchronized head `a9e6db2f72c8d450753c5e70e4eea5eea2d78565`, found the indeterminate-publication/history-erasure race recorded in ADR-018. That review also failed the zero-edit gate.

The next full attack review, explicitly started against synchronized head `0f7f95c8220d16121144de5d1c1a4f42978550bd`, found destructive Durable Object rollback/recreation, managed private-workflow Action pinning, and v9 managed-support authority-alignment gaps and produced ADR-019. That review also failed the zero-edit gate.

The following full attack review, explicitly started against synchronized head `4dbdef5c08142856ba1795544795cea254193398`, found that the fresh auditor could prove candidate validity/binding/scope without proving the in-scope semantic bytes were actually derived from the exact sealed request. ADR-020 added independent Core-owned request-to-candidate conformance. That review also failed the zero-edit gate.

The current full attack review, explicitly started against synchronized head `34421411f4501f762c9c104e45d2cc92a9c3c5cb`, confirmed ADR-020 closes the candidate-content hole but found the same finalizer trust boundary could still be bypassed by falsely returning successful terminal `NO_OP` or `ALREADY_COMMITTED` results which never entered candidate audit. ADR-021 requires fresh independent Core/repository proof for those terminal-success claims. **This review therefore also fails the zero-edit gate and cannot unlock implementation.** No additional attack run is started in the same prompt after these edits. A later implementation-unlocking review must start from the new exact synchronized planning head and itself require zero edits.

Prototype questions may remain only when accepted ADRs already provide a safe invariant-preserving fallback and architecture does not depend on guessing the result. Current GitHub GraphQL documentation exposes an expected-old `beforeOid` ref update, so the delegated clone-free publication prototype should test exact candidate-object identity and App-permission behavior before building unnecessary publisher machinery; the accepted Git-protocol publisher remains the safe fallback until that proof exists.

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
- ADR-017 amends Phase 2.6 reconciliation/privacy/publication-capability/managed-bootstrap details without introducing a second semantic owner.
- ADR-018 amends Phase 2.6 publication/reconciliation recovery so proven or possibly published candidate history cannot be forgotten by a later out-of-band ref rewrite.
- ADR-019 amends Phase 2.6 recovery/bootstrap details so destructive DO rollback cannot erase externally relevant safety facts, retained private-memory Actions use immutable pins, and v9 support prose/migration agrees with the non-authoritative project-view model.
- ADR-020 amends the independent audit so request metadata/scope cannot stand in for proof that exact candidate semantic bytes were generated from the exact sealed request.
- ADR-021 amends the terminal-result trust boundary so finalizer-only `NO_OP`/`ALREADY_COMMITTED` claims cannot bypass independent success verification.