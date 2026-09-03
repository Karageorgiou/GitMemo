from pathlib import Path


def replace_exact(path: str, old: str, new: str, count: int) -> None:
    p = Path(path)
    text = p.read_text()
    actual = text.count(old)
    if actual != count:
        raise SystemExit(f"{path}: expected {count} occurrences, found {actual}: {old!r}")
    p.write_text(text.replace(old, new))


replace_exact(
    "internal/starter/init_test.go",
    "buildinfo.ReleaseVersion",
    "buildinfo.ContractReleaseVersion",
    1,
)
replace_exact(
    "internal/trust/lock_test.go",
    "buildinfo.ReleaseVersion",
    "buildinfo.ContractReleaseVersion",
    1,
)
replace_exact(
    "internal/upgrader/upgrader_test.go",
    "buildinfo.ReleaseVersion",
    "buildinfo.ContractReleaseVersion",
    5,
)
