# ADR-023: Rollback-durable terminal operation dispositions

Status: **Accepted**
Date: 2026-09-05
Tracking issue: #20
Amends: ADR-014, ADR-019, ADR-021, and ADR-022 where they define lane-releasing operation outcomes and rollback recovery

## Context

This Phase 2.6 architecture-freeze review started from exact synchronized planning head `1b4ff70402f8aa298c4e219e0e9254e998f037ca`.

ADR-019 correctly makes several rollback-sensitive facts write-ahead durable outside the Durable Object PITR domain: client-visible `ACCEPTED`, winning cancellation, accepted canonical-history anchors, and publication intent before write-capable external I/O. ADR-021 additionally makes independently verified `NO_OP` / `ALREADY_COMMITTED` terminal-success evidence rollback-recoverable before those successful no-candidate results release the lane.

A remaining gap exists for the general terminal-operation boundary.

ADR-014 stores the final committed revision or failure outcome in ordinary Durable Object state. ADR-019 says the safety journal retains definitive publication outcomes and references to immutable receipts needed for recovery, but it does not require **every client-visible terminal disposition which releases an accepted operation from the hosted lane** to become rollback-independent before that terminal state is exposed. Its write-ahead examples cover acceptance, cancellation, accepted-anchor adoption, and publication authorization, not ordinary lane-releasing failure/stale results or the final candidate `COMMITTED` disposition.

That permits a destructive control-plane rollback to resurrect work which the caller was already told was finished.

For example:

1. operation `O1` is durably accepted and therefore has an ADR-019 acceptance record;
2. the finalizer deterministically returns a request-local failure, or Core returns `NEEDS_REPREPARE` for proven-uncommitted stale work;
3. the repository DO records the terminal result in SQLite, returns that terminal result to the caller, releases the lane, and later accepted operation `O2` proceeds;
4. the DO is restored to a point before `O1` terminalized;
5. the rollback-independent journal still proves that `O1` was accepted, but there is no required rollback-independent terminal-disposition record proving that `O1` had already ended;
6. recovery can therefore reconstruct `O1` as accepted live work and execute it again.

The same structural problem exists for a candidate-producing `COMMITTED` result if the caller was told success before the definitive publication outcome/terminal disposition was made rollback-independent. ADR-018's publication-intent/protected-candidate rules prevent unsafe guessing about whether `C` may have published, but a mere possibly-published intent is not the same durable fact as a terminal success which was already returned to the caller.

This is not the same trust problem solved by ADR-021. A compromised finalizer can always deny service by returning failure, and Phase 2.6 intentionally does not require a second semantic replay merely to prove every unsuccessful result. The missing invariant is **durability of the terminal decision once the hosted control plane has committed to exposing it and releasing/reordering later work**.

## Decision

### 1. Every durable lane-releasing terminal disposition crosses one rollback-independent terminalization barrier

For every accepted hosted operation, any result which is exposed as a durable terminal status, causes the operation to stop being live, or releases its serialized lane position MUST have a rollback-independent immutable terminal-disposition record established first.

This rule covers at least:

- candidate publication `COMMITTED`;
- independently verified `NO_OP`;
- independently verified `ALREADY_COMMITTED`;
- `NEEDS_REPREPARE`;
- request-local deterministic finalization failure;
- deterministic audit failure when the operation itself is terminalized even if the repository lane also enters suspension/reconciliation;
- `CANCELLED`;
- any other future result class whose semantics are "this accepted hosted operation will not continue".

A retry/backoff state, in-doubt publication, transient provider error, or lane-level reconciliation state is not a terminal disposition merely because progress is temporarily blocked.

### 2. Terminal-disposition evidence is minimized, immutable, and bound to the exact operation

The existing ADR-019 safety-journal/evidence boundary carries the terminal-disposition record. No second live state machine or semantic database is introduced.

The record binds at minimum:

- repository binding epoch and immutable repository identity;
- canonical-ref identity;
- hosted operation/attempt identity;
- exact sealed-request reference/digest;
- Core idempotency identity when applicable;
- terminal result class;
- phase/execution generation which produced or authorized the disposition;
- pinned hosted/Core/contract/protocol identities;
- the exact immutable finalization/audit/terminal-verification/publication evidence reference(s) on which the disposition relies;
- canonical revision/history identity required to interpret the result, such as `H0`, exact committed `C`, or the directly observed stale/current revision when applicable;
- a digest of the stable client-visible result payload or equivalent release-defined result identity.

