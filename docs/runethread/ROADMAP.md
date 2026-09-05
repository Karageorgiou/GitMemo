# Runethread implementation roadmap

Status: **Proposed**

This roadmap sequences the work so identity is made coherent while the project is still pre-v1, then each engineering stage proves one architectural assumption before adding another dependency. `CURRENT_MILESTONE.md` remains the authority for the immediate work boundary.

## Phase 0 — architecture freeze

Goal: define boundaries before implementation changes.

Deliverables:

- system architecture;
- clean GitMemo -> Runethread cutover strategy;
- ADR catalog for canonical ownership, deterministic Core, two-phase mutation, concurrency, orchestration, worker abstraction, worktree isolation, permissions, migration, and repository topology;
- explicit non-goals;
- security/trust boundary carried forward from GitMemo.

Exit criteria:

- architecture is reviewable as documentation-only changes;
- no code/module/repository-format rename has occurred yet;
- unresolved architecture questions are recorded rather than hidden in implementation;
- ADRs define the target Runethread-native identity and migration invariants.

## Phase 1 — controlled Runethread identity cutover

Goal: remove avoidable GitMemo naming before substantial new APIs and integrations are built.

Target identity:

```text
implementation       runethread/core
Go module            github.com/runethread/core
CLI                  runethread
native metadata      .runethread/
private repo naming  <user>/runethread-memory
first RT release     v0.6.0
```

Primary work:

- preserve Git history while transferring/renaming the implementation repository;
- update Go module/import identity;
- rename CLI entry point and user-facing commands;
- convert setup/bootstrap/release/template identity to Runethread;
- define the native `.runethread/` config/lock format;
- implement/test deterministic migration from the known GitMemo v0.5.0 source state;
- migrate the known private memory repository without replacing canonical UUID identities;
- change project slug/metadata from `gitmemo` to `runethread` where semantically appropriate;
- review the small project-memory set for product-name references rather than global search/replace;
- regenerate derived indexes;
- preserve historical GitMemo tags/releases/commits as immutable history;
- retain an explicit pre-migration recovery point.

Exit criteria:

- current implementation identity is consistently Runethread;
- `go test ./...` and all release gates pass under the new module identity;
- newly initialized repositories are Runethread-native;
- migrated private memory repository passes full validation;
- migrated memory count and UUID set match the pre-migration snapshot exactly;
- no unintended GitMemo operational identifiers remain in current native state;
- historical GitMemo releases remain reachable and unmodified.

## Phase 2 — deterministic Core mutation service

Goal: make the memory engine usable through a small deterministic application boundary under the final Runethread identity.

Primary work:

```text
MemoryService
  Search
  Get
  PrepareMutation
  ApplyMutation
  Withdraw
  Status
```

Supporting work:

- repository abstraction;
- expected Git revision in prepared mutations;
- local write serialization where needed;
- mutation transaction staging/rollback;
- validator integration as a hard write gate;
- relationship/lifecycle invariant enforcement;
- machine-readable result/error contracts;
- JSON CLI output suitable for automation.

Suggested CLI:

```text
runethread get --json
runethread prepare --json
runethread apply --json
runethread withdraw --json
runethread status --json
```

Exit criteria:

- an AI/tool no longer needs to know how canonical Markdown/JSON/index files are physically edited;
- stale-revision writes are rejected;
- a failed hard validation cannot produce a successful commit/result;
- tests verify rollback and concurrency failure modes.

## Phase 2.5 — compatibility hardening and release separation

Goal: harden repository/runtime compatibility before adding new transport or delivery surfaces.

Primary work:

- explicit runtime-release versus contract-release identity;
- contract v8 compatibility matrix and migration rules;
- filesystem/repository-boundary safety hardening;
- deterministic upgrade behavior across supported source states;
- public template and known private-memory migration to the v0.8 contract state;
- engineering-process and cross-platform pipeline hardening.

Exit criteria:

- Runethread v0.8.0 is published as an immutable release;
- contract v8 semantics and release separation are explicit and tested;
- the public template and known private memory repository are migrated and permanently validated;
- historical compatibility and migration evidence are retained.

Status: **Complete**.

## Phase 2.6 — Memory Write Delivery Pipeline

Goal: make normal hosted external memory delivery fast, deterministic, race-safe, auditable, and ready for later MCP transport without turning GitHub Actions or another redundant orchestration layer into a temporary primary execution architecture.

Governing decisions:

- ADR-012 — audited candidate promotion for external memory delivery;
- ADR-013 — per-repository serialized mutation-delivery queue;
- ADR-014 — Cloud-hosted Phase 2.6 memory-delivery control plane;
- ADR-015 — project current-state documents become asynchronous orientation projections only in the next operational contract;
- ADR-016 — hosted contract eligibility, evidence capability/retention, and exact-publication hardening;
- ADR-017 — accepted-history reconciliation, private-repository eligibility, publisher-capability fencing, and managed validation-bootstrap rollout;
- ADR-018 — proven/possible publication-history preservation across ambiguous completion and out-of-band rewrites;
- ADR-019 — rollback-independent recovery evidence, destructive-restore barriers, immutable managed-workflow Action pins, and v9 managed-support authority alignment;
- ADR-020 — independent candidate-to-request semantic conformance before publication;
- ADR-021 — independent verification of successful terminal no-candidate mutation results;
- ADR-022 — deterministic, fully audited candidate Git commit envelope and verified exact-C object closure;
- ADR-023 — rollback-durable terminal operation dispositions before client-visible completion/lane release.

