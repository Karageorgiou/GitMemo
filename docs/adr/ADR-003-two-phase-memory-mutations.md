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

A no-op is a first-class outcome. Preparing does not imply that a write must occur.

## Consequences

- Semantic decisions are based on explicit current state instead of hidden repository assumptions.
- Concurrent changes can invalidate a prepared decision before any write occurs.
- Clients must retain a short-lived preparation token/result or expected revision.
- The protocol creates a natural approval point for sensitive mutations.

## Alternatives considered

### One-shot `store(text)`
Rejected because retrieval, semantic choice, concurrency checks, and mutation become entangled and difficult to inspect or retry safely.

### Direct file mutation followed by repair
Rejected because invalid intermediate state can escape and rollback becomes harder.

## Verification

1. `PrepareMutation` is read-only.
2. prepared output includes the canonical expected revision.
3. stale preparations are rejected at apply time.
4. no-op operations produce no canonical change.
5. hard validation failure cannot yield a successful commit/result.
6. tests cover create, correction, supersession, resolution, withdrawal, no-op, and rollback paths.
