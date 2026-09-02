# ADR-003: Two-phase memory mutation protocol

Status: **Accepted**
Date: 2026-09-02

## Context

A memory write is not just a file append. New information may overlap with, correct, supersede, resolve, or duplicate existing memory. The semantic decision needs current related state, while the actual repository mutation must remain deterministic and safe.

## Decision

Memory mutation uses two explicit phases.

### Phase 1 — Prepare

`PrepareMutation`:

1. captures the current canonical repository revision;
2. retrieves relevant/overlapping memories and relationships;
3. returns legal semantic operation classes and structured context;
4. performs no canonical write.

### Phase 2 — Apply

After the user/AI chooses a semantic operation, `ApplyMutation`:

1. requires the prepared expected revision;
2. validates the requested operation;
3. stages canonical changes;
4. updates lifecycle/relationships as required;
5. regenerates affected derived indexes;
6. runs hard validation;
7. commits/pushes only verified state according to policy;
8. returns a structured verified result.

Each apply operation also carries a stable idempotency identity/key. `ApplyMutation` must be idempotent for retries of the same operation. If an earlier attempt successfully committed but its response was lost, repeating that same apply must return the original verified outcome (for example, `already_applied`) without creating another memory, another mutation, or another canonical commit.

This retry rule is an exception to treating the post-commit revision advance as an ordinary stale preparation: an exact retry of an operation already known to have committed is recognized as already applied. If the idempotency identity has not already committed, the normal expected-revision check still applies and stale preparations must fail and be re-prepared.

A no-op is a first-class outcome. Preparing does not imply that a write must occur.

## Consequences

- Semantic decisions are based on explicit current state instead of hidden repository assumptions.
- Concurrent changes can invalidate a prepared decision before any write occurs.
- Clients must retain a short-lived preparation token/result or expected revision.
- Apply callers must preserve the operation's idempotency identity across retries.
- Lost responses can be retried safely without duplicating canonical memories, relationships, mutations, or commits.
- Core must distinguish an exact retry of an already committed operation from a new stale operation.
- The protocol creates a natural approval point for sensitive mutations.

## Alternatives considered

### One-shot `store(text)`
Rejected because retrieval, semantic choice, concurrency checks, and mutation become entangled and difficult to inspect or retry safely.

### Direct file mutation followed by repair
Rejected because invalid intermediate state can escape and rollback becomes harder.

## Verification

1. `PrepareMutation` is read-only.
2. prepared output includes the canonical expected revision.
3. stale preparations are rejected at apply time unless the request is an exact retry of the same operation already committed under its idempotency identity.
4. no-op operations produce no canonical change.
5. hard validation failure cannot yield a successful commit/result.
6. retrying the same successfully committed apply after a simulated lost response returns the original verified outcome and creates no duplicate memory, relationship, mutation, or commit.
7. tests cover create, correction, supersession, resolution, withdrawal, no-op, rollback, stale preparation, and committed-apply retry paths.