### Contract and managed-support prerequisite

Contract v8 remains immutable and requires relevant project current-state synchronization as part of memory-write completion. Current `MemoryService.ApplyMutation` does not implement that project-view semantic write. Therefore, after the architecture-freeze gate passes, the first implementation slice is the ADR-015 contract transition (target contract v9), exact v8 fixture/migration/release verification, template migration, and private-repository migration.

That same released migration sequence deliberately updates managed support/bootstrap state under ADR-017/ADR-019:

- normal hosted canonical pushes no longer trigger redundant full validation;
- every retained external `uses:` action in `.github/workflows/validate.yml` is pinned to a verified full-length commit SHA;
- the exact previous managed workflow remains a supported migration source, while customized/unrecognized workflow bytes are not silently overwritten;
- generated/current Runethread support prose no longer describes project current-state/overview files as canonical sources under v9;
- existing user project-view bytes remain preserved;
- a private-memory README is automatically replaced only when its prior bytes/source state are exactly recognized as Runethread-managed, not merely because it has a Runethread heading and lock reference.

Normal hosted Phase 2.6 mutation is admitted only after the repository is explicitly migrated to the projection-capable contract implementing ADR-015, or a later contract explicitly supported by the hosted release, **and** the repository has the supported managed support/bootstrap state. Contract-v8 repositories may be read, verified, reconciled, and upgraded, but the normal hosted mutation path does not silently bypass the v8 synchronization rule.

Target topology:

```text
runethread/core
  deterministic MemoryService/CLI/runtime

runethread/hosted
  public API + webhook edge
  per-repository runtime Durable Objects + SQLite + alarms
  attached finalizer Containers
  private role-scoped evidence boundary/storage
    + append-only rollback-independent safety journal
  fresh auditor/verifier Container/DO contexts
  private GitHub App gateway
  exact-CAS publication path, with minimal Git publisher fallback
```

`runethread/hosted` is the target provider-specific component unless implementation preflight finds a concrete topology/naming conflict. Hosted provider churn must not become a dependency of Core. Hosted releases pin immutable verified Core/runtime/container/protocol identities rather than floating public `main`.

Architecture:

```text
semantic caller
      |
authenticated sealed mutation request
      |
public Worker/API edge
      |
repository-runtime Durable Object
  SQLite lane/queue/phase-generation + alarm-driven drive()
      |
attached finalizer Container
  Core-owned deterministic candidate commit envelope
      |
role-scoped immutable finalization/candidate evidence
      |
fresh reduced-privilege auditor/verifier
  candidate: Core-owned H0 + sealed-request -> exact C
             semantic + raw commit-envelope proof
  terminal success: Core/repository NO_OP or ALREADY_COMMITTED proof
      |
role-scoped immutable audit/terminal-success evidence
      |
repository DO authorizes terminal success or PUBLISHING
      |
rollback-independent publication intent recorded before write-capable I/O
      |
private GitHub App gateway
      |
short-lived one-repo Contents-write authority
      |
verified object closure only
      |
exact expected-old H0 -> exact audited C
(API path if exact-object identity is proven;
 minimal Git publisher is safe fallback)
      |
user-owned private GitHub canonical branch ref
```

Primary work:

- authenticated request/status/cancel API plus explicit caller-to-App-installation repository authorization;
- explicit repository binding to immutable repository identity + App installation + canonical branch ref + last accepted revision, normally discovering the ref from default branch at adoption but never silently following later ref/default changes; directly observed private visibility is a live hosted-write eligibility condition;
- private sealed request bodies stored as content-addressed/no-overwrite objects, with only opaque references/digests in DO state/logs/status;
- role-scoped evidence creation so admission/finalizer/auditor-verifier can create only their exact artifact class/key for the current repository/attempt/phase/generation under digest/size/create-if-absent constraints; finalizer cannot manufacture authoritative candidate-audit or terminal-success-verification evidence and auditor/verifier cannot replace finalization evidence;
- one repository DO remains the sole **live** hosted lane/operation-state/publication-authorization authority, with transactional SQLite for bounded queue, one active op, exact phase/execution generation, retry/backoff/deadline state, evidence refs, canonical-ref binding, private-visibility eligibility, publisher-attempt/fencing state, protected publication anchors, safety-journal checkpoint/binding epoch, suspension/maintenance/reconciliation;
- the existing private evidence-storage boundary also carries a minimized append-only rollback-independent safety journal rather than a second queue/state machine. Binding/enrollment, client-visible acceptance, cancellation wins, accepted canonical anchors, publication intents/outcomes, required immutable receipt references, independently verified terminal-success receipt references/result identities, and ADR-023 terminal-disposition records are create-if-absent/digest-bound and contain no plaintext memory bodies or credentials;
- Durable Object PITR/recreation/detected state rollback is an exclusive maintenance/recovery barrier, not an ordinary retry. A restored/recreated DO must reconcile its journal checkpoint/epoch, immutable evidence, current GitHub binding/privacy, accepted operations/cancellations, terminal dispositions, protected publication anchors, terminal-success evidence, and possibly-issued publication capabilities before normal admission/publication resumes;
- missing/corrupt/forked rollback-independent safety evidence fails closed rather than silently creating a fresh empty runtime for a repository known to have an existing binding epoch;
- no Cloudflare Workflow in v1; one DO alarm is the at-least-once scheduler/recovery wakeup and the state driver explicitly persists/reschedules prolonged retryable failure;
- explicit async-interleaving protocol: local atomic phase+generation claim before every Container/object-store/GitHub external action, external I/O outside `blockConcurrencyWhile()`, and compare operation/phase/generation before accepting result; stale-generation output cannot advance state;
- durable acceptance handoff: return `ACCEPTED` only after request/operation metadata, a recoverable alarm, and rollback-independent acceptance evidence are established; exact resubmission/status/recovery repairs missing alarms for ordinary stored work while destructive rollback enters the recovery barrier;
- rollback-sensitive cancellation and accepted-anchor adoption are write-ahead recoverable: a local winning transition is not exposed/released as durably final until the corresponding immutable safety record exists;
- every accepted operation crosses ADR-023's write-ahead terminalization barrier before any durable terminal status is returned or its serialized lane position is released; the terminal-disposition record binds exact result class, operation/request identity, generation/release, authoritative receipt/evidence and canonical basis, and recovery never requeues a journaled terminal operation;
- referenced request/candidate/finalization/audit/terminal-verification/publication evidence remains live while needed by queued/active/retrying/audited/publishing/reconciling work, including ADR-018 protected candidates; private content may expire after its safe terminal window, while minimized safety-journal/terminal-disposition history required by an active binding epoch remains until explicit safe binding deletion/rebaseline policy permits removal;
- separate hosted attempt identity from Core idempotency identity; hosted identity binds repo/canonical-ref/request digest while Core owns semantic committed retry/conflict;
- Container/DO relationship used so repository runtime manages attached finalizer lifecycle rather than another coordinator;
- private internal GitHub gateway retains the long-lived App private key; public API, finalizer, auditor/verifier, evidence storage, and publisher executor never receive that key; ordinary runtime App baseline is Contents/Metadata without Administration/Workflows;
- full hosted operation serialized per repo while preserving ADR-003 committed-idempotency-before-stale ordering; stale work may require cold source preflight but stops before candidate/Index/package/audit once proven uncommitted;
- finalizer runs exact pinned Runethread Go/Core + Git; cold source target at most one clone/fetch with historical idempotency reachable, optional warm cache only after direct canonical-ref reset;
- hardened Git policy disabling/refusing hooks, recursive submodules, filters, unsafe config/includes, credential helpers, and similar repository-controlled execution;
- every fresh finalization starts from directly observed bound canonical ref/revision, never surviving unpromoted local candidate history;
- immutable generation-bound finalization receipt: candidate evidence first, receipt last, create-if-absent/no-overwrite; a valid receipt selects the finalizer's claimed result but does not by itself make a success-like terminal result authoritative;
- finalization via existing `MemoryService.ApplyMutation`, preserving committed-retry-before-stale, one Index v2 write, hard validation, commit creation, local-only fast-forward;
- candidate-producing success requires ADR-020 fresh request-to-candidate conformance plus all existing exact parent/trust/index/scope checks before publication;
- under ADR-022, candidate commit construction is a deterministic Core-owned envelope: release-defined author/committer identity, author/committer time deterministically derived from exact request `mutation_time`, sanitized Git identity/date/config inputs, an explicit allowed commit-header set, and exact reproducibility of candidate object ID for the same H0 + request + pinned release;
- claimed `NO_OP` requires fresh reduced-privilege Core verification of exact immutable sealed request + exact H0 before durable client-visible terminal success/lane release; a finalizer cannot bypass candidate audit by falsely declaring no-op;
- claimed `ALREADY_COMMITTED` requires fresh reduced-privilege Core/repository committed-idempotency lookup against directly observed accepted canonical history for the exact request, preserving committed-idempotency-before-stale ordering and proving exact request fingerprint/mutation metadata before durable client-visible terminal success/lane release;
- immutable role-separated terminal-success receipt binds exact repository/ref/request/attempt/generation/release plus independently proven result/canonical evidence; finalizer cannot author it, and its minimal receipt reference/result identity is established in rollback-independent recovery evidence before terminal success releases the lane;
- unsuccessful/stale finalizer results remain generation-bound/fail-closed and do not require equivalent independent semantic replay solely to defend against denial of service by a compromised finalizer, but any one accepted as a durable terminal outcome still requires ADR-023 terminal-disposition evidence before response/lane release;
- canonical repository/trust/compatibility/ref-binding failure separated from request-local mutation failure;
- candidate evidence bound to repo/canonical-ref/H0/C/tree/parent/Core-idempotency/request fingerprint/hosted attempt+generation/runtime/container/delivery/contract/candidate-envelope identities/digests;
- candidate transport selected by measured exact bytes/runtime; prototype Git-native H0->C delta against self-contained package and prove completeness under partial-clone/promisor modes, while unrelated unreachable finalizer-supplied Git objects are outside the authorized exact-C closure and cannot be imported merely because they were packaged;
- fresh reduced-privilege audit of exact C with bounded exact-base acquisition: independently derive the expected semantic mutation from exact immutable sealed request + exact H0 through the same pinned Core mutation semantics (or a Core-owned equivalent verifier), prove C contains exactly those authoritative memory/relationship/lifecycle changes, then parse/prove ADR-022's complete raw commit envelope and exact expected candidate object ID, and separately require hard validation, strict Index v2 freshness, exact commit/tree/parent/binding/scope, and no unauthorized unrelated/control-plane changes; finalizer-provided expected diff/manifests/commit metadata are not conformance authority;
- auditor scratch writes used for Core-owned conformance replay are disposable/local only; the auditor never repairs C, replaces candidate evidence, or obtains publication authority;
- immutable generation-bound candidate audit receipt binds exact request digest/fingerprint, H0, C/tree, independently derived conformance result, and pinned runtime/delivery/contract identities, with deterministic audit failure durably suspending/reconciling lane before release; if that audit failure is exposed as terminal for the operation, ADR-023 closes it in rollback-independent evidence first;
- DO-only `AUDITED -> PUBLISHING` authorization after current generation, cancellation, lane, evidence, App/repo/canonical-ref auth, directly observed private visibility, direct bound-ref==H0, and barrier checks;
- cancellation-vs-publication linearized by local atomic state transition, with rollback-independent cancellation/terminal-disposition evidence required before a cancellation is durably released;
- before **any** externally effective publication/token/executor/API I/O, durably create an immutable rollback-independent publication-intent record bound to repository/ref/H0/C/operation/generation/evidence/protocol/publication attempt and a conservative fencing horizon; a later restored DO treats that exact C as possibly published until definitive evidence proves otherwise;
- require a true expected-old ref update for exact H0 -> exact audited C. GitHub REST `Update a reference` with `force=false` is not sufficient; current GraphQL `updateRefs` documents an expected-old `beforeOid`/`afterOid` update and is therefore an eligible clone-free implementation prototype, but it replaces the minimal Git publisher only if integration tests also prove exact candidate-object identity and required GitHub App permission behavior;
- until that API proof exists, only after durable `PUBLISHING` plus publication-intent safety evidence does the private App gateway mint a short-lived one-repository minimum Contents-write installation token to the accepted minimal trusted publisher fallback; the publisher imports only the independently verified object closure required by exact audited C relative to H0, ignores/rejects unrelated unreachable package objects, performs no source clone/semantic mutation/repair/audit, never constructs C2, performs at most one expected-old Git-protocol update per publisher-attempt identity, and has no autonomous retry loop;
- `PUBLISHING` remains capability-bearing/in-doubt until every issued publication capability for the logical publication is fenced: clean completion must satisfy the path's termination/token-disposal policy, while ambiguous executor/token loss stops/destroys the executor and conservatively waits for unconfirmed token expiry before lane release or another publication attempt;
- after fencing, classify the publication result conservatively as **proven not published**, **proven published**, or **indeterminate**; timeout, lost response, process loss, or merely reading `ref != C` is not proof that C was not published;
- every proven or possibly published exact C is a protected history anchor with required evidence pinned. A definitive success is a durable committed fact even if an authorized owner later rewrites the current branch so C is no longer reachable;
- a candidate result cannot be exposed as terminal `COMMITTED` while publication is merely possible/indeterminate: positive exact-C publication proof and the corresponding ADR-023 `COMMITTED(C)` terminal-disposition record must exist before durable client-visible success or lane release;
- indeterminate publication resolves under ADR-018: current ref C or any current descendant containing C proves the operation committed at exact C; current exact H0 may retry only the same exact C; any other current revision excluding C remains `RECONCILIATION_REQUIRED` even if it descends from the older H0, because Runethread cannot distinguish “C never published, then N advanced” from “C published, then was rewritten away”;
- a rewrite excluding proven C must preserve or restore C in canonical ancestry before the normal hosted lane reopens; Phase 2.6 does not silently destructive-rebaseline previously committed/idempotency history;
- cheap post-publication ref/ancestry confirmation and reconciliation, no redundant full validation;
- signed push and repository-visibility webhooks only as hints; always reread authoritative current GitHub state, never mutate accepted state from payload alone, and never clear a protected publication anchor or safety-journal fact from a webhook/current-ref observation;
- proven uncommitted stale work -> `NEEDS_REPREPARE`; unexpected active-operation ref movement -> reconciliation; if `NEEDS_REPREPARE` is returned as terminal, ADR-023 prevents a later rollback from silently rerunning it even if the ref later returns to the old H0;
- ordinary out-of-band adoption runs only when no ADR-018 protected publication anchor blocks it, requires the last accepted canonical revision to remain an ancestor of the proposed new accepted revision, and requires exact trust/repository/index/mutation-idempotency-history checks; backward/sideways non-descendant rewrites and sibling descendants excluding a proven/possible C remain reconciliation-required until ancestry-preserving recovery because Core committed-idempotency evidence is reachable Git history;
- observed non-private visibility suspends/fails closed for normal hosted writes and requires explicit revalidation before later resume; threat-model documentation states that an authorized GitHub owner/admin can change visibility outside the Git-ref CAS and Runethread does not claim atomic visibility+ref locking without Administration authority;
- one hosted implementation for Free/paid private GitHub, with paid branch/ruleset protection optional defense-in-depth;
- explicit release/protocol versioning and non-atomic Worker/DO/finalizer/auditor-verifier/evidence/publication/reconciliation/privacy/managed-bootstrap/canonical-ref/safety-journal/recovery/audit-conformance/terminal-success-verification/candidate-envelope/terminal-disposition deployment handling using maintenance/drain or versioned blue/green;
- operation outcomes `NO_OP`, `ALREADY_COMMITTED`, `NEEDS_REPREPARE`, finalization/audit/cancel/reconciliation plus lane OPEN/SUSPENDED/MAINTENANCE, with `NO_OP`/`ALREADY_COMMITTED` not authoritative until ADR-021 verification succeeds and no terminal outcome lane-releasing until ADR-023 terminalization succeeds;
- control-plane barriers for contract/schema/trust/repository-format/bootstrap/delivery/migration/evidence/publication/reconciliation/privacy/canonical-ref binding/recovery/audit-conformance/terminal-success-verification/candidate-envelope/terminal-disposition changes;
- request/rate/repository/artifact/Container/retry/operation-history/log/private-data retention limits and threat model, distinguishing short-lived private content from minimized recovery metadata;
- project current-state/overview prose treated as materialized orientation views only after the ADR-015 contract migration, not as an implicit contract-v8 relaxation;
- remove push-on-every-normal-memory full Actions validation only through the released starter/upgrader/template/private-repository migration; every retained external Action is pinned to a full immutable commit SHA and customized workflow/README content is not silently overwritten;
- measure cold/warm source, bytes, idempotency/stale preflight, finalization, deterministic candidate-object construction, package, request-conformance audit, terminal-success verification, terminal-disposition persistence, publication, publisher/API-path startup/fencing, journal recovery, alarm/retry/interleaving, provider startup, total latency/cost.

