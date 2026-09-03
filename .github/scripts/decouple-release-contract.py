from pathlib import Path


def replace_exact(path: str, old: str, new: str, count: int = 1) -> None:
    p = Path(path)
    text = p.read_text()
    actual = text.count(old)
    if actual != count:
        raise SystemExit(f"{path}: expected {count} occurrences, found {actual}: {old!r}")
    p.write_text(text.replace(old, new))


replace_exact(
    "internal/buildinfo/version.go",
    '\tReleaseVersion           = "v0.7.0"\n\tRepositoryFormatVersion  = 2',
    '\t// ReleaseVersion identifies the running product/binary release.\n\tReleaseVersion = "v0.7.0"\n\t// ContractReleaseVersion identifies the immutable official release that owns\n\t// the embedded memory-repository control plane. Advance it only when that\n\t// control plane changes; executable-only releases keep the existing anchor.\n\tContractReleaseVersion  = "v0.7.0"\n\tRepositoryFormatVersion = 2',
)

replace_exact(
    "internal/trust/lock.go",
    "\t\tRunethreadVersion: buildinfo.ReleaseVersion,",
    "\t\tRunethreadVersion: buildinfo.ContractReleaseVersion,",
)
replace_exact(
    "internal/trust/lock.go",
    "// ReadPinnedVersion is intentionally forward-tolerant. The stable validation\n// bootstrap only needs the lock envelope version and pinned Runethread release.",
    "// ReadPinnedVersion is intentionally forward-tolerant. The stable validation\n// bootstrap only needs the lock envelope version and pinned control-plane release.",
)
replace_exact(
    "internal/trust/lock.go",
    '\tif actual.RunethreadVersion != expected.RunethreadVersion {\n\t\taddLock(fmt.Sprintf("runethread_version is %q, but running validator is %q", actual.RunethreadVersion, expected.RunethreadVersion))\n\t}',
    '\tif actual.RunethreadVersion != expected.RunethreadVersion {\n\t\taddLock(fmt.Sprintf("runethread_version is %q, expected pinned contract release %q (running release %q)", actual.RunethreadVersion, expected.RunethreadVersion, buildinfo.ReleaseVersion))\n\t}',
)

replace_exact(
    "internal/starter/init.go",
    "\t\tRunethreadVersion: buildinfo.ReleaseVersion,",
    "\t\tRunethreadVersion: buildinfo.ContractReleaseVersion,",
)

replace_exact(
    "internal/upgrader/native_previous.go",
    "\tcase buildinfo.ReleaseVersion:",
    "\tcase buildinfo.ContractReleaseVersion:",
)
replace_exact(
    "internal/upgrader/native_previous.go",
    'return fmt.Errorf("embedded %s contract digest %s differs from the trusted %s compatible digest %s; explicit contract migration is required", buildinfo.ReleaseVersion, expected.ContractSHA256, previousNativeReleaseVersion, previousNativeContractSHA256)',
    'return fmt.Errorf("embedded contract release %s digest %s differs from the trusted %s compatible digest %s; explicit contract migration is required", buildinfo.ContractReleaseVersion, expected.ContractSHA256, previousNativeReleaseVersion, previousNativeContractSHA256)',
)

