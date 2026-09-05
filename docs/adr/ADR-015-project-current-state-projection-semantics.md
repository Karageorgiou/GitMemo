# ADR-015: Project current-state documents are asynchronous orientation projections

Status: **Accepted**
Date: 2026-09-05
Tracking issue: #20
Related: ADR-001, ADR-002, ADR-014

## Context

Runethread contract v8 already distinguishes project `current-state.md`/overview documents from atomic memories: they are fast orientation views and are not replacements for atomic memories. However, the same immutable v8 `MEMORY_PROTOCOL.md` also requires an operator to synchronize the relevant current-state view when a new memory materially changes present project state, and treats affected current-state synchronization as part of memory-write completion.

That wording is part of the hash-pinned operational contract. It cannot be relaxed by an architecture ADR or hosted implementation without a new contract release.

Phase 2.6 initially proposed treating project current-state prose as an asynchronously refreshed materialized/orientation view. Adversarial review found that doing so while claiming the v8 contract remained unchanged would be a semantic compatibility violation.

The alternative is to force every hosted atomic memory transaction to also rewrite curated project-summary prose. That would require one of:

- letting Cloudflare/provider code become a second semantic writer for project prose;
- expanding `MemoryService.ApplyMutation` into a compound memory-plus-freeform-project-summary transaction; or
- publishing a memory commit and then requiring a second semantic summary commit before the operation can be called complete.

All three increase the critical mutation surface. The second requires an AI-proposed project summary to become part of the deterministic memory transaction, while the third creates a canonical intermediate state and failure/retry coupling between two semantically different writes.

Industry-standard source-of-truth/materialized-view separation points the other way: authoritative records commit independently; non-authoritative orientation projections may lag and carry explicit freshness/verification semantics.

## Decision

Runethread's **next operational contract version (contract v9)** changes project current-state/overview synchronization semantics so these documents are explicitly **asynchronous orientation/materialized projections**, not a completion dependency of an otherwise valid atomic-memory mutation.

This ADR records the architecture decision. It does **not** modify the current v8 contract bytes. A separate contract-change implementation must follow the normal contract/version/migration/release gates before a hosted Phase 2.6 path may rely on the new semantics for a repository.

### Authority

Under contract v9:

- canonical atomic memories remain authoritative for durable memory facts/history;
- authoritative project repositories remain authoritative for current source-code/project-source facts;
- project current-state/overview prose remains a convenience orientation projection;
- project current-state/overview prose MUST NOT become the only durable location of a fact that should be an atomic memory;
- conflict with validated atomic memory or authoritative project source is resolved in favor of the applicable authoritative source, not the orientation projection.

### Memory-write completion

A valid atomic-memory mutation is complete when its canonical memory/lifecycle/relationship/index/trust/validation invariants pass and it is committed according to the applicable delivery policy.

It does **not** require a project current-state/overview projection to be rewritten in the same transaction or before the atomic memory mutation may be reported committed.

Phase 2.6 provider code MUST NOT hand-edit project-summary prose as a hidden second semantic mutation engine merely to emulate the v8 synchronization wording.

### Projection freshness

Contract v9 must make it explicit that a project orientation view can be older than the atomic memories or live project source it summarizes.

The contract update must define retrieval behavior conservatively:

- retain an explicit `Last reviewed`/equivalent freshness signal in project current-state documents;
- when current correctness materially depends on project state, verify against authoritative project source when available;
- when a project view appears stale, conflicts with canonical memory/source, or lacks adequate freshness evidence, treat it as incomplete orientation and fall back to targeted canonical-memory/project-source retrieval;
- never promote stale orientation prose to authoritative fact merely because it is the first retrieval entry point.

The exact representation of any additional projection freshness metadata is a contract-implementation decision. It MUST NOT be invented as a canonical semantic-memory field solely for hosted delivery.

### Projection refresh

Refreshing a project orientation view is a separate semantic/projection operation from an atomic-memory mutation.

A future deterministic or semantically assisted projection service may:

1. read current authoritative project memories and, when relevant, live project-source evidence;
2. produce a concise proposed current-state view;
3. verify/record its freshness/source basis;
4. update the view under ordinary exact-revision/concurrency safeguards.

That refresh may be triggered after meaningful project-memory changes, periodically, on project access, or explicitly. Phase 2.6 does not need to design the general refresh scheduler before memory delivery can ship.

Projection refresh MUST NOT silently rewrite atomic memories to agree with the projection.

### Contract-v9 migration prerequisite

Before the Phase 2.6 hosted data plane relies on asynchronous project-view semantics for a repository, the repository must be migrated to the contract release containing this decision.

The contract-change implementation must include at least:

- new contract version/release identity and immutable contract bytes/digests;
- updated `MEMORY_PROTOCOL.md` wording removing project-view synchronization as a mandatory atomic-memory completion condition;
- retrieval/failure wording reflecting potentially stale orientation views;
- exact historical contract-v8 fixture/migration coverage;
- trust/lock/bootstrap compatibility and unsupported-state tests;
- `runethread/memory-template` migration;
- known private-memory repository migration after the immutable release is independently verified;
- proof that canonical atomic-memory UUIDs/content/provenance/relationships are preserved unless an explicitly reviewed representation change requires otherwise;
- preservation of existing project current-state prose bytes during the compatibility migration unless the contract implementation explicitly requires a representation update.

A contract-v8 repository continues to be governed by the v8 synchronization rule until explicitly migrated. The hosted service MUST NOT silently reinterpret v8 as v9.

## Consequences

- Phase 2.6 no longer hides a contract semantic change inside hosted-provider policy.
- The first implementation work after architecture freeze includes the contract-v9 change/migration before normal hosted writes rely on eventual project-view freshness.
- Atomic memory delivery remains small and deterministic instead of becoming a freeform project-summary dual write.
- Project orientation can be refreshed independently and optimized later without changing canonical atomic-memory transactions.
- Retrieval must remain freshness-aware because an orientation view may legitimately lag.
- Existing contract-v8 repositories retain their published behavior until explicit supported migration.

## Alternatives considered

### Pretend v8 already allows eventual project-view staleness

Rejected. The immutable v8 protocol explicitly requires relevant current-state synchronization as part of write completion.

### Add project-summary prose to every `ApplyMutation` request

Rejected. It couples one atomic memory mutation to a freeform semantic summary and expands Core's deterministic transaction boundary unnecessarily.

### Publish memory first, then require a second current-state commit before success

Rejected as the normal model. It creates a canonical intermediate state, compound recovery semantics, and another race/failure boundary for a non-authoritative view.

### Let provider/workflow code update the summary directly

Rejected. It creates a second semantic mutation implementation outside Core/application boundaries.

### Delete current-state views entirely

Rejected. They remain useful orientation accelerators when freshness/authority is represented honestly.

## Verification

The contract-v9 implementation satisfies this ADR only if evidence demonstrates:

1. `MEMORY_PROTOCOL.md` and any other affected contract files no longer require project-current-state synchronization as a prerequisite for atomic-memory completion;
2. project current-state/overview documents remain explicitly non-authoritative orientation views;
3. retrieval/failure rules handle stale/unknown-freshness project views conservatively;
4. a normal atomic-memory Apply/hosted publication can complete without modifying project current-state prose;
5. provider code does not become a hidden project-summary semantic writer;
6. contract v8 remains immutable and is represented by an exact historical fixture;
7. v8 -> v9 migration is deterministic, supported, and trust/lock/bootstrap verified;
8. canonical atomic-memory identities/content/provenance/relationships are preserved across the migration except for separately approved representation changes;
9. existing project current-state prose is preserved during migration unless a separately reviewed representation change requires otherwise;
10. template migration passes permanent bootstrap validation before a real private-memory migration;
11. a contract-v8 repository is never silently treated as having v9 project-view semantics;
12. future projection refresh can be added without changing atomic-memory canonicality or reintroducing a dual-write requirement.