The record contains no plaintext memory body, GitHub credential, or unrestricted provider error/log payload.

For one logical hosted operation, conflicting terminal-disposition records are integrity failure. Exact duplicate creation is idempotent; a byte-different attempt to terminalize the same operation inconsistently fails closed.

### 3. Terminalization is write-ahead with respect to client-visible completion and lane release

The repository DO may first win a local atomic transition identifying the intended terminal result, but it MUST NOT:

- return that result as durably terminal to the caller;
- release the operation's queue/lane slot;
- admit a later transition whose correctness assumes the operation can never run again; or
- garbage-collect evidence needed to prove the terminal result

until the exact terminal-disposition record is durably established and re-readable.

Crash before the terminal-disposition write leaves the operation conservatively nonterminal and recovery may finish the same terminalization from immutable evidence. Crash after the rollback-independent write but before local acknowledgement reconstructs the terminal result from that record.

Existing ADR-019 cancellation ordering and ADR-021 terminal-success ordering are special cases of this general rule.

### 4. Candidate `COMMITTED` success requires durable positive publication disposition before success is exposed

For candidate-producing operations, publication intent remains write-ahead before any write-capable external I/O under ADR-019, and publication ambiguity remains governed by ADR-018.

A caller-visible durable `COMMITTED` result is stronger than a publication intent or possibly-published candidate. Before exact `COMMITTED` is returned/released:

1. exact `C` must be proven published under ADR-018, either by trusted exact-publication success or independently verified canonical ancestry;
2. the definitive positive publication evidence required by ADR-019 must be durably retained/referenced; and
3. the rollback-independent terminal-disposition record must bind the operation to exact committed `C` and that positive evidence.

If publication is still indeterminate, the operation remains live/in-doubt or reconciliation-required and MUST NOT be journaled/exposed as terminal `COMMITTED` merely because `C` might have published.

If a later owner rewrite removes proven committed `C`, ADR-018 still controls canonical recovery. The terminal-disposition record preserves the additional fact that the caller had already been told the operation committed; recovery must not downgrade that historical operation to an uncommitted semantic retry.

### 5. Unsuccessful terminal dispositions are durable without acquiring a second semantic verifier

ADR-021's trust distinction remains intact.

`NEEDS_REPREPARE`, request-local failure, deterministic audit failure, and equivalent unsuccessful terminal outcomes do **not** require a fresh second semantic execution solely to defend against a compromised finalizer. A false failure is an availability attack, not a false canonical success claim.

Once the repository DO accepts such a result under the existing generation/evidence/classification rules and chooses to expose it as terminal, however, the terminalization itself becomes durable under this ADR.

The rollback-independent record therefore proves **which terminal disposition the trusted hosted state machine committed to exposing**, not that an untrusted component could never have caused denial of service.

### 6. Recovery never resurrects a journaled terminal operation

During ADR-019 PITR/recreation recovery, the repository runtime reconstructs terminal-disposition records before rebuilding the active/queued lane.

An accepted operation with an intact terminal-disposition record is restored as that same terminal outcome and MUST NOT be requeued for semantic finalization, audit, publication, or stale reclassification.

Examples:

- journaled `NEEDS_REPREPARE` stays terminal even if the current ref later happens to equal the operation's old `H0`; a new semantic attempt requires a new client preparation/submission or another explicitly designed operation identity;
- journaled request-local failure stays terminal rather than being retried against newer repository state;
- journaled `COMMITTED(C)` stays historically committed and follows ADR-018 if current canonical ancestry later excludes `C`;
- journaled `NO_OP` / `ALREADY_COMMITTED` restores the exact independently verified success already required by ADR-021;
- journaled `CANCELLED` remains cancelled under ADR-019.

Missing/corrupt terminal evidence for a journaled disposition fails closed. Recovery does not silently reinterpret a terminal record into a different result class.

### 7. Terminal disposition orders later accepted work during rollback recovery

The append-only safety-journal ordering used by ADR-019 must preserve enough ordering information to reconstruct which accepted operations had terminalized before later lane work became eligible.

Recovery MUST NOT replay an earlier terminal operation merely because later operations accepted or committed after it are also present in the journal. The terminal-disposition record closes that operation's position in the serialized lane.

This does not make the journal a normal queue. In ordinary execution the DO remains the sole live scheduling/transition authority; the journal is consulted for this ordering only during destructive recovery/reconstruction.

### 8. Retention and release barriers include terminal-disposition protocol