replace_exact(
    "internal/upgrader/upgrader.go",
    "\t\tToVersion:      buildinfo.ReleaseVersion,",
    "\t\tToVersion:      buildinfo.ContractReleaseVersion,",
)
old_compat = '''func checkNativeCompatibility(cfg repositoryConfig) error {
\tif cfg.RepositoryFormat != buildinfo.RepositoryFormatVersion {
\t\treturn fmt.Errorf("repository format %d is not supported by %s (supports %d)", cfg.RepositoryFormat, buildinfo.ReleaseVersion, buildinfo.RepositoryFormatVersion)
\t}
\tif cfg.SchemaVersion != buildinfo.SchemaVersion {
\t\tif cfg.SchemaVersion > buildinfo.SchemaVersion {
\t\t\treturn fmt.Errorf("repository schema version %d is newer than %s supports (%d)", cfg.SchemaVersion, buildinfo.ReleaseVersion, buildinfo.SchemaVersion)
\t\t}
\t\treturn fmt.Errorf("no Runethread schema migration from version %d to %d is implemented", cfg.SchemaVersion, buildinfo.SchemaVersion)
\t}
\tif cfg.ContractVersion != buildinfo.ContractVersion {
\t\tif cfg.ContractVersion > buildinfo.ContractVersion {
\t\t\treturn fmt.Errorf("repository contract version %d is newer than %s supports (%d)", cfg.ContractVersion, buildinfo.ReleaseVersion, buildinfo.ContractVersion)
\t\t}
\t\treturn fmt.Errorf("no native Runethread contract migration from version %d to %d is implemented", cfg.ContractVersion, buildinfo.ContractVersion)
\t}
\tif cfg.RunethreadVersion != buildinfo.ReleaseVersion && cfg.RunethreadVersion != previousNativeReleaseVersion {
\t\treturn fmt.Errorf("repository pins Runethread %q; %s only upgrades exact trusted %s or %s native state, or trusted GitMemo v0.5.0", cfg.RunethreadVersion, buildinfo.ReleaseVersion, previousNativeReleaseVersion, buildinfo.ReleaseVersion)
\t}
\treturn nil
}'''
new_compat = '''func checkNativeCompatibility(cfg repositoryConfig) error {
\treturn checkNativeCompatibilityFor(cfg, buildinfo.ReleaseVersion, buildinfo.ContractReleaseVersion)
}

func checkNativeCompatibilityFor(cfg repositoryConfig, runtimeRelease, contractRelease string) error {
\tif cfg.RepositoryFormat != buildinfo.RepositoryFormatVersion {
\t\treturn fmt.Errorf("repository format %d is not supported by %s (supports %d)", cfg.RepositoryFormat, runtimeRelease, buildinfo.RepositoryFormatVersion)
\t}
\tif cfg.SchemaVersion != buildinfo.SchemaVersion {
\t\tif cfg.SchemaVersion > buildinfo.SchemaVersion {
\t\t\treturn fmt.Errorf("repository schema version %d is newer than %s supports (%d)", cfg.SchemaVersion, runtimeRelease, buildinfo.SchemaVersion)
\t\t}
\t\treturn fmt.Errorf("no Runethread schema migration from version %d to %d is implemented", cfg.SchemaVersion, buildinfo.SchemaVersion)
\t}
\tif cfg.ContractVersion != buildinfo.ContractVersion {
\t\tif cfg.ContractVersion > buildinfo.ContractVersion {
\t\t\treturn fmt.Errorf("repository contract version %d is newer than %s supports (%d)", cfg.ContractVersion, runtimeRelease, buildinfo.ContractVersion)
\t\t}
\t\treturn fmt.Errorf("no native Runethread contract migration from version %d to %d is implemented", cfg.ContractVersion, buildinfo.ContractVersion)
\t}
\tif cfg.RunethreadVersion != contractRelease && cfg.RunethreadVersion != previousNativeReleaseVersion {
\t\treturn fmt.Errorf("repository pins contract release %q; running %s supports exact trusted %s or %s native contract state, or trusted GitMemo v0.5.0", cfg.RunethreadVersion, runtimeRelease, previousNativeReleaseVersion, contractRelease)
\t}
\treturn nil
}'''
replace_exact("internal/upgrader/upgrader.go", old_compat, new_compat)
replace_exact(
    "internal/upgrader/upgrader_test.go",
    "\t\tRunethreadVersion: buildinfo.ReleaseVersion,",
    "\t\tRunethreadVersion: buildinfo.ContractReleaseVersion,",
)

