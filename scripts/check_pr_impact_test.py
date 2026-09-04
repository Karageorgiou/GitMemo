#!/usr/bin/env python3

from __future__ import annotations

import shutil
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path


SCRIPT = Path(__file__).with_name("check-pr-impact.py")


class ImpactGuardIntegrationTests(unittest.TestCase):
    def setUp(self) -> None:
        self.tempdir = tempfile.TemporaryDirectory()
        self.root = Path(self.tempdir.name)
        (self.root / "scripts").mkdir(parents=True)
        shutil.copy2(SCRIPT, self.root / "scripts" / SCRIPT.name)
        self.git("init", "-b", "main")
        self.git("config", "user.name", "Runethread Impact Test")
        self.git("config", "user.email", "impact-test@example.invalid")

    def tearDown(self) -> None:
        self.tempdir.cleanup()

    def git(self, *args: str) -> str:
        result = subprocess.run(
            ["git", *args],
            cwd=self.root,
            check=True,
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
        )
        return result.stdout.strip()

    def write(self, path: str, content: str) -> None:
        target = self.root / path
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_text(content, encoding="utf-8")

    def commit(self, message: str) -> str:
        self.git("add", "-A")
        self.git("commit", "-m", message)
        return self.git("rev-parse", "HEAD")

    def buildinfo_text(
        self,
        *,
        release: str = "v0.7.0",
        contract_release: str | None = None,
        repository_format: int = 2,
        schema_version: int = 1,
        contract_version: int = 7,
        index_format: int = 2,
        trust_lock: int = 2,
    ) -> str:
        contract_release_line = ""
        if contract_release is not None:
            contract_release_line = f'\tContractReleaseVersion = "{contract_release}"\n'
        return (
            "package buildinfo\n\nconst (\n"
            f'\tReleaseVersion = "{release}"\n'
            f"{contract_release_line}"
            f"\tRepositoryFormatVersion = {repository_format}\n"
            f"\tSchemaVersion = {schema_version}\n"
            f"\tContractVersion = {contract_version}\n"
            f"\tIndexFormatVersion = {index_format}\n"
            f"\tTrustLockVersion = {trust_lock}\n"
            ")\n"
        )

    def initialize_repository(
        self,
        *,
        release: str = "v0.7.0",
        contract_release: str | None = None,
        contract_version: int = 7,
        schema_version: int = 1,
    ) -> str:
        self.write(
            "internal/buildinfo/version.go",
            self.buildinfo_text(
                release=release,
                contract_release=contract_release,
                contract_version=contract_version,
                schema_version=schema_version,
            ),
        )
        self.write(
            "contract.go",
            "package runethread\n\n"
            "var contractPaths = []string{\n"
            '\t"MEMORY_PROTOCOL.md",\n'
            '\t"schema/memory-item.schema.json",\n'
            "}\n",
        )
        self.write("MEMORY_PROTOCOL.md", "contract v1\n")
        self.write("schema/memory-item.schema.json", '{"schema_version":1}\n')
        self.write("docs/COMPATIBILITY.md", "compatibility baseline\n")
        self.write("internal/upgrader/upgrader.go", "package upgrader\n")
        self.write("internal/upgrader/testdata/baseline/README.txt", "fixture\n")
        return self.commit("baseline")

    def write_migration_evidence(self) -> None:
        self.write("docs/COMPATIBILITY.md", "updated compatibility\n")
        self.write("internal/upgrader/upgrader.go", "package upgrader\n// migration update\n")
        self.write("internal/upgrader/testdata/runethread-v0.7.0/README.txt", "exact fixture marker\n")

    def run_guard(self, base: str) -> subprocess.CompletedProcess[str]:
        return subprocess.run(
            [sys.executable, "scripts/check-pr-impact.py", base],
            cwd=self.root,
            check=False,
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
        )

    def test_non_contract_change_passes(self) -> None:
        base = self.initialize_repository()
        self.write("README.md", "non-contract documentation\n")
        self.commit("docs")

        result = self.run_guard(base)

        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn("no embedded contract-path change detected", result.stdout)

    def test_contract_change_without_version_and_migration_fails(self) -> None:
        base = self.initialize_repository()
        self.write("MEMORY_PROTOCOL.md", "changed contract semantics\n")
        self.commit("bad contract change")

        result = self.run_guard(base)

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("ContractVersion did not advance", result.stderr)
        self.assertIn("contract release anchor did not advance", result.stderr)
        self.assertIn("docs/COMPATIBILITY.md was not updated", result.stderr)
        self.assertIn("no upgrader implementation/test changed", result.stderr)
        self.assertIn("no frozen historical upgrader testdata changed", result.stderr)

    def test_properly_versioned_contract_migration_passes(self) -> None:
        base = self.initialize_repository()
        self.write(
            "internal/buildinfo/version.go",
            self.buildinfo_text(
                release="v0.8.0",
                contract_release="v0.8.0",
                contract_version=8,
            ),
        )
        self.write("MEMORY_PROTOCOL.md", "contract v8 semantics\n")
        self.write_migration_evidence()
        self.commit("contract migration")

        result = self.run_guard(base)

        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn("embedded contract change detected", result.stdout)
        self.assertIn("ContractVersion: 7 -> 8", result.stdout)
        self.assertIn("contract release anchor: v0.7.0 -> v0.8.0", result.stdout)

    def test_testdata_alone_does_not_count_as_upgrader_change(self) -> None:
        base = self.initialize_repository()
        self.write(
            "internal/buildinfo/version.go",
            self.buildinfo_text(
                release="v0.8.0",
                contract_release="v0.8.0",
                contract_version=8,
            ),
        )
        self.write("MEMORY_PROTOCOL.md", "contract v8 semantics\n")
        self.write("docs/COMPATIBILITY.md", "updated compatibility\n")
        self.write("internal/upgrader/testdata/runethread-v0.7.0/README.txt", "fixture only\n")
        self.commit("incomplete migration")

        result = self.run_guard(base)

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("no upgrader implementation/test changed", result.stderr)

    def test_runtime_only_release_after_split_does_not_require_contract_bump(self) -> None:
        base = self.initialize_repository(
            release="v0.8.0",
            contract_release="v0.8.0",
            contract_version=8,
        )
        self.write(
            "internal/buildinfo/version.go",
            self.buildinfo_text(
                release="v0.9.0",
                contract_release="v0.8.0",
                contract_version=8,
            ),
        )
        self.commit("runtime-only release")

        result = self.run_guard(base)

        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn("no embedded contract-path change detected", result.stdout)

    def test_contract_anchor_change_requires_migration_evidence(self) -> None:
        base = self.initialize_repository()
        self.write(
            "internal/buildinfo/version.go",
            self.buildinfo_text(release="v0.8.0"),
        )
        self.commit("unmigrated contract reanchor")

        result = self.run_guard(base)

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("docs/COMPATIBILITY.md was not updated", result.stderr)
        self.assertIn("no upgrader implementation/test changed", result.stderr)
        self.assertIn("no frozen historical upgrader testdata changed", result.stderr)

    def test_contract_version_cannot_advance_without_contract_bytes(self) -> None:
        base = self.initialize_repository()
        self.write(
            "internal/buildinfo/version.go",
            self.buildinfo_text(
                release="v0.8.0",
                contract_release="v0.8.0",
                contract_version=8,
            ),
        )
        self.write_migration_evidence()
        self.commit("version-only contract bump")

        result = self.run_guard(base)

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("ContractVersion changed but no embedded contract path", result.stderr)

    def test_schema_change_without_schema_version_bump_fails(self) -> None:
        base = self.initialize_repository()
        self.write("schema/memory-item.schema.json", '{"schema_version":1,"changed":true}\n')
        self.commit("bad schema change")

        result = self.run_guard(base)

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("SchemaVersion did not advance", result.stderr)

    def test_schema_version_cannot_advance_without_schema_bytes(self) -> None:
        base = self.initialize_repository()
        self.write(
            "internal/buildinfo/version.go",
            self.buildinfo_text(schema_version=2),
        )
        self.write_migration_evidence()
        self.commit("version-only schema bump")

        result = self.run_guard(base)

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("SchemaVersion changed but schema/memory-item.schema.json did not change", result.stderr)

    def test_repository_format_change_requires_contract_bytes(self) -> None:
        base = self.initialize_repository()
        self.write(
            "internal/buildinfo/version.go",
            self.buildinfo_text(repository_format=3),
        )
        self.write_migration_evidence()
        self.commit("version-only repository format bump")

        result = self.run_guard(base)

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("RepositoryFormatVersion changed but no embedded contract path", result.stderr)

    def test_contract_manifest_change_is_treated_as_contract_change(self) -> None:
        base = self.initialize_repository()
        self.write(
            "contract.go",
            "package runethread\n\n"
            "var contractPaths = []string{\n"
            '\t"MEMORY_PROTOCOL.md",\n'
            '\t"schema/memory-item.schema.json",\n'
            '\t"docs/NEW_CONTRACT_FILE.md",\n'
            "}\n",
        )
        self.write("docs/NEW_CONTRACT_FILE.md", "new contract surface\n")
        self.commit("bad manifest change")

        result = self.run_guard(base)

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("embedded contract change detected", result.stdout)
        self.assertIn("ContractVersion did not advance", result.stderr)


if __name__ == "__main__":
    unittest.main()