While a repository binding epoch remains active, minimized terminal-disposition records required to prevent resurrection or preserve returned success/failure history follow the ADR-019 correctness-retention policy rather than the shorter private-content TTL.

The hosted release/version barrier additionally covers:

- terminal-disposition schema/keying/result classes;
- result-payload identity/digest rules;
- publication-success-to-terminalization ordering;
- terminal reconstruction/queue-closing semantics.

An in-flight operation cannot cross incompatible terminal-disposition protocol versions.

## Consequences

- A Durable Object restore cannot silently turn a previously returned terminal failure/stale result back into executable work.
- A returned candidate `COMMITTED` success remains distinguishable from a merely possible publication after rollback.
- ADR-021 does not expand into expensive replay of every failure; unsuccessful results gain durability, not independent semantic truth proof.
- Cancellation, no-op/already-committed success, candidate committed success, stale terminalization, and request-local failure use one coherent rollback terminalization rule.
- The safety journal remains recovery-only and minimized; the repository DO remains the sole live state-machine authority.
- Some terminal responses may wait for one additional small immutable journal write before the lane is released. This is intentional durability cost at an externally visible completion boundary.
- This review therefore fails the zero-edit architecture-freeze gate. Implementation remains blocked until a later full adversarial review of the new exact synchronized planning head itself requires zero architecture/planning edits.

## Alternatives considered

### Reconstruct only from acceptance records and rerun anything not present in restored DO state

Rejected. That can execute an operation which the caller was already told had failed, gone stale, been cancelled, or otherwise terminalized.

### Rely on immutable finalization/audit receipts existing somewhere in object storage

Rejected as the complete recovery rule. Without a required rollback-independent terminal-disposition reference/result identity before lane release, recovery may not know which receipt was authoritative or that the operation had already been exposed as terminal.

### Treat publication intent as sufficient durable `COMMITTED` evidence

Rejected. Publication intent deliberately means "possibly published" after ambiguous loss; it does not prove the caller-visible committed result.

### Independently replay every unsuccessful result before terminalizing it

Rejected. ADR-021 intentionally protects false success while not claiming availability against a compromised finalizer. Terminal durability does not require changing that trust boundary.

### Keep terminal failures rollback-prone because retrying them is conservative

Rejected. Retrying an operation after the caller was told it was terminal is not conservative: repository state may have changed, and the operation may later mutate canonical Git without a new client action.

## Verification

Implementation satisfies this ADR only if tests/evidence demonstrate at minimum:

1. a returned `NEEDS_REPREPARE` followed by DO restore to before that result reconstructs the same terminal outcome and never re-runs `ApplyMutation` automatically;
2. a returned request-local finalization failure followed by rollback does not resurrect the operation against newer canonical state;
3. a deterministic terminal audit failure remains terminal after rollback even when the lane separately stays suspended/reconciliation-required;
4. a candidate exact `C` is proven published and returned `COMMITTED`, then the DO is restored to before publication and the branch is rewritten to exclude C: recovery restores the committed terminal fact and applies ADR-018 rather than treating the operation as merely uncommitted accepted work;
5. an indeterminate/possibly-published candidate cannot acquire a `COMMITTED` terminal-disposition record until positive publication proof exists;
6. exact duplicate terminal-disposition writes are idempotent while conflicting result classes/evidence for one operation fail closed;
7. terminal-disposition write failure after local terminal claim but before response prevents durable terminal response/lane release and is safely retried;
8. crash after terminal-disposition record creation but before local DO acknowledgement reconstructs the exact terminal result;
9. recovery rebuilds acceptance/terminal ordering so later accepted/committed work does not cause an earlier terminal operation to be replayed;
10. `NO_OP` / `ALREADY_COMMITTED` still require ADR-021 independent verification before their terminal-disposition record is accepted;
11. unsuccessful terminal outcomes do not acquire a new independent semantic verifier solely because of this ADR;
12. `CANCELLED` retains ADR-019 cancellation-vs-publication linearization and is represented consistently by the general terminalization rule;
13. terminal-disposition journal records contain no plaintext memory body, credential, or unrestricted provider error text;
14. missing/corrupt referenced terminal evidence fails closed without changing the stored result class;
15. binding deletion/disconnect cannot remove terminal-disposition safety metadata while it is still required to prevent operation resurrection or preserve protected committed history; and
16. local/offline MemoryService behavior remains unchanged by hosted terminal-disposition recovery.