### Pre-implementation architecture-freeze gate

Before implementation begins, exact current ADR/planning state must complete a fresh adversarial review covering correctness, contract compatibility, state ownership/component necessity, async interleaving, concurrency, crash/retry/ambiguous response, destructive Durable Object rollback/recreation, rollback-durable terminal dispositions, privilege/evidence-authority boundaries, evidence/safety-journal liveness and retention, independent request-to-candidate conformance, deterministic/full candidate commit-object envelope and exact object closure, independent terminal-success verification, publisher/publication-capability lifetime, exact publication, proven/possible publication-history preservation, accepted-history reconciliation, repository visibility/privacy, canonical-ref lifecycle, managed-bootstrap/support rollout, workflow supply-chain immutability, deployment/version skew, resource limits, and avoidable duplication/latency.

The review passes only with **zero required architecture/planning edits**. Any material correction/simplification/missing invariant/changed boundary is recorded first and resets the gate; full review repeats against the new exact head. Prototype questions may remain only with accepted safe invariant-preserving fallback.

The attack review completed on 2026-09-05 against pre-amendment head `68549677e0fbb76b0018ce3aaa574c1d1ba4e1bb` found material changes and produced ADR-016 plus synchronized planning edits. It failed the zero-edit freeze gate.

The subsequent full attack review, explicitly started against synchronized head `0a1ea0b871105d6497754fbbee93a387cb2494b4`, found material changes and produced ADR-017 plus synchronized planning edits. It also failed the zero-edit freeze gate.

