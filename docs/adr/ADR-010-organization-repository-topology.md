# ADR-010: Runethread organization and repository topology

Status: **Accepted**
Date: 2026-09-02

## Context

Runethread is broader than the original GitMemo repository and needs clear public component boundaries without unnecessary repository sprawl. User-owned memory must also remain distinct from project-owned software repositories.

## Decision

The public GitHub organization is `runethread`.

Initial public repository topology:

```text
runethread/
  core
  orchestrator
  memory-template
  .github
```

Responsibilities:

- `core` — deterministic memory engine, CLI, validator, trust/compatibility logic, schemas/contracts, and Core adapters;
- `orchestrator` — tasks, workers, routing, approvals, worktrees, runtime persistence, and orchestration interfaces;
- `memory-template` — generated public setup artifact from official Core releases, never an independently authored semantic contract;
- `.github` — organization profile and shared organization-level community/configuration material where appropriate.

User-owned memory repositories remain under the user's own GitHub account/organization, for example:

```text
<user>/runethread-memory
```

They must not be moved under the Runethread software organization merely for convenience.

New repositories are created only when a component has a genuinely independent responsibility/release lifecycle. SDKs, docs, website, examples, protocol packages, plugins, and cloud services remain inside the most appropriate existing repository until a concrete reason justifies a split.

## Consequences

- The organization communicates the architecture clearly without becoming a collection of tiny repositories.
- Users retain ownership/control of private memory data.
- Core and Orchestrator can release independently.
- Shared organization infrastructure remains intentionally small initially.

## Alternatives considered

### Monorepo for Core and Orchestrator
Rejected because their trust surfaces, dependencies, release cadence, and operational responsibilities differ materially.

### Many repositories from day one
Rejected because speculative splitting creates navigation, CI, versioning, and maintenance overhead before boundaries are proven.

### Host user memory repositories under `runethread`
Rejected because it weakens the user-owned-data model and creates unnecessary central control.

## Verification

1. initial public software is limited to the agreed repositories unless a later ADR/change explains a split;
2. Core does not absorb Orchestrator provider/runtime dependencies;
3. generated `memory-template` content is reproducible from an official Core release;
4. private user memory remains user-owned outside the Runethread organization.
