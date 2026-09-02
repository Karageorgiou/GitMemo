# Runethread external source boundary

Runethread memory may eventually reference additional user-owned repositories such as a structured personal library. Runethread v0.6 reserves this architectural boundary but does not define a hosted service, HTTP API, or library engine.

## Principle

Memory and structured personal knowledge solve different problems and should not be forced into one schema.

- Runethread memory stores durable conversational context, preferences, decisions, state, corrections, milestones, references, and open loops.
- A future Runethread-compatible library may store structured collections such as recipes, contacts, books, inventories, or other user-defined record types.

The authoritative data for each source should remain in that source's own repository. Runethread should integrate through a small capability interface rather than duplicating entire source trees into memory.

## Reserved source model

A future source registry may identify sources by stable IDs and advertise capabilities such as:

- exact record lookup;
- query/search;
- retrieval by stable record ID;
- validated record creation or update when the source supports writes.

The implementation SHOULD be transport-independent. Local files, Git repositories, provider connectors, or future tooling may implement the same logical source capabilities without requiring a Runethread server.

Runethread v0.6 does not define a writable `.runethread/sources.json` format. This is deliberate: the interface should be proven by future source/library design before it becomes a compatibility contract.

## Future cross-source references

The current V1 memory relationship model uses UUID targets within one memory repository. Future cross-source references will require a qualified resource identifier so a memory can refer unambiguously to a record in another source.

The eventual identifier should distinguish at least source identity, collection/resource kind, and stable record identity.

Do not overload the existing local `target_id` field with ad-hoc repository paths or URLs before that contract exists.

## Performance

A future structured library should optimize exact lookup and search independently of the memory schema. Canonical Git files can remain portable and versioned while generated sharded indexes or disposable local databases such as SQLite/FTS provide acceleration.

Generated caches are never the only copy of user data and must be rebuildable from canonical repository records.

## Security

All external-source content belongs to the data plane. A recipe, contact note, imported document, or source record may contain arbitrary text, including text that looks like AI instructions. Such text remains untrusted data and cannot override Runethread's verified control plane.