The following full attack review, explicitly started against synchronized head `a9e6db2f72c8d450753c5e70e4eea5eea2d78565`, found the indeterminate-publication/history-erasure race and produced ADR-018 plus synchronized planning edits. It also failed the zero-edit freeze gate.

The next full attack review, explicitly started against synchronized head `0f7f95c8220d16121144de5d1c1a4f42978550bd`, found material destructive-control-plane-recovery and managed-support/security corrections and produced ADR-019 plus synchronized planning edits. It also failed the zero-edit freeze gate.

The following full attack review, explicitly started against synchronized head `4dbdef5c08142856ba1795544795cea254193398`, found that the fresh audit could validate candidate C and its request metadata/scope without independently proving that C's in-scope semantic bytes are the exact Core-derived result of the immutable sealed request. ADR-020 added that request-to-candidate conformance gate. It also failed the zero-edit freeze gate.

The next full attack review, explicitly started against synchronized head `34421411f4501f762c9c104e45d2cc92a9c3c5cb`, confirmed ADR-020 closes the candidate-content hole but found successful terminal `NO_OP`/`ALREADY_COMMITTED` claims still bypassed fresh independent verification. ADR-021 requires those no-candidate success claims to be independently proven before durable client-visible success/lane release. It also failed the zero-edit freeze gate.

