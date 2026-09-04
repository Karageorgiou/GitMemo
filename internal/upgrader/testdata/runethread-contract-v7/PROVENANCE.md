# Native contract-v7 historical fixture provenance

These files freeze trusted native Runethread source state for upgrader regression tests. They are evidence, not generated target output.

## v0.6.0 source

- source repository: `runethread/memory-template`
- signed source commit: `5d5d3da9f2e8f1827402da634128c4e493bed408`
- commit purpose: exact validated native template produced by the GitMemo v0.5.0 -> Runethread v0.6.0 cutover
- config Git blob: `baf9c8d023f92926d10877c7e2772b378fbe3de2`
- lock Git blob: `ac3eac82643828ea304fffdca8bd87291ce09079`

## v0.7.0 source

- source repository: `runethread/memory-template`
- signed source commit: `4376d9263c0c31d2376308d68ad37931c071898a`
- parent: exact v0.6.0 template commit above
- commit purpose: repin the exact trusted v0.6.0 native state to immutable Runethread v0.7.0
- config Git blob: `93388236849c4d1035f074911129a1206dbad286`
- lock Git blob: `91723b5995545005ce112f7be5b0239d8941db56`
- validation workflow run on that exact commit: GitHub Actions run `33811990421`, conclusion `success`

## Shared contract-v7 identity

Both native source releases carry:

```text
repository_format  2
schema_version      1
contract_version    7
lock_version        2
contract_sha256     5b245c55e640555c797c3f86f02b54a431da40e959bdd466f90c0c5c88c45766
control-plane files 19
```

The v0.6.0 and v0.7.0 locks contain the same 19 per-file SHA-256 values. Their native contract bytes are therefore identical; only the release pin differs.

The shared bootstrap workflow is frozen as Git blob `983939d66d840c4825af6dfd60865580b5242294`, matching both template states.

## Contract files frozen here

Only contract-v7 files whose bytes intentionally change in contract v8 are duplicated under this fixture:

- `MEMORY_PROTOCOL.md` — source blob `e6c02219d9818cf06c2a435aa19311d172032d4d`
- `docs/TRUST_MODEL.md` — source blob `0f20c82f941ae4cd454f8077de9c5b5e85754e78`
- `docs/REPOSITORY_VALIDATION.md` — source blob `ef7599d03178028a4ff9598e2b7d06d61a00b8a6`
- `docs/INDEX_FORMAT.md` — source blob `e2e5560bc64a1802d4eae6441e249decf9f303f5`

For the other 15 contract paths, fixture materialization may reuse the currently embedded byte only after computing SHA-256 and proving it exactly equals the corresponding historical lock entry. A mismatch must fail the test and require that historical file to be frozen explicitly. This prevents a future generator from silently manufacturing historical source state.
