#!/usr/bin/env python3
"""Fail closed when Runethread's mandatory development safety surface drifts."""

from __future__ import annotations

import re
import sys
from pathlib import Path

SHA40 = re.compile(r"^[0-9a-f]{40}$")

REQUIRED_FILES = (
    "AGENTS.md",
    "docs/runethread/ENGINEERING_PROCESS.md",
    "docs/runethread/CURRENT_MILESTONE.md",
    ".github/pull_request_template.md",
    ".github/workflows/validate.yml",
    ".github/workflows/release.yml",
    "scripts/check-pr-impact.py",
    "scripts/check_pr_impact_test.py",
    "scripts/check_development_policy.py",
    "scripts/check_development_policy_test.py",
)

VALIDATE_NEEDLES = (
    "permissions:\n  contents: read",
    "go mod verify",
    "gofmt -l",
    "git diff --check",
    "python3 scripts/check_development_policy_test.py",
    "python3 scripts/check_development_policy.py",
    "python3 scripts/check_pr_impact_test.py",
    "python3 scripts/check-pr-impact.py",
    "go test -count=1 ./...",
    "go test -race -count=1 ./...",
    "go vet ./...",
    "go build ./cmd/runethread",
    "runethread index --check",
    "runethread validate",
    "macos-latest",
    "windows-latest",
    "needs: [quality, platform]",
)

AGENT_NEEDLES = (
    "MUST",
    "ENGINEERING_PROCESS.md",
    "CURRENT_MILESTONE.md",
    "Validation is read-only",
    "Fail closed",
)

PROCESS_NEEDLES = (
    "Preflight gate",
    "Impact matrix",
    "Historical / backward compatibility gate",
    "Forward compatibility gate",
    "Negative and failure-mode gate",
    "Verification gate on the committed branch",
    "Draft PR review gate",
    "Post-merge gate",
    "Correction / incident protocol",
    "Stop conditions",
)


def read(root: Path, rel: str, errors: list[str]) -> str:
    path = root / rel
    try:
        return path.read_text(encoding="utf-8")
    except OSError as exc:
        errors.append(f"{rel}: cannot read required file: {exc}")
        return ""


def check_action_pins(rel: str, text: str, errors: list[str]) -> None:
    for action in ("actions/checkout", "actions/setup-go"):
        matches = re.findall(rf"uses:\s*{re.escape(action)}@([^\s#]+)", text)
        if not matches:
            errors.append(f"{rel}: missing {action} usage")
            continue
        for ref in matches:
            if not SHA40.fullmatch(ref):
                errors.append(f"{rel}: {action} must use an immutable 40-hex commit SHA, got {ref!r}")


def check(root: Path) -> list[str]:
    errors: list[str] = []
    for rel in REQUIRED_FILES:
        if not (root / rel).is_file():
            errors.append(f"{rel}: required development-safety file is missing")

    validate = read(root, ".github/workflows/validate.yml", errors)
    release = read(root, ".github/workflows/release.yml", errors)
    agents = read(root, "AGENTS.md", errors)
    process = read(root, "docs/runethread/ENGINEERING_PROCESS.md", errors)

    if "pull_request_target:" in validate:
        errors.append("validate.yml: pull_request_target is forbidden for validation CI")
    if re.search(r"(?m)^\s*contents:\s*write\s*$", validate):
        errors.append("validate.yml: validation CI must not have contents: write")
    for needle in VALIDATE_NEEDLES:
        if needle not in validate:
            errors.append(f"validate.yml: missing mandatory safety surface {needle!r}")

    check_action_pins(".github/workflows/validate.yml", validate, errors)
    check_action_pins(".github/workflows/release.yml", release, errors)

    if not re.search(r"(?m)^\s*contents:\s*write\s*$", release):
        errors.append("release.yml: release publication requires explicit contents: write")

    for needle in AGENT_NEEDLES:
        if needle not in agents:
            errors.append(f"AGENTS.md: missing mandatory policy marker {needle!r}")
    for needle in PROCESS_NEEDLES:
        if needle not in process:
            errors.append(f"ENGINEERING_PROCESS.md: missing mandatory process section {needle!r}")

    return errors


def main(argv: list[str]) -> int:
    root = Path(argv[1]).resolve() if len(argv) > 1 else Path(__file__).resolve().parents[1]
    errors = check(root)
    if errors:
        for error in errors:
            print(f"development-policy: {error}", file=sys.stderr)
        return 1
    print("development-policy: OK")
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv))