The following full attack review, explicitly started against synchronized head `9151c9d2e1a383e79449af2963fc2c547bb49429`, confirmed ADR-021 closes the candidate-success bypass but found that exact candidate `C` still allowed ambient/nondeterministic commit metadata outside ADR-020's semantic tree/message proof. ADR-022 therefore makes the entire candidate commit envelope deterministic/Core-owned, independently derives exact expected C, and limits publication to exact-C object closure. It also failed the zero-edit freeze gate.

The current full attack review, explicitly started against synchronized head `1b4ff70402f8aa298c4e219e0e9254e998f037ca`, confirmed ADR-022 closes the candidate-envelope/object-closure gap but found that accepted operations could still be resurrected by destructive DO rollback after a client-visible terminal result. ADR-019 protected acceptance/cancellation/anchors/publication intent and ADR-021 protected successful no-candidate terminal results, but ordinary stale/failure/audit terminal dispositions and candidate `COMMITTED` lacked one general write-ahead rollback-independent terminalization barrier. ADR-023 closes that gap without adding independent semantic replay for unsuccessful results. **This review therefore also fails the zero-edit freeze gate.** No further review run is started in the same prompt after these edits; implementation remains blocked until a later explicitly requested full review of the new exact synchronized head passes with zero edits.

The currently documented GraphQL expected-old ref path is a non-gating implementation prototype under the escape hatch already accepted by ADR-016: exact Git-protocol publication remains the safe fallback until exact candidate-object identity and GitHub App permission behavior are proven end-to-end.

Phase 2.6 v1 deliberately uses singleton operations. Deferred: semantic dependency quantification, batching/coalescing, automatic semantic re-preparation, general AI task orchestration, and project-orientation refresh machinery beyond the ADR-015 correctness semantics.

Exit criteria are tracked in issue #20. At minimum:

- zero-edit architecture-freeze review recorded on the exact final planning head before implementation;
- contract-v8 repository is not admitted for normal hosted mutation and supported migration to the ADR-015 contract plus supported managed support/bootstrap state is verified first;
- retained private-memory workflow uses full-length immutable Action SHAs, exact previous managed workflow remains a recognized source, and customized workflow bytes are not silently overwritten;
- generated/current support prose agrees that project current-state/overview files are non-authoritative orientation views under v9, project-view user bytes are preserved, and customized native README content is not overwritten by a broad heuristic;
- remote authenticated caller submits sealed mutation without canonical file editing/GitHub write credential;
- repository/App/canonical-ref auth cannot be gained by guessed identifiers and directly observed non-private visibility blocks normal hosted writes;
- Free/protected paid private repos use same hosted mutation path;
- hosted provider code outside Core, exact pinned runtime/container/protocol identities;
- one repository DO is sole live lane/state/publication-authorization authority, no second Workflow state machine;
- rollback-independent safety journal exists only as recovery evidence and cannot authorize ordinary transitions;
- DO state + alarms recover accepted work without client traffic in ordinary execution;
- explicit PITR/recreation recovery reconstructs client-visible accepted operations, cancellation wins, accepted canonical anchors, all ADR-023 terminal dispositions, independently verified terminal-success outcomes, and possible/proven publication history before normal writes resume;
- rollback to before a successful cancellation cannot resurrect cancelled work;
- rollback to before publication cannot forget a journaled possible/proven `C` or start a new semantic candidate as if publication never happened;
- rollback to before any returned terminal stale/failure/audit result reconstructs that exact terminal disposition rather than re-running the operation against current repository state;
- rollback to before a returned verified `NO_OP`/`ALREADY_COMMITTED` reconstructs or safely re-verifies that exact terminal-success fact rather than silently executing a new semantic mutation;
- a returned candidate `COMMITTED(C)` is journaled only after positive publication proof and is restored as historically committed after rollback even if later canonical history excludes C;
- missing/corrupt/forked required recovery journal fails closed;
- alarm/request/cancel/webhook interleaving is safe through atomic phase/generation claims and stale result rejection;
- accepted response cannot strand work without durable alarm/recovery schedule and rollback-independent acceptance evidence;
- committed idempotency resolved before stale classification;
- cold finalizer preserves historical idempotency, fresh retry never treats unpromoted local C as committed;
- finalization evidence+receipt ordering is deterministic/no-overwrite;
- wrong-role/wrong-generation evidence cannot advance operation state and finalizer cannot create authoritative candidate-audit or terminal-success-verification receipts;
- live referenced evidence survives nominal retention boundaries, including protected proven/possible publication candidates, while private content and minimized safety metadata follow their distinct explicit retention/deletion policies;
- MemoryService builds exact C, writes Index once, hard-validates, leaves remote canonical ref H0 before audit;
- repeated candidate construction from the same exact H0 + sealed request + pinned envelope release yields the same exact candidate commit SHA;
- candidate author/committer identity and dates are release-defined/request-derived and cannot be changed by inherited `GIT_AUTHOR_*`, `GIT_COMMITTER_*`, `EMAIL`, locale/timezone, or Git config/include state;
- private request/runtime bytes cannot be persisted through author/committer metadata or unexpected commit headers; raw exact C envelope is independently audited and unknown/unapproved headers fail;
- private request/candidate/audit/terminal-verification data does not leak in ordinary metadata/log/status/safety journal;
- fresh auditor independently proves C's authoritative semantic bytes are the Core-derived result of exact sealed request + exact H0, independently derives the complete expected raw commit object and exact expected C identity, binds that conformance to the audit receipt, and separately verifies strict index/tree/parent/scope/trust without repair/publication authority;
- a valid/index-fresh/in-scope candidate with copied legitimate request fingerprint but altered requested Markdown/JSON/relationship semantics fails audit;
- a candidate with the exact authorized tree/message but altered author/committer/date/header metadata fails audit;
- a finalizer candidate package containing unrelated unreachable Git objects cannot cause those objects to be imported/uploaded by the fallback publisher or an accepted API path;
- a candidate-producing request falsely reported by finalizer as `NO_OP` cannot become terminal success; fresh exact request+H0 Core verification must independently return no-op;
- a false `ALREADY_COMMITTED` claim cannot become terminal success; fresh accepted-canonical Core/repository lookup must independently prove exact idempotency key/request fingerprint/mutation metadata before stale classification;
- terminal-success verification evidence is immutable/role-separated and rollback-recoverable before the result releases the lane;
- every durable lane-releasing result, including unsuccessful terminal results, has exactly one immutable rollback-independent ADR-023 terminal disposition before client-visible completion/release; conflicting terminal dispositions fail closed;
- only repository DO can authorize publication after exact evidence/cancel/lane/auth/ref/private-visibility checks;
- rollback-independent publication intent is established before any externally effective publication/token/executor/API I/O;
- default/canonical-branch rename/change/deletion/transfer fails closed rather than silently redirects;
- ordinary reconciliation never adopts a non-descendant rewrite that loses last accepted canonical/idempotency history and never adopts a sibling descendant which could exclude a proven/possibly published C;
- long-lived App key remains gateway-only and any publisher fallback receives only a short-lived one-repo minimum write token after PUBLISHING + publication-intent record;
- publication changes only exact bound ref H0 -> exact audited C through a true expected-old Git update; an API path replaces the fallback publisher only after exact-C object identity and permissions are proven;
- lane cannot leave publication in-doubt while an old publication capability can still act;
- a proven or possibly published C cannot be forgotten because a later current-ref read, force-rewrite, process restart, or DO rollback excludes/forgets it;
- indeterminate publication at C/descendant/H0/sibling states resolves exactly according to ADR-018 and never silently constructs/re-finalizes C2;
- previously proven committed Runethread operation metadata remains reachable in canonical history before the normal hosted lane reopens;
- forward/sideways/backward/deleted/recreated ref races cannot be overwritten as if H0 still held;
- ambiguous publication and webhook/order failures reconcile deterministically after capability fencing;
- deployment/version barriers prevent mixed incompatible finalizer/auditor-verifier/evidence/publication/reconciliation/privacy/managed-bootstrap/safety-journal/recovery/audit-conformance/terminal-success-verification/candidate-envelope/terminal-disposition semantics;
- resource/provider/privacy failures leave canonical Git unchanged;
- local/offline MemoryService remains Cloudflare-independent;
- push-on-every-memory Actions validation removed from normal hosted data plane through the released managed migration;
- real private-repo rollout and measured latency/cost prove redundant work removed without deleting distinct trust boundaries.

Hosted Runethread infrastructure account requires Workers Paid because Containers are needed; this is not an end-user Cloudflare-plan requirement.

**Phase 3 is blocked until Phase 2.6 exits green.**

## Phase 3 — MCP adapters over the established memory boundary

Goal: expose the already-proven local and hosted memory operations to compatible AI clients without duplicating business logic.

Phase 3 begins only after Phase 2.6 is implemented and independently verified. MCP is transport over established MemoryService/delivery semantics; it does not own mutation, queueing, auditing, concurrency, or Git publication.

```text
local client                 remote client
    |                            |
MCP stdio/local            MCP authenticated HTTP
    |                            |
MemoryService              Runethread hosted delivery API
    |                            |
local Git                    MemoryService/Core -> user GitHub
```

Initial memory tools:

```text
memory.search
memory.get
memory.prepare
memory.apply
memory.withdraw
memory.status
```

Start with smallest transports needed for local/offline and hosted remote use. Re-check current MCP protocol/SDK/auth when Phase 3 becomes current.

Exit criteria:

- compatible AI client retrieves/mutates without repo-layout knowledge;
- local clients work without Cloudflare when local Git/exec available;
- remote clients use Phase 2.6 lifecycle without canonical Git write credentials;
- MCP code is transport translation, not repository/queue/audit rules;
- MCP integration changes do not alter canonical memory or Phase 2.6 publication invariants.

## Phase 4 — Orchestrator skeleton

Goal: prove the execution control plane without paying for or depending on real model workers.

Create separate repository:

```text
runethread/orchestrator
```

Implement:

- project registry;
- task model/state machine;
- SQLite persistence and migrations;
- append-only task events;
- capability router;
- policy/approval model;
- CLI;
- fake worker implementation.

