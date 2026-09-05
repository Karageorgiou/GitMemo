# ADR-019: Control-plane rollback recovery and managed-support hardening

Status: **Accepted**
Date: 2026-09-05
Tracking issue: #20
Amends: ADR-014, ADR-015, ADR-016, ADR-017, and ADR-018

## Context

The next Phase 2.6 adversarial review started from synchronized planning head `0f7f95c8220d16121144de5d1c1a4f42978550bd` and found three remaining pre-implementation gaps.

First, ADR-014 deliberately makes one repository Durable Object the sole live hosted lane/operation-state authority and says hosted-state loss must not make committed history ambiguous. ADR-017/ADR-018 then add state whose loss is correctness-sensitive: accepted canonical anchors, publisher-attempt fencing, and proven/possibly-published candidates which must survive an out-of-band rewrite that later excludes the candidate from canonical `HEAD`.

Cloudflare SQLite-backed Durable Objects provide point-in-time recovery which can restore the entire Durable Object SQLite/KV database to an earlier point. A Durable Object restore does not atomically roll back GitHub ref updates, already-created private evidence objects, external publication requests, issued GitHub installation tokens, or client-visible acceptance/cancellation decisions. A transparent restore can therefore recreate an older operation generation while external effects from the discarded interval still exist. It can also erase an ADR-018 protected publication anchor after the Git branch has already been rewritten away from that candidate.

The existing rule that ordinary retries reconstruct from canonical Git is not sufficient for this failure mode. Git remains semantic authority, but a current branch which excludes a previously published candidate cannot prove that the publication never happened. Likewise, a restored Durable Object cannot safely infer that an accepted request or successful cancellation never existed merely because the restored SQLite state predates it.

Second, the Runethread-managed private-memory validation workflow is already scheduled to change during the contract-v9/managed-bootstrap migration, but its current generated form references `actions/checkout@v7` and `actions/setup-go@v7`. Those are movable major-version tags. The Core development workflow already pins those same external Actions to full commit SHAs, and GitHub's current secure-use guidance states that a full-length commit SHA is the immutable Action reference. A retained validation/health/PR workflow which checks out private memory should not keep a weaker supply-chain pin merely because it is no longer triggered on every normal hosted push.

Third, ADR-015 changes project `current-state.md`/overview documents to non-authoritative asynchronous orientation projections in contract v9, but current generated/support text still calls `projects/` "canonical project state views." The generated memory README contains that wording, as does `docs/REPOSITORY_ROLES.md`. The current upgrader also replaces any native README matching a broad `# Runethread Memory` + `.runethread/lock.json` pattern with the generated README, which is too broad a basis for silently rewriting a user-customized support document during the v9 migration.

These findings do not require another coordinator or another semantic store. They require a rollback-independent safety record inside the existing private evidence-storage boundary, an explicit restore barrier, and stricter managed-support migration rules.

## Decision

### 1. Durable Object rollback is a recovery barrier, not an ordinary retry

A repository Durable Object remains the sole **live state-machine authority** for normal Phase 2.6 operation. However, its SQLite state is not treated as the only surviving evidence across an explicit point-in-time restore, namespace recreation, detected state rollback, or equivalent destructive control-plane recovery.

A Durable Object storage restore/recreation which can move persisted operation state backward is an exclusive recovery barrier:

- new mutation admission and publication are stopped for the affected repository before normal work resumes;
- the restored object MUST NOT simply continue from its restored phase/generation as if later external effects could not exist;
- any alarm state is treated as needing verification/repair rather than proof that all accepted work is still represented;
- publication capabilities which may have been issued after the restored point are conservatively fenced before ref-based publication resolution;
- the repository remains in maintenance/recovery until rollback-independent safety evidence and current GitHub state have been reconciled.

Production code MUST NOT use Durable Object PITR as a transparent operation retry mechanism. An intentional operator restore first enters maintenance/recovery and follows the same reconstruction rules as detected state loss.

### 2. The existing private evidence boundary also carries an append-only safety journal

Phase 2.6 uses the existing private object/evidence storage boundary to retain a small rollback-independent **repository safety journal**. This is not a second queue, semantic database, or publication authority. It is append-only recovery evidence which constrains what a restored/recreated Durable Object is allowed to forget.

