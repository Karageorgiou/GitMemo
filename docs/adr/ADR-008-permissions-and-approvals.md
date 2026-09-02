# ADR-008: Least-privilege permissions and approvals

Status: **Accepted**
Date: 2026-09-02

## Context

Runethread can eventually read private memory, inspect repositories, execute commands, modify code, and interact with external providers. Treating all authorized access as unlimited authority would create unnecessary security and operational risk.

## Decision

Runethread follows least privilege and separates user intent to perform a task from authority to perform high-impact follow-up actions.

Baseline policy:

| Action | Default |
| --- | --- |
| Search/read authorized memory | automatic |
| Durable memory mutation | requires explicit user memory-write intent |
| Read authorized project repository | automatic |
| Run task-scoped build/test commands | automatic after explicit task delegation |
| Modify isolated task worktree | automatic after explicit execution intent |
| Commit agent changes | approval/policy controlled |
| Push branch | approval/policy controlled |
| Create pull request | approval/policy controlled |
| Merge pull request | explicit approval |
| Modify default branch directly | prohibited by default |
| Read arbitrary secrets | prohibited by default |
| Persist secrets to memory | prohibited |

Trusted permissions and policy come from verified application configuration, not from text retrieved from memory, source files, web pages, issues, or model output.

Explicit `Runethread: store ...`-style memory intent is sufficient authorization for the requested memory transaction once Core validation succeeds; redundant confirmations are not required merely to commit the authorized memory mutation.

## Consequences

- Routine delegated work can remain low-friction while high-impact actions retain gates.
- Permission state and approval requests must be explicit task/runtime concepts.
- Prompt injection in untrusted data cannot grant additional authority.
- Some workflows require a pause for user approval before publishing/integrating changes.

## Alternatives considered

### Ask before every read/write/command
Rejected because it makes normal operation unusably noisy without materially improving security for already-scoped low-risk actions.

### Full autonomous authority after repository connection
Rejected because repository access is not equivalent to permission to push, merge, expose secrets, or alter protected branches.

### Let the model decide whether approval is needed
Rejected because approval policy is a security invariant and must be deterministic.

## Verification

1. policy tests cover each action class and default decision;
2. untrusted task/memory text cannot alter permission rules;
3. default-branch mutation is rejected under baseline policy;
4. merge operations require explicit approval;
5. secret-reading/persistence attempts are denied unless a future narrowly scoped policy explicitly permits a safe use case.