replace_exact(
    "docs/runethread/MEMORY_SERVICE.md",
    '''## Release compatibility

v0.7.0 does not change repository format, memory schema, operational contract, Index v2, or trust-lock format. Those remain 2 / 1 / 7 / 2 / 2 respectively.

Because repositories intentionally pin the exact trusted Runethread release, the v0.7.0 CLI includes an explicit rollback-safe upgrade from the exact trusted v0.6.0 native state. That migration repins managed config/trust metadata, rebuilds indexes, validates, and never rewrites canonical memories or project data.
''',
    '''## Release compatibility

v0.7.0 does not change repository format, memory schema, operational contract, Index v2, or trust-lock format. Those remain 2 / 1 / 7 / 2 / 2 respectively.

A memory repository pins the immutable **contract release** that owns its embedded control plane, not necessarily the version of the executable currently operating on it. A newer Runethread runtime may operate on the existing pin without repository churn only when its `ContractReleaseVersion`, compatibility dimensions, aggregate contract digest, and per-file control-plane digests still match that pinned contract exactly. Executable-only releases therefore do not require a memory-repository repin.

When the control plane itself changes, `ContractReleaseVersion` must advance and the repository moves through an explicit supported upgrade. The v0.7.0 contract release includes the rollback-safe upgrade from the exact trusted v0.6.0 native state; that migration repins managed config/trust metadata, rebuilds indexes, validates, and never rewrites canonical memories or project data.
''',
)

replace_exact("docs/adr/README.md", "## Phase 0 decision catalog", "## Decision catalog")
replace_exact(
    "docs/adr/README.md",
    "| [ADR-010](ADR-010-organization-repository-topology.md) | Runethread organization and repository topology | Accepted |",
    "| [ADR-010](ADR-010-organization-repository-topology.md) | Runethread organization and repository topology | Accepted |\n| [ADR-011](ADR-011-runtime-contract-release-separation.md) | Runtime releases and contract releases are separate version dimensions | Accepted |",
)

adr = Path("docs/adr/ADR-011-runtime-contract-release-separation.md")
if adr.exists():
    raise SystemExit(f"{adr} already exists")
adr.write_text('''# ADR-011: Separate runtime releases from contract releases

Status: **Accepted**
Date: 2026-09-04

## Context

Runethread memory repositories pin an immutable official release so their vendored operational contract can be verified without trusting mutable `main`. The initial native implementation used one version value, `ReleaseVersion`, both for the executable product release and for the repository's pinned control-plane authority.

That coupling is unnecessarily strong. Phase 2 demonstrated the problem: v0.7.0 added the transport-independent MemoryService without changing repository format 2, memory schema 1, contract 7, Index v2, trust-lock 2, or any vendored control-plane bytes, yet existing memory repositories still had to repin from v0.6.0 to v0.7.0 merely because the binary version advanced.

Future executable-only work such as the local MCP adapter must not rewrite trusted repository metadata when the memory contract has not changed.

## Decision

Runethread maintains two explicit release dimensions:

- `ReleaseVersion` identifies the running product/binary release.
- `ContractReleaseVersion` identifies the immutable official Runethread release that owns the embedded memory-repository control plane.

The existing `.runethread/config.json` and `.runethread/lock.json` field `runethread_version` remains backward-compatible and represents the pinned **contract release**. It is not a requirement that the running executable have that same release number.

A running Runethread release may operate on a repository pinned to an older contract release only when all of the following hold:

1. the running binary's `ContractReleaseVersion` equals the repository pin;
2. repository format, memory schema, operational contract, Index format, and trust-lock compatibility are supported;
3. the aggregate embedded contract digest matches the lock;
4. every locked control-plane file digest matches both the embedded contract and the vendored repository copy.

The running `ReleaseVersion` is never substituted for those checks.

If any vendored control-plane file or compatibility dimension changes, `ContractReleaseVersion` must advance and repositories require an explicit supported upgrade before using that new contract. A runtime-only release keeps the previous `ContractReleaseVersion` and must not repin otherwise-valid repositories.

The stable validation bootstrap continues to resolve `runethread_version` from the repository lock and install that immutable contract release for independent repository validation. That remains intentional even when a newer compatible runtime exists.

## Consequences

- MCP, Orchestrator-facing adapters, CLI UX, performance work, packaging, and other executable-only releases can ship without meaningless memory-repository commits.
- Trust remains release-anchored and digest-verified; no mutable `main`, remote lookup, compatibility range, or second trust root is introduced.
- The runtime version and repository contract pin may legitimately differ and diagnostics must describe them distinctly.
- New repository initialization pins `ContractReleaseVersion`, not `ReleaseVersion`.
- Repository upgrade results describe movement between contract releases rather than merely reporting the running binary version.
- A future real contract change still requires an explicit migration and new contract release anchor.

## Alternatives considered

### Keep exact runtime-release pinning

Rejected because executable-only releases create repository churn without increasing trust or compatibility guarantees.

### Replace the release pin with only `contract_version`

Rejected because the immutable release plus aggregate/per-file digests provides a concrete independently verifiable authority for the exact contract bytes. The integer contract version alone is insufficient.

### Add semantic-version ranges to the lock

Rejected because ranges weaken the simple immutable trust anchor and introduce compatibility policy into repository metadata. Runtime compatibility is better expressed by the running binary's embedded contract anchor and existing version/digest dimensions.

### Rename `runethread_version` immediately

Rejected for now because the existing field can carry the more precise contract-release meaning without a repository-format or lock-format migration. The semantic distinction is recorded here and in current documentation.

## Verification

Implementation and tests must demonstrate that:

- trust locks and starter configs are generated from `ContractReleaseVersion`;
- native compatibility accepts a repository pinned to contract release v0.7.0 when a simulated runtime release is v0.8.0 and the compatibility dimensions are unchanged;
- a repository pin equal only to the simulated runtime release is not accepted when the runtime's contract release remains v0.7.0;
- `upgrade` reports the target contract release, not merely `ReleaseVersion`;
- no contract file, repository format, schema, contract, Index format, or trust-lock version changes as part of this decoupling;
- a future runtime release can deliberately advance `ReleaseVersion` while leaving `ContractReleaseVersion` unchanged and still validate a repository pinned to that contract release.
''')