Journal records are repository-binding-epoch scoped, immutable/create-if-absent, digest-bound, and ordered or hash-linked so a restored Durable Object cannot silently fork an earlier journal position. The implementation may use deterministic sequence keys, a hash chain, or an equivalently testable append-only scheme. Normal Container roles do not receive generic journal write/delete authority.

The journal contains only minimized operational metadata and opaque evidence references/digests. It MUST NOT contain plaintext memory Markdown/JSON, plaintext GitHub write tokens, App private keys, or other secret request bodies.

At minimum, rollback-sensitive facts are independently recorded as follows:

1. repository binding/enrollment epoch and canonical-ref identity;
2. every client-visible durable `ACCEPTED` operation, bound to its opaque sealed-request reference/digest;
3. cancellation decisions which won the cancellation-versus-publication race, before the cancelled operation is treated as durably terminal and its lane slot is released;
4. every accepted canonical-history anchor, including initial adoption and later validated out-of-band descendant adoption;
5. every exact publication intent **before any externally effective publication/token/executor/API I/O**, binding repository/ref/H0/C/operation/generation/publication-attempt/release identities plus a conservative fencing horizon when a delayed write capability may be issued;
6. definitive publication outcome evidence when the transport proves published or proves no write;
7. references/digests for existing immutable finalization/audit/publication receipts needed to reconstruct the safe state after rollback.

A journaled publication intent is conservative. If the Durable Object later loses the state saying whether publication I/O actually began, recovery treats the exact `C` as possibly published under ADR-018 until definitive evidence proves otherwise.

The active Durable Object still decides all normal transitions. A journal record by itself MUST NOT manufacture a semantic mutation, successful audit, publication authorization, or committed result. During recovery it may only restore already-established facts or force the system to remain closed until Git/evidence proves a safe outcome.

### 3. Rollback-sensitive transitions use write-ahead recovery evidence

Where a Durable Object decision could become unsafe if that local decision were later rolled back, the operation is not exposed as durably complete until the corresponding rollback-independent record exists.

Examples:

- `ACCEPTED` is not returned until the normal DO durability/alarm requirements and its immutable acceptance record are established;
- cancellation may atomically win locally, but the operation is not released as durably cancelled until its cancellation record is established;
- an accepted out-of-band canonical anchor is not used to reopen the normal lane until its immutable anchor record is established;
- `PUBLISHING` may be durably claimed in the DO, but no write-capable external publication action is started until the immutable publication-intent record exists.

Crash between the local claim and the external journal write is repaired idempotently. Crash after the journal write but before the final local acknowledgement is recovered from the journal rather than forgotten.

### 4. Restore/recreation recovery reconciles journal, evidence, and canonical Git

On a new/recreated repository runtime, after an explicit PITR restore, or whenever the Durable Object's stored journal checkpoint/binding epoch disagrees with rollback-independent recovery evidence, the repository enters recovery before normal writes.

Recovery must, at minimum:

1. verify the repository binding epoch, immutable repository identity, App installation, bound canonical ref, and directly observed private visibility;
2. enumerate/verify the rollback-independent journal tail and required immutable evidence/receipts;
3. restore accepted-but-not-terminal request identities which the rolled-back DO had forgotten;
4. honor any durable cancellation record before restarting semantic or publication work;
5. identify all journaled publication intents whose outcome was not definitively proven not-written and fence any capability which could still act;
6. resolve each proven/possible `C` under ADR-018 before ordinary descendant adoption;
7. reconstruct the latest accepted canonical-history anchor only from journal + independently verified Git/evidence, never from a guessed current tree;
8. re-establish the active/queued operation state and a recoverable alarm only after the above safety checks succeed.

If required safety-journal data is missing, corrupt, forked, or unavailable, normal hosted writes fail closed. The implementation does not create a fresh empty runtime for a repository known to have an existing binding epoch and then pretend prior hosted history never existed.

A deliberate deletion/disconnection may eventually remove recovery metadata under explicit account/data-retention policy, but re-enrollment after destructive hosted-state deletion is a new binding epoch and MUST NOT silently claim preservation of old hosted acceptance/publication knowledge. Any destructive-rebaseline semantics beyond that explicit boundary require separate reviewed design.

