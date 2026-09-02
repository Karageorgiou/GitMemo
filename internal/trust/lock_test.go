package trust

import (
	"os"
	"path/filepath"
	"testing"

	runethread "github.com/runethread/core"
	"github.com/runethread/core/internal/buildinfo"
)

func TestExpectedLockCoversContractPaths(t *testing.T) {
	lock, err := ExpectedLock()
	if err != nil {
		t.Fatal(err)
	}
	if lock.RunethreadVersion != buildinfo.ReleaseVersion || lock.SourceRepository != buildinfo.SourceRepository || lock.ContractVersion != buildinfo.ContractVersion || lock.LockVersion != buildinfo.TrustLockVersion {
		t.Fatalf("unexpected lock metadata: %#v", lock)
	}
	if len(lock.FilesSHA256) != len(runethread.ContractPaths()) {
		t.Fatalf("lock covers %d paths, contract has %d", len(lock.FilesSHA256), len(runethread.ContractPaths()))
	}
	if len(lock.ContractSHA256) != 64 {
		t.Fatalf("unexpected aggregate digest %q", lock.ContractSHA256)
	}
}

func TestReadPinnedVersionReadsNativeRelease(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".runethread"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".runethread", "lock.json"), []byte(`{"lock_version":2,"runethread_version":"v0.6.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := ReadPinnedVersion(root)
	if err != nil {
		t.Fatal(err)
	}
	if got != "v0.6.0" {
		t.Fatalf("pinned version = %q", got)
	}
}

func TestReadPinnedVersionRejectsPreRunethreadRelease(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".runethread"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".runethread", "lock.json"), []byte(`{"lock_version":2,"runethread_version":"v0.5.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadPinnedVersion(root); err == nil {
		t.Fatal("expected pre-v0.6 pinned version to be rejected by Runethread bootstrap")
	}
}
