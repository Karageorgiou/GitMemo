# ADR-022: Deterministic candidate commit envelope

Status: **Accepted**
Date: 2026-09-05
Tracking issue: #20
Amends: ADR-014, ADR-016, and ADR-020 where they define candidate construction, audit, and exact-object publication

## Context

The next Phase 2.6 architecture-freeze review started from exact synchronized planning head `9151c9d2e1a383e79449af2963fc2c547bb49429`.

ADR-020 correctly requires the fresh auditor to derive the semantic mutation independently from exact `H0` + the immutable sealed request and compare that result to exact candidate `C`. It also compares the expected parent, semantic tree/delta, mutation message/trailers, request fingerprint, strict Index state, scope, and trust/repository validity.

However, ADR-020 deliberately allowed separately replayed commits to have different object IDs because current Git commit construction includes author/committer metadata such as timestamps. It required any allowed nondeterministic commit-object fields to be bounded, but did not make the complete candidate commit envelope an independently derived invariant.

That leaves a trust-boundary channel.

Current Core creates the mutation commit by invoking Git roughly as:

```text
git -c user.name=Runethread \
    -c user.email=runethread@localhost \
    commit --no-gpg-sign -m <Core-owned message>
```

Git documents that `GIT_AUTHOR_NAME`, `GIT_AUTHOR_EMAIL`, `GIT_AUTHOR_DATE`, `GIT_COMMITTER_NAME`, `GIT_COMMITTER_EMAIL`, and `GIT_COMMITTER_DATE` environment variables take precedence over the corresponding Git configuration values. A faulty or compromised finalizer can therefore keep the exact authorized tree, parent, and Core mutation message while constructing a different exact commit object whose author/committer identity or date fields contain attacker-chosen values.

Because Phase 2.6 publishes exact audited candidate object `C`, unchecked commit metadata would become durable canonical Git history. Besides spoofing identity and defeating deterministic candidate identity, free-form author/committer fields can be used as a persistence/exfiltration channel for bytes from the private sealed request or runtime environment even when those bytes are not part of the authorized semantic mutation.

The candidate trust boundary must therefore cover the complete published Git commit envelope, not only its semantic tree and message.

## Decision

### 1. Exact candidate `C` has a Core-owned commit envelope

For hosted candidate-producing mutations, every field of the exact commit object that can affect `C` is either:

- independently derived from immutable operation inputs by the pinned Core release; or
- explicitly forbidden.

At minimum the accepted hosted candidate envelope binds:

- exactly one parent: exact `H0`;
- exact independently derived candidate tree;
- exact Core-owned mutation subject/message/trailers and normalized request fingerprint;
- exact release-defined author name and email;
- exact release-defined committer name and email;
- author and committer timestamps deterministically derived from a Core-owned immutable input;
- the canonical encoding/normalization rules used to construct the commit object;
- the absence of unapproved commit headers such as signatures, merge/tag headers, alternate encoding declarations, or future extension headers unless that exact header is deliberately introduced by a later compatible release/protocol decision.

A candidate is not publishable merely because its tree/message are correct if any other commit-object field falls outside the release-defined envelope.

### 2. Candidate commit time uses deterministic request-bound input

Every candidate-producing `ApplyMutation` operation already requires an RFC3339 `mutation_time`, and that value is part of the normalized request fingerprint and the authoritative memory temporal update.

The hosted candidate commit envelope therefore derives author/committer time deterministically from the exact normalized sealed request's `mutation_time` under one release-defined canonical Git timestamp encoding, for example the equivalent instant normalized to a fixed timezone representation.

Ambient wall clock, Container locale/timezone, Git defaults, or inherited `GIT_AUTHOR_DATE` / `GIT_COMMITTER_DATE` values are not candidate identity inputs.

A future Core release may deliberately choose a different deterministic commit-time rule only through the normal release/protocol compatibility barrier. It may not fall back to unconstrained runtime environment state.

### 3. Core commit construction sanitizes Git identity/config environment

The Core-owned candidate-construction path must not rely on `-c user.name` / `-c user.email` alone.

Before Git commit-object construction, the runtime establishes an explicit sanitized environment/config boundary which prevents inherited process/repository/global Git state from overriding the candidate envelope. At minimum it controls or clears:

- `GIT_AUTHOR_NAME`;
- `GIT_AUTHOR_EMAIL`;
- `GIT_AUTHOR_DATE`;
- `GIT_COMMITTER_NAME`;
- `GIT_COMMITTER_EMAIL`;
- `GIT_COMMITTER_DATE`;
- fallback identity such as `EMAIL`;
- Git config/include surfaces capable of changing commit encoding, signing, hooks, or other commit-object construction behavior.

The existing hosted Git hardening against hooks, filters, credential helpers, unsafe includes, recursive submodules, and similar execution surfaces remains required.

### 4. Fresh audit parses and proves the raw commit object

ADR-020 request-to-candidate conformance remains required, but the auditor additionally parses the raw exact candidate commit object and compares the complete allowed envelope against independently derived Core expectations.

The audit fails on any unexpected author/committer identity, timestamp, parent count/order, message/trailer bytes, signature/header, encoding declaration, or other commit-object field even when the candidate tree is otherwise exactly correct.

A finalizer-supplied description of commit metadata is evidence to compare, not authority.

### 5. Deterministic envelope upgrades ADR-020 replay equality

For a hosted release implementing this ADR, exact candidate commit construction is deterministic from:

- exact `H0`;
- exact immutable sealed request / normalized fingerprint;
- exact independently derived semantic/index tree;
- pinned Core/contract/runtime candidate-envelope version.