upgrader_test = Path("internal/upgrader/release_compatibility_test.go")
if upgrader_test.exists():
    raise SystemExit(f"{upgrader_test} already exists")
upgrader_test.write_text('''package upgrader

import (
\t"testing"

\t"github.com/runethread/core/internal/buildinfo"
)

func TestNativeCompatibilitySeparatesRuntimeAndContractRelease(t *testing.T) {
\tcfg := repositoryConfig{
\t\tRepositoryFormat:  buildinfo.RepositoryFormatVersion,
\t\tSchemaVersion:     buildinfo.SchemaVersion,
\t\tContractVersion:   buildinfo.ContractVersion,
\t\tRunethreadVersion: "v0.7.0",
\t}

\tif err := checkNativeCompatibilityFor(cfg, "v0.8.0", "v0.7.0"); err != nil {
\t\tt.Fatalf("compatible contract pin rejected by newer runtime: %v", err)
\t}

\tcfg.RunethreadVersion = "v0.8.0"
\tif err := checkNativeCompatibilityFor(cfg, "v0.8.0", "v0.7.0"); err == nil {
\t\tt.Fatal("runtime release was incorrectly accepted as the repository contract pin")
\t}
}
''')

trust_test = Path("internal/trust/release_compatibility_test.go")
if trust_test.exists():
    raise SystemExit(f"{trust_test} already exists")
trust_test.write_text('''package trust

import (
\t"testing"

\t"github.com/runethread/core/internal/buildinfo"
)

func TestExpectedLockPinsContractRelease(t *testing.T) {
\tlock, err := ExpectedLock()
\tif err != nil {
\t\tt.Fatal(err)
\t}
\tif lock.RunethreadVersion != buildinfo.ContractReleaseVersion {
\t\tt.Fatalf("lock pins %q, want contract release %q", lock.RunethreadVersion, buildinfo.ContractReleaseVersion)
\t}
}
''')
