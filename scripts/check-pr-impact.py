#!/usr/bin/env python3
"""Fail closed on high-risk PR versioning mistakes.

This guard intentionally covers only deterministic relationships that can be
proven from the Git diff. It does not replace the semantic impact review in
`docs/runethread/ENGINEERING_PROCESS.md`.
"""

from __future__ import annotations

import re
import subprocess
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
BUILDINFO = "internal/buildinfo/version.go"
CONTRACT_GO = "contract.go"
SCHEMA_PATH = "schema/memory-item.schema.json"
COMPATIBILITY_PATH = "docs/COMPATIBILITY.md"
UPGRADER_PREFIX = "internal/upgrader/"
UPGRADER_TESTDATA_PREFIX = "internal/upgrader/testdata/"


def git(*args: str) -> str:
    result = subprocess.run(
        ["git", *args],
        cwd=ROOT,
        check=True,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )
    return result.stdout


def read_at(ref: str, path: str) -> str:
    return git("show", f"{ref}:{path}")


def read_head(path: str) -> str:
    return (ROOT / path).read_text(encoding="utf-8")


def contract_paths(source: str) -> set[str]:
    match = re.search(
        r"var\s+contractPaths\s*=\s*\[\]string\s*\{(?P<body>.*?)\n\}",
        source,
        re.DOTALL,
    )
    if not match:
        raise SystemExit("could not parse contractPaths from contract.go")
    return set(re.findall(r'"([^"\\]+)"', match.group("body")))


def string_const(source: str, name: str) -> str | None:
    match = re.search(rf"(?m)^\s*{re.escape(name)}\s*=\s*\"([^\"]+)\"", source)
    return match.group(1) if match else None


def int_const(source: str, name: str) -> int:
    match = re.search(rf"(?m)^\s*{re.escape(name)}\s*=\s*([0-9]+)\b", source)
    if not match:
        raise SystemExit(f"could not parse {name} from {BUILDINFO}")
    return int(match.group(1))


def contract_release_anchor(source: str) -> str:
    # Before runtime/contract separation, ReleaseVersion is the contract anchor.
    # After separation, ContractReleaseVersion is authoritative for repository pins.
    value = string_const(source, "ContractReleaseVersion")
    if value is not None:
        return value
    value = string_const(source, "ReleaseVersion")
    if value is None:
        raise SystemExit(f"could not parse release anchor from {BUILDINFO}")
    return value


def require(condition: bool, message: str, failures: list[str]) -> None:
    if not condition:
        failures.append(message)