Therefore the fresh Core-owned replay/conformance path may and SHOULD derive the exact expected commit object ID and require exact equality with audited candidate `C`.

Where ADR-020 previously allowed replay-created commit SHA inequality solely because commit metadata was nondeterministic, this ADR controls for releases implementing the deterministic envelope. An implementation may compare the raw expected commit object directly or construct it through a deterministic Core helper; it does not substitute a separately created `C2` for the candidate. Exact `C` remains the publication object.

### 6. Candidate transport cannot make unrelated Git objects publishable

Candidate packages may contain transport framing and whatever exact reachable Git objects are needed to materialize `C` relative to `H0`, but finalizer-supplied extra unreachable Git objects are not part of the authorized candidate.

The auditor/publisher boundary must derive the object closure required by exact `C` and must not upload/import unrelated unreachable Git objects solely because they were present in a finalizer package. Byte-different or extra-object package content outside the verified candidate closure is rejected, ignored as nonauthoritative transport junk, or otherwise prevented from reaching canonical Git object storage by the accepted publication path.

This preserves the exact-object publication rule and closes a second persistence channel adjacent to free-form commit metadata.

### 7. Terminal-success paths remain governed by ADR-021

`NO_OP` and `ALREADY_COMMITTED` produce no new candidate commit and therefore do not use this candidate-envelope rule. Their independent success verification remains governed by ADR-021.

Candidate-producing success still requires ADR-020 semantic conformance plus all existing trust/index/scope checks, now strengthened by this full commit-envelope proof.

### 8. Release barriers include candidate-envelope identity

The hosted release/version barrier additionally covers:

- candidate author/committer identity constants;
- deterministic timestamp derivation/encoding;
- raw commit-header allowlist;
- deterministic commit-construction helper/protocol;
- candidate package object-closure rules.

An operation cannot cross incompatible candidate-envelope versions.

## Consequences

- A compromised/faulty finalizer cannot preserve a valid semantic tree while smuggling arbitrary private bytes or spoofed identity through author/committer metadata.
- Exact candidate object identity becomes deterministic and independently reproducible for the hosted mutation path.
- ADR-020 audit becomes simpler/stronger for releases implementing this ADR because exact replayed candidate object identity can be compared directly to exact `C`.
- Ambient Container/Git configuration no longer participates in canonical candidate identity.
- Publication transports import/push only the verified object closure of exact `C`, not arbitrary objects supplied in candidate evidence.
- No new coordinator, semantic engine, queue, database, or publication authority is introduced.
- This review therefore fails the zero-edit architecture-freeze gate. Implementation remains blocked until a later full review of the new exact synchronized planning head itself requires zero architecture/planning edits.

## Alternatives considered

### Keep timestamps nondeterministic but merely bound them to a time window

Rejected. A bounded timestamp still makes candidate identity depend on runtime state and leaves unnecessary encoding capacity. Candidate-producing requests already contain a deterministic mutation time.

### Trust `-c user.name` / `-c user.email`

Rejected. Git environment variables override those configuration values.

### Ignore author/committer metadata because memory tree bytes are correct

Rejected. The exact commit object is canonical published Git history. Unchecked metadata is persistent user-visible state and can carry bytes outside the authorized mutation.

### Let the auditor compare only tree + message and accept any remaining commit headers

Rejected. That is the hole this ADR closes.

### Strip/rewrite candidate metadata in the publisher

Rejected. Publication must remain exact audited `C`; the publisher must never repair or manufacture `C2`.

### Upload every object found in the candidate package and rely on ref reachability

Rejected. Unreachable objects are not part of the authorized candidate and need not be persisted in GitHub merely because an untrusted finalizer supplied them.

## Verification

Implementation satisfies this ADR only if tests/evidence demonstrate at minimum:

1. two fresh candidate constructions from exact same `H0` + sealed request + pinned release produce exact same candidate commit SHA;
2. candidate author/committer name and email equal the release-defined Core identity exactly;
3. candidate author/committer dates equal the deterministic canonical encoding derived from exact request `mutation_time`;
4. setting `GIT_AUTHOR_NAME`, `GIT_AUTHOR_EMAIL`, `GIT_COMMITTER_NAME`, `GIT_COMMITTER_EMAIL`, `GIT_AUTHOR_DATE`, `GIT_COMMITTER_DATE`, or `EMAIL` in the finalizer environment cannot change the candidate envelope;
5. a finalizer deliberately constructs the exact authorized tree/message but embeds private request bytes in author/committer metadata: audit fails;
6. unexpected `gpgsig`, `encoding`, merge/tag, extra-parent, or unknown commit headers fail audit unless explicitly allowed by the pinned envelope version;
7. the auditor independently derives the raw expected commit envelope and exact expected candidate object ID rather than trusting finalizer metadata;
8. exact expected replay/object ID equals exact candidate `C` for the hosted release implementing this ADR;
9. candidate with correct semantic tree but wrong commit metadata cannot reach `AUDITED`/`PUBLISHING`;
10. candidate package containing an extra unreachable blob/tree/commit cannot cause that extra object to be uploaded/imported by either the minimal Git publisher fallback or any accepted API publication path;
11. the publisher still changes only the exact bound ref from exact `H0` to exact audited `C` and never rewrites `C`;
12. ADR-020 semantic/index/trust/scope checks remain additive rather than being replaced by commit-SHA equality alone;
13. ADR-021 `NO_OP` / `ALREADY_COMMITTED` independent terminal verification remains unchanged; and
14. local/offline and hosted release compatibility tests document the deliberate candidate-commit metadata behavior change without changing canonical memory/schema/index semantics.