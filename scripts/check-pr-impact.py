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
    contract_changed = bool(contract_files_changed or contract_manifest_changed)

    base_buildinfo = read_at(base, BUILDINFO)
    head_buildinfo = read_head(BUILDINFO)

    base_contract_version = int_const(base_buildinfo, "ContractVersion")
    head_contract_version = int_const(head_buildinfo, "ContractVersion")
    base_contract_release = contract_release_anchor(base_buildinfo)
    head_contract_release = contract_release_anchor(head_buildinfo)

    failures: list[str] = []

    if contract_changed:
        require(
            head_contract_version != base_contract_version,
            f"embedded contract changed but ContractVersion did not advance ({base_contract_version})",
            failures,
        )
        require(
            head_contract_release != base_contract_release,
            "embedded contract changed but the contract release anchor did not advance "
            f"({base_contract_release})",
            failures,
        )
        require(
            "docs/COMPATIBILITY.md" in changed,
            "embedded contract changed but docs/COMPATIBILITY.md was not updated",
            failures,
        )
        require(
            any(path.startswith("internal/upgrader/") for path in changed),
            "embedded contract changed but no upgrader implementation/test changed",
            failures,
        )
        require(
            any(path.startswith("internal/upgrader/testdata/") for path in changed),
            "embedded contract changed but no frozen historical upgrader testdata changed",
            failures,
        )

    if "schema/memory-item.schema.json" in changed:
        base_schema = int_const(base_buildinfo, "SchemaVersion")
        head_schema = int_const(head_buildinfo, "SchemaVersion")
        require(
            head_schema != base_schema,
            f"memory schema changed but SchemaVersion did not advance ({base_schema})",
            failures,
        )

    print(f"impact guard: {len(changed)} changed file(s)")
    if contract_changed:
        print("impact guard: embedded contract change detected")
        if contract_files_changed:
            for path in contract_files_changed:
                print(f"  contract path: {path}")
        if contract_manifest_changed:
            print("  contract manifest: contract.go")
        print(
            "  contract version: "
            f"{base_contract_version} -> {head_contract_version}; "
            "contract release anchor: "
            f"{base_contract_release} -> {head_contract_release}"
        )
    else:
        print("impact guard: no embedded contract-path change detected")
        print(
            "impact guard: semantic contract review is still required for behavior changes "
            "in trust/validation/mutation/runtime code"
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