Suggested CLI:

```text
runethread project add
runethread task dispatch
runethread task status
runethread task result
runethread task cancel
```

Exit criteria:

- fake tasks traverse full lifecycle deterministically;
- restart/resume preserves task state;
- invalid transitions rejected;
- no AI provider required for tests.

## Phase 5 — Codex worker

Goal: prove safe delegated code execution.

Implement:

- Codex App Server adapter;
- per-task Git worktree creation;
- task branch naming;
- streamed worker events;
- approval bridge;
- cancellation;
- normalized result contract;
- build/test verification capture.

Target proof:

```text
CLI -> Orchestrator -> capability router -> isolated worktree -> Codex
    -> build/tests -> structured result -> SQLite task history
```

Exit criteria:

- Codex cannot modify normal checkout by default;
- changed files/commands/tests/branch/worktree/unresolved verification reported;
- execution failure distinct from verification failure;
- cancellation/approval pauses tested.

## Phase 6 — general agent worker

Goal: support substantial non-coding delegated work.

Implement provider adapter under provider-independent worker interface.

Capabilities may include web research, file/document analysis, long-running reasoning, structured artifacts, and project-repo read access without automatic mutation.

Exit criteria:

- router distinguishes `chat`, `codex`, `general` capability sets;
- provider thread IDs confined to adapter/runtime metadata;
- paid API usage can be disabled without breaking Core/Codex/local workflows.

## Phase 7 — project context aggregation

Goal: make chat/session replacement cheap and reliable.

Assemble execution/context packet from relevant memory, project repo metadata/instructions/ADR pointers, active/recent tasks, blockers/next actions. It is not a new canonical database.

Exit criteria:

- fresh AI conversation recovers project continuity without prior chat URL;
- project source truth referenced rather than duplicated;
- stale runtime metadata distinguished from canonical project Git state.

## Phase 8 — broader remote project/task integration

Goal: extend established remote memory/MCP surface to project/task operations where product/platform support permits it.

Expose authenticated later-Orchestrator capabilities such as `project.*` and `task.*`. Memory remote transport is already Phase 2.6 + Phase 3.

Exit criteria:

- remote project/task access explicit auth/least privilege;
- writes retain Core/Orchestrator invariants;
- changing AI-client APIs does not require canonical-storage or Phase 2.6 architecture changes.

## Phase 9 — hardening and public readiness

Goal: make Runethread-native system safe for external users before compatibility becomes expensive.

Work includes release/migration fixtures, fuzz/property coverage, cross-platform CI/release, threat model, install/doctor UX, signing/SBOM/dependency scanning where practical, public docs, org/repo protection policy.

Exit criteria:

- new users install without GitMemo knowledge;
- supported release compatibility policy stated;
- recovery/migration tested before external adoption raises breaking-change cost.

## Later, only when justified

Possible future work:

- notifications/webhooks beyond canonical-movement webhook;
- collaboration/team policies;
- richer UI;
- multiple parallel workers;
- additional provider adapters;
- remote synchronization conveniences;
- alternative hosted execution providers;
- PostgreSQL/additional queue infrastructure only when measured scale requires it;
- container orchestration/Kubernetes only when measured requirements justify it.

## Immediate next engineering milestone

The immediate milestone is **Phase 2.6 — Memory Write Delivery Pipeline**.

> Finish the pre-implementation architecture freeze, then implement the ADR-015 contract + managed-support migration followed by the Cloudflare-hosted, audited, per-repository mutation-delivery path defined by ADR-012 through ADR-023. Normal hosted writes begin only on an explicitly compatible migrated private repository. Reconciliation preserves accepted Git/idempotency history and every proven/possibly published candidate. A minimized append-only safety journal prevents destructive Durable Object rollback/recreation from forgetting accepted promises, cancellations, accepted anchors, publication intents, independently verified terminal-success outcomes, or any ADR-023 lane-closing terminal disposition without becoming a second live state machine. Candidate success crosses ADR-020's Core-owned exact request+H0 semantic conformance audit and ADR-022's deterministic full commit-envelope proof, so the exact candidate object `C` is independently reproducible and cannot carry ambient author/committer metadata or unrelated packaged objects. Successful no-candidate `NO_OP`/`ALREADY_COMMITTED` claims cross ADR-021's fresh Core/repository verification so the finalizer cannot bypass the independent boundary by falsely declaring success. Unsuccessful terminal stale/failure/audit outcomes do not gain a second semantic verifier, but ADR-023 makes their terminalization rollback-durable before the caller is told they are final or the lane is released. Candidate `COMMITTED` likewise requires positive exact-C publication proof plus a rollback-independent terminal disposition. Canonical publication uses only the verified exact-C object closure plus a true expected-old Git update with publication-capability fencing; the current GraphQL expected-old path remains an implementation prototype, with the minimal Git publisher retained as safe fallback until exact-object identity is proven. The review against `1b4ff704…` found the terminal-disposition rollback gap, so implementation remains blocked until a later zero-edit full review of the new exact planning head passes.

Only after Phase 2.6 exits green should Phase 3 MCP implementation begin.