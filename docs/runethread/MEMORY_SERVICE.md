# MemoryService

Status: **Implemented in Runethread v0.7.0 and current in v0.8.0**

MemoryService is the deterministic application boundary over a Runethread memory repository. It exists so AI clients and adapters can retrieve and mutate memory without editing canonical Markdown, JSON, generated indexes, or Git state directly.

## Operations

```text
Search
Get
PrepareMutation
ApplyMutation
Withdraw
Status
```

`PrepareMutation` is read-only. It retrieves candidate memories, reports legal operation classes, and returns the exact Git revision that the semantic decision was prepared against.

`ApplyMutation` accepts a structured operation, a stable idempotency key, that expected revision, an RFC3339 mutation time, and where required a proposed memory plus Markdown. Core derives canonical paths and owns the storage transaction.

Supported operation classes are `create`, `update`, `correct`, `supersede`, `resolve`, `withdraw`, and `noop`.

## Write transaction

A non-noop apply follows this deterministic sequence:

```text
idempotency lookup
   |
   +-- exact committed retry -> already_applied
   +-- same key / different request -> idempotency_conflict
   |
verify clean canonical checkout + expected Git revision
   |
detached temporary Git worktree at expected revision
   |
apply semantic transition
   |
rebuild Index v2
   |
hard repository validation
   |
create mutation commit with Runethread operation metadata
   |
fast-forward publish only if canonical branch is still expected revision
```

Validation failure discards the isolated transaction and cannot publish a canonical commit. In-process writes through one MemoryService instance are serialized. Git revision compare-and-swap remains the correctness boundary against other processes or writers.

## Idempotency and lost responses

Every real mutation carries `idempotency_key`. The request is fingerprinted deterministically and the key/fingerprint are stored in Git commit metadata.

If a commit succeeds but the caller loses the response, retrying the exact same request/key returns `already_applied` with the original commit. This lookup happens before ordinary stale-revision rejection because the successful first attempt necessarily advanced HEAD.

Reusing a committed key with a different request returns `idempotency_conflict`.

## CLI JSON boundary

The CLI service commands always emit JSON; `--json` is accepted explicitly for automation/readability.

```text
runethread get --root DIR --json <uuid>
runethread prepare --root DIR --json --request request.json
runethread apply --root DIR --json --request request.json
runethread withdraw --root DIR --json --request request.json
runethread status --root DIR --json
```

For `prepare`, `apply`, and `withdraw`, `--request -` reads one JSON value from stdin. Unknown JSON fields and trailing JSON values are rejected.

Successful results are written to stdout. Structured errors are written to stderr:

```json
{
  "error": {
    "code": "stale_revision",
    "operation": "apply",
    "message": "repository revision is ..."
  }
}
```

Malformed command/request input exits with code 2. MemoryService/repository failures exit with code 1.

## Apply request shape

```json
{
  "expected_revision": "<git-commit-sha>",
  "idempotency_key": "stable-operation-id",
  "operation": "create",
  "mutation_time": "2026-09-03T12:00:00Z",
  "proposed": {
    "memory": { "...": "schema-valid memory sidecar fields" },
    "markdown": "# Canonical memory content\n..."
  }
}
```

Operations that modify an existing memory also use `target_id`. `withdraw` has a narrower request containing `expected_revision`, `idempotency_key`, `target_id`, and `mutation_time`.

The semantic caller chooses the intended operation and proposed content. Deterministic Core enforces lifecycle/relationship rules, canonical placement, indexing, validation, idempotency, and Git publication.

## Release compatibility

Runethread v0.8.0 keeps the MemoryService operation model and repository format stable while introducing contract version 8 and explicit runtime-release / contract-release separation.

Current compatibility dimensions are:

```text
runtime release        v0.8.0
contract release       v0.8.0
repository format      2
memory schema          1
contract version       8
Index format           2
trust-lock version     2
bootstrap protocol     1
bootstrap verifier     v0.6.0
```

Under contract v8, `.runethread/config.json` and `.runethread/lock.json` `runethread_version` identify the immutable **contract release** rather than necessarily the runtime executable version. `runethread status` reports runtime and contract release identities separately. A future runtime-only release may operate against an unchanged v0.8 contract repository when the embedded contract identity/dimensions/digests match; that does not require repository churn.

The v0.8.0 CLI includes rollback-safe migration from the exact trusted native v0.6.0 and v0.7.0 contract-v7 source anchors, while retaining the deliberately narrow exact GitMemo v0.5.0 predecessor bridge. Supported migration updates managed contract/config/lock state, adds the managed root `.gitattributes` support file when appropriate, rebuilds/validates derived state, and preserves canonical memory/project bytes where the transition does not require a representation change. Unknown, mixed, tampered, newer-unknown, customized, or unsafe authoritative source states fail closed rather than being guessed or repaired.

The MemoryService remains transport-independent. The Phase 3 MCP adapter is expected to expose these same application operations rather than duplicate memory business logic in the transport layer.