def main() -> int:
    if len(sys.argv) != 2:
        print("usage: check-pr-impact.py <base-sha>", file=sys.stderr)
        return 2

    base = sys.argv[1].strip()
    if not re.fullmatch(r"[0-9a-fA-F]{40}", base):
        print("base SHA must be a full 40-character Git SHA", file=sys.stderr)
        return 2

    changed = {
        line.strip()
        for line in git("diff", "--name-only", base, "HEAD").splitlines()
        if line.strip()
    }
    if not changed:
        print("impact guard: no changed files")
        return 0

    base_contract_source = read_at(base, CONTRACT_GO)
    head_contract_source = read_head(CONTRACT_GO)
    base_contract_paths = contract_paths(base_contract_source)
    head_contract_paths = contract_paths(head_contract_source)
    all_contract_paths = base_contract_paths | head_contract_paths

    contract_files_changed = sorted(changed & all_contract_paths)
    contract_manifest_changed = CONTRACT_GO in changed
    embedded_contract_changed = bool(contract_files_changed or contract_manifest_changed)

    base_buildinfo = read_at(base, BUILDINFO)
    head_buildinfo = read_head(BUILDINFO)

    base_contract_release = contract_release_anchor(base_buildinfo)
    head_contract_release = contract_release_anchor(head_buildinfo)
    contract_release_changed = base_contract_release != head_contract_release

    version_names = (
        "RepositoryFormatVersion",
        "SchemaVersion",
        "ContractVersion",
        "IndexFormatVersion",
        "TrustLockVersion",
    )
    base_versions = {name: int_const(base_buildinfo, name) for name in version_names}
    head_versions = {name: int_const(head_buildinfo, name) for name in version_names}
    changed_versions = {
        name for name in version_names if base_versions[name] != head_versions[name]
    }

    compatibility_updated = COMPATIBILITY_PATH in changed
    upgrader_code_or_test_changed = any(
        path.startswith(UPGRADER_PREFIX)
        and not path.startswith(UPGRADER_TESTDATA_PREFIX)
        for path in changed
    )
    historical_testdata_changed = any(
        path.startswith(UPGRADER_TESTDATA_PREFIX) for path in changed
    )

    failures: list[str] = []

    # Any change to the embedded operational contract must be explicitly
    # versioned and anchored to a new immutable contract release.
    if embedded_contract_changed:
        require(
            "ContractVersion" in changed_versions,
            "embedded contract changed but ContractVersion did not advance "
            f"({base_versions['ContractVersion']})",
            failures,
        )
        require(
            contract_release_changed,
            "embedded contract changed but the contract release anchor did not advance "
            f"({base_contract_release})",
            failures,
        )

    # A declared contract-version change without changed normative contract
    # bytes/manifest is also invalid: the version must describe real contract
    # semantics rather than becoming an arbitrary counter.
    if "ContractVersion" in changed_versions:
        require(
            embedded_contract_changed,
            "ContractVersion changed but no embedded contract path or contract manifest changed",
            failures,
        )
        require(
            contract_release_changed,
            "ContractVersion changed but the contract release anchor did not advance",
            failures,
        )

    # Schema version and schema bytes must move together. The schema is itself a
    # contract path, so a real schema change is also subject to the contract gate.
    schema_file_changed = SCHEMA_PATH in changed
    if schema_file_changed:
        require(
            "SchemaVersion" in changed_versions,
            "memory schema changed but SchemaVersion did not advance "
            f"({base_versions['SchemaVersion']})",
            failures,
        )
    if "SchemaVersion" in changed_versions:
        require(
            schema_file_changed,
            "SchemaVersion changed but schema/memory-item.schema.json did not change",
            failures,
        )

    # Repository/index/trust-lock format changes alter how the operational
    # contract is interpreted. Require corresponding normative contract bytes,
    # which in turn trigger the contract-version/release checks above.
    for name in ("RepositoryFormatVersion", "IndexFormatVersion", "TrustLockVersion"):
        if name in changed_versions:
            require(
                embedded_contract_changed,
                f"{name} changed but no embedded contract path or contract manifest changed",
                failures,
            )

    # Moving the contract release anchor, changing contract semantics, or
    # changing a compatibility dimension requires explicit compatibility and
    # migration evidence. This deliberately does not trigger for a future
    # runtime-only ReleaseVersion bump once ContractReleaseVersion is separate.
    migration_evidence_required = bool(
        embedded_contract_changed or contract_release_changed or changed_versions
    )
    if migration_evidence_required:
        require(
            compatibility_updated,
            "compatibility/version state changed but docs/COMPATIBILITY.md was not updated",
            failures,
        )
        require(
            upgrader_code_or_test_changed,
            "compatibility/version state changed but no upgrader implementation/test changed",
            failures,
        )
        require(
            historical_testdata_changed,
            "compatibility/version state changed but no frozen historical upgrader testdata changed",
            failures,
        )

    print(f"impact guard: {len(changed)} changed file(s)")
    if embedded_contract_changed:
        print("impact guard: embedded contract change detected")
        if contract_files_changed:
            for path in contract_files_changed:
                print(f"  contract path: {path}")
        if contract_manifest_changed:
            print("  contract manifest: contract.go")
    else:
        print("impact guard: no embedded contract-path change detected")
        print(
            "impact guard: semantic contract review is still required for behavior changes "
            "in trust/validation/mutation/runtime code"
        )

    if contract_release_changed:
        print(
            "impact guard: contract release anchor: "
            f"{base_contract_release} -> {head_contract_release}"
        )
    for name in version_names:
        if name in changed_versions:
            print(
                f"impact guard: {name}: "
                f"{base_versions[name]} -> {head_versions[name]}"
            )

    if failures:
        print("impact guard failed:", file=sys.stderr)
        for failure in failures:
            print(f"- {failure}", file=sys.stderr)
        return 1

    print("impact guard: passed")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