### 5. Recovery metadata has different retention from private content evidence

The short-retention privacy policy for request/candidate bodies remains. The safety journal is minimized operational metadata required to uphold durable acceptance, cancellation, and history-preservation guarantees across a control-plane rollback.

While a repository binding epoch remains active, safety-journal records needed to establish its accepted-history chain MUST NOT be deleted merely because normal candidate/request content reached its ordinary TTL. Plaintext content can expire while the small commit/digest/opaque-reference history remains.

Deletion of a repository binding is itself a control-plane barrier: outstanding publication capabilities are fenced and unresolved ADR-018 anchors are resolved or explicitly handed to operator recovery before safety metadata needed for correctness is removed.

### 6. The managed private-memory workflow uses immutable external Action pins

The contract-v9/managed-bootstrap release required by ADR-017 also hardens the generated `.github/workflows/validate.yml` supply chain.

Every external `uses:` action in the Runethread-managed private-memory workflow MUST be pinned to a verified full-length Git commit SHA. Human-readable version comments may accompany the SHA. Movable tags or branches such as `@v7` are not accepted for the managed workflow.

This applies to GitHub-authored Actions as well as any future third-party action/reusable component. Tests for the generated workflow MUST fail if an external Action reference is not a full immutable commit SHA.

The exact current managed workflow remains a recognized historical migration source even though it contains the older movable major tags. Customized/unrecognized workflow state is still never silently overwritten. Trigger reduction, immutable Action pinning, and historical-source recognition ship together through the released starter/upgrader/template/private-repository migration.

This rule does not require GitHub Actions for the normal hosted mutation data plane. It hardens the distinct validation/PR/recovery/health workflow which remains after ADR-017 removes redundant push-on-every-memory execution.

### 7. Contract-v9 support text must agree with project-view authority semantics

The v9 migration does not change existing `projects/.../current-state.md` bytes merely to relabel their authority. It **does** update Runethread-managed/generated support text so users and agents are not told that asynchronous project views are canonical sources.

At minimum:

- the generated memory README describes `projects/` current-state/overview files as non-authoritative orientation/materialized views;
- `docs/REPOSITORY_ROLES.md` and other current Runethread support documentation are updated to use the same authority model;
- contract-v9 normative files continue to control whenever support prose conflicts;
- upgrader tests prove project-view user bytes remain unchanged by this compatibility migration.

The upgrader MUST NOT use the current broad `# Runethread Memory` + `.runethread/lock.json` pattern alone as authority to replace an arbitrary native README during this migration. Automatic README replacement is allowed only for an exact recognized prior Runethread-managed README/source state. A customized/unrecognized README is preserved or surfaced for explicit/manual reconciliation; it is not silently overwritten merely to update wording.

README/support-file recognition is a managed-bootstrap migration concern, not a reason to add README to `ContractPaths()`.

### 8. Hosted release barriers include the recovery protocol

The hosted release/version barrier now also covers:

- repository binding-epoch representation;
- safety-journal schema/keying/hash/sequence protocol;
- write-ahead acceptance/cancellation/publication/anchor receipts;
- restore/recreation recovery logic and fencing semantics;
- managed private-memory workflow immutable pin policy;
- supported managed README/workflow historical-source recognition.

An incompatible release MUST NOT read an older journal/evidence state and silently reinterpret it. Upgrade/drain/maintenance or explicit versioned recovery is required.

## Consequences

- The Durable Object remains the single normal state-machine coordinator; no Cloudflare Workflow or second live queue/database is introduced.
- Durable Object PITR becomes safe to use only as an explicit maintenance/recovery operation rather than a hidden rollback of the distributed system.
- Accepted work, cancellation, accepted history, and possibly-published candidates cannot silently disappear merely because the DO database is restored to an earlier point.
- R2/object evidence already required by ADR-016/ADR-018 gains a small append-only recovery role, with strong separation between minimized safety metadata and short-lived private content.
- Recovery may remain blocked when evidence is missing rather than risking duplicate semantic mutation or resurrecting cancelled work.
- The retained private-memory GitHub Actions surface now matches Runethread's own immutable-Action-pinning standard.
- Contract-v9 users and agents are no longer given contradictory "canonical project view" wording by Runethread-managed support files.
- Custom README/workflow content is not silently replaced under broad textual heuristics.
- This ADR records material changes found by the architecture-freeze attack. That review therefore does **not** satisfy the zero-edit gate, and implementation remains blocked pending another full review of the new exact synchronized planning head.

