#!/usr/bin/env python3
"""Fail closed when Runethread's mandatory development safety surface drifts."""

from __future__ import annotations

import re
import sys
from pathlib import Path

SHA40 = re.compile(r"^[0-9a-f]{40}$")
USES_RE = re.compile(r"(?m)^\s*uses:\s*([^\s#]+)")

REQUIRED_FILES = (
    ".gitattributes",
    "AGENTS.md",
    "docs/runethread/ENGINEERING_PROCESS.md",
    "docs/runethread/DEVELOPMENT_PIPELINE.md",
    "docs/runethread/CURRENT_MILESTONE.md",
    ".github/pull_request_template.md",
    ".github/workflows/validate.yml",
    ".github/workflows/release.yml",
    ".github/dependabot.yml",
    ".github/CODEOWNERS",
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
    "DEVELOPMENT_PIPELINE.md",
    "CURRENT_MILESTONE.md",
    "Validation is read-only",
    "Fail closed",
    "Respect the process/product scope boundary",
    "Do not hide platform defects",
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

PIPELINE_NEEDLES = (
    "Mandatory semantic scope boundary",
    "Cheap deterministic gates",
    "Linux deterministic quality gate",
    "Cross-platform gate",
    "CI self-protection and supply-chain baseline",
    "Draft PR gate",
    "Merge and post-merge gate",
    "Mandatory future-agent behavior",
    "Cross-platform failures MUST NOT",
    "contents: read",
    "gofmt -l",
    "git diff --check",
    "go test -race -count=1 ./...",
    "macOS through the platform matrix",
    "Windows through the platform matrix",
)

PR_NEEDLES = (
    "Development infrastructure / CI / engineering policy",
    "Scope-boundary decision",
    "Mandatory pipeline on exact head",
    "No platform was removed/skipped/weakened to obtain green CI",
    "required final `validate` job passed on the exact reviewed head",
)

DEPENDABOT_NEEDLES = (
    'package-ecosystem: "gomod"',
    'package-ecosystem: "github-actions"',
)

CODEOWNERS_NEEDLES = (
    "/AGENTS.md @Karageorgiou",
    "/docs/runethread/ENGINEERING_PROCESS.md @Karageorgiou",
    "/.github/workflows/ @Karageorgiou",
    "/internal/trust/ @Karageorgiou",
    "/internal/upgrader/ @Karageorgiou",
)

GITATTRIBUTES_NEEDLES = (
    "* text=auto eol=lf",
    "*.exe binary",
)


def read(root: Path, rel: str, errors: list[str]) -> str:
    path = root / rel
    try:
        return path.read_text(encoding="utf-8")
    except OSError as exc:
        errors.append(f"{rel}: cannot read required file: {exc}")
        return ""


def check_action_pins(rel: str, text: str, errors: list[str]) -> None:
    uses = USES_RE.findall(text)
    if not uses:
        errors.append(f"{rel}: no actions are declared")
        return

    seen_checkout = False
    seen_setup_go = False
    for target in uses:
        if target.startswith("./"):
            continue
        if "@" not in target:
            errors.append(f"{rel}: external action {target!r} must be pinned with @<40-hex-sha>")
            continue
        action, ref = target.rsplit("@", 1)
        if action == "actions/checkout":
            seen_checkout = True
        if action == "actions/setup-go":
            seen_setup_go = True
        if not SHA40.fullmatch(ref):
            errors.append(f"{rel}: external action {action} must use an immutable 40-hex commit SHA, got {ref!r}")

    if not seen_checkout:
        errors.append(f"{rel}: missing actions/checkout usage")
    if not seen_setup_go:
        errors.append(f"{rel}: missing actions/setup-go usage")


def require_needles(label: str, text: str, needles: tuple[str, ...], errors: list[str]) -> None:
    for needle in needles:
        if needle not in text:
            errors.append(f"{label}: missing mandatory policy marker {needle!r}")


def check(root: Path) -> list[str]:
    errors: list[str] = []
    for rel in REQUIRED_FILES:
        if not (root / rel).is_file():
            errors.append(f"{rel}: required development-safety file is missing")

    gitattributes = read(root, ".gitattributes", errors)
    validate = read(root, ".github/workflows/validate.yml", errors)
    release = read(root, ".github/workflows/release.yml", errors)
    dependabot = read(root, ".github/dependabot.yml", errors)
    codeowners = read(root, ".github/CODEOWNERS", errors)
    pr_template = read(root, ".github/pull_request_template.md", errors)
    agents = read(root, "AGENTS.md", errors)
    process = read(root, "docs/runethread/ENGINEERING_PROCESS.md", errors)
    pipeline = read(root, "docs/runethread/DEVELOPMENT_PIPELINE.md", errors)

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

    require_needles(".gitattributes", gitattributes, GITATTRIBUTES_NEEDLES, errors)
    require_needles("dependabot.yml", dependabot, DEPENDABOT_NEEDLES, errors)
    require_needles("CODEOWNERS", codeowners, CODEOWNERS_NEEDLES, errors)
    require_needles("AGENTS.md", agents, AGENT_NEEDLES, errors)
    require_needles("ENGINEERING_PROCESS.md", process, PROCESS_NEEDLES, errors)
    require_needles("DEVELOPMENT_PIPELINE.md", pipeline, PIPELINE_NEEDLES, errors)
    require_needles("pull_request_template.md", pr_template, PR_NEEDLES, errors)

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
