# ADR-007: Isolated Git worktrees for delegated code mutation

Status: **Accepted**
Date: 2026-09-02

## Context

Coding agents need repository write access, shell execution, and build/test capability. Allowing an agent to modify the user's normal checkout creates avoidable risk: uncommitted work can be overwritten, concurrent tasks can interfere, and rollback/review becomes harder.

## Decision

Every delegated code-modifying task uses an isolated Git worktree and task branch by default.

Target convention:

```text
normal checkout:  ~/dev/<project>
agent worktree:   ~/.runethread/worktrees/<project>/<task-id>/
branch:           runethread/task-<id>-<slug>
```

The worker receives the isolated worktree as its repository root. Direct mutation of the user's ordinary checkout/default branch is prohibited by default.

Task results must report at least changed files, commands executed, build/test results, worktree/branch identifiers, unresolved issues, and incomplete verification.

## Consequences

- User work is isolated from agent changes.
- Parallel coding tasks can be separated cleanly.
- Review, cancellation, and cleanup have concrete filesystem/Git boundaries.
- Worktree lifecycle/garbage collection becomes an Orchestrator responsibility.

## Alternatives considered

### Modify the normal checkout
Rejected because it mixes human and agent state and weakens rollback/review.

### Clone the whole repository per task
Rejected as the default because Git worktrees preserve isolation with less storage and faster setup.

### Containerize every coding task immediately
Rejected as a baseline requirement because worktrees solve source isolation without introducing container infrastructure; stronger sandboxing may be added independently when needed.

## Verification

1. coding worker integration tests run against temporary worktrees;
2. the normal checkout remains byte/Git-state unchanged after delegated edits;
3. two concurrent tasks use distinct branches/worktrees;
4. cancellation/cleanup cannot delete the user's normal checkout;
5. default-branch mutation requires an explicit different policy path.