## Alternatives considered

### Treat Durable Object PITR as equivalent to a process restart

Rejected. A process restart preserves Durable Object storage; PITR intentionally rewinds it while GitHub/R2/external requests are not atomically rewound.

### Trust current Git alone after a DO rollback

Rejected. ADR-018 demonstrates a valid state where a publication committed and a later force-push removed `C` from current ancestry. Current Git can then prove neither that `C` never existed nor that a rolled-back cancellation/acceptance decision did not occur.

### Mirror the complete Durable Object database into a second database

Rejected. That would create another mutable state-machine owner and duplicate queue/retry semantics. The safety journal is append-only, minimized, and recovery-only.

### Use provider logs/webhooks as the recovery journal

Rejected. Delivery/retention are not the architecture's durable correctness contract and webhooks are already hints only.

### Disable or ignore PITR

Rejected as the sole answer. Accidental/destructive state loss and future operator recovery still require an explicit safety model, and Cloudflare exposes rollback as a supported storage capability.

### Keep `actions/*@v7` because the actions are GitHub-authored

Rejected. GitHub's own secure-use guidance identifies full-length commit SHA pinning as the immutable reference, and Runethread already enforces that standard on its Core validation workflow.

### Add the generated README to the operational contract

Rejected. The contract-v9 semantic rules remain in verified `ContractPaths()`. README wording is managed support/bootstrap state and should be migrated without expanding the normative contract surface unnecessarily.

### Replace every native README matching the Runethread heading

Rejected. A heading plus lock-file reference does not prove the entire README is still Runethread-managed bytes; a user may have customized it.

## Verification

Implementation satisfies this ADR only if tests/evidence demonstrate at minimum:

1. a repository DO restored to a bookmark before a previously returned `ACCEPTED` operation reconstructs that accepted operation from rollback-independent evidence instead of forgetting it;
2. a restore to before a successful cancellation cannot resurrect the cancelled operation into finalization/publication;
3. a restore to before `PUBLISHING` while a journaled publication intent exists treats exact `C` as possibly published and fences/reconciles it under ADR-018;
4. exact `H0 -> C` succeeds, the DO is restored to before publication, and the branch is later rewritten to exclude C: recovery does not re-run the semantic mutation or adopt the rewrite as though C never committed;
5. accepted out-of-band canonical anchors survive DO rollback and still constrain later descendant/non-descendant reconciliation;
6. a restored/recreated DO with a stale journal checkpoint cannot begin normal admission/publication until the rollback-independent journal tail is verified and reconciled;
7. missing/corrupt/forked required safety-journal data fails closed rather than silently creating a fresh empty runtime;
8. journal records are immutable/create-if-absent, repository-binding-epoch scoped, digest-bound, and cannot be authored/deleted by finalizer/auditor/public clients;
9. safety-journal records contain no plaintext memory body or GitHub write credential;
10. a crash before/after each write-ahead acceptance/cancellation/publication/anchor record is idempotently recoverable;
11. recovery repairs required alarms only after accepted work and publication/cancellation history are reconstructed;
12. ordinary non-restored execution still uses the DO as the sole live transition authority and does not consult a second mutable queue;
13. the generated private-memory workflow uses full-length verified commit SHAs for every external `uses:` action and tests reject tag/branch references;
14. the exact previous managed workflow remains recognized as a supported migration source while customized workflow bytes are refused/preserved rather than overwritten;
15. the generated v9 memory README and current support docs describe project current-state/overview prose as non-authoritative orientation/materialized views;
16. v8 -> v9 migration preserves existing project-view bytes while updating only explicitly managed/support/control-plane state;
17. an exact recognized prior managed README may be updated, but a customized/unrecognized native README is not silently replaced by the broad heading/lock heuristic; and
18. incompatible safety-journal/recovery protocol versions are stopped by the hosted release barrier rather than silently mixed.