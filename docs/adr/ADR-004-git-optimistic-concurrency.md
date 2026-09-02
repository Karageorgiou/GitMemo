# ADR-004: Git-revision optimistic concurrency

Status: **Accepted**
Date: 2026-09-02

## Context

Multiple AI clients or processes may read the same memory repository and attempt writes independently. A local mutex cannot protect against another machine, process, or remote agent updating the repository between read and write.

## Decision

Runethread Core uses optimistic concurrency anchored in canonical Git revisions.

- Every prepared mutation records `expected_revision`.
- Apply must compare the current canonical revision with `expected_revision` before writing.
- If they differ, the mutation is rejected as stale.
- The caller must prepare again and re-evaluate semantic intent against current state.
- Local locking may serialize writes within one process/machine, but it does not replace revision checking.
- Silent overwrite/last-writer-wins behavior is prohibited for canonical memory mutations.

## Consequences

- Independent clients can safely share a repository without a central lock server.
- Some operations may need to be retried after legitimate concurrent writes.
- Error contracts must distinguish stale state from schema/validation failure.
- Git itself remains part of the transaction identity.

## Alternatives considered

### Last writer wins
Rejected because it can silently discard newer canonical state.

### Central distributed lock service
Rejected for the local-first baseline because it creates an availability dependency and unnecessary infrastructure.

### Local file locks only
Rejected because they do not coordinate independent machines/clients.

## Verification

1. concurrent-write tests prepare two mutations from the same revision;
2. after one mutation commits, the second stale apply is rejected;
3. no stale apply modifies canonical files;
4. re-prepare against the new revision can subsequently succeed;
5. local locking does not bypass the Git revision check.
