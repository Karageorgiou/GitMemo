package trust

import (
	"os"
	"path/filepath"
	"testing"

	gitmemo "github.com/Karageorgiou/GitMemo"
)

func TestExpectedLockCoversContractPaths(t *testing.T) {
	lock, err := ExpectedLock()
	if err != nil {
		t.Fatal(err)
	}
	if lock.GitMemoVersion != "v0.3.0" || lock.ContractVersion != 5 || lock.LockVersion != 1 {
		t.Fatalf("unexpected lock metadata: %#v", lock)
	}
	if len(lock.FilesSHA256) != len(gitmemo.ContractPaths()) {
		t.Fatalf("lock covers %d paths, contract has %d", len(lock.FilesSHA256), len(gitmemo.ContractPaths()))
	}
	if len(lock.ContractSHA256) != 64 {
		t.Fatalf("unexpected aggregate digest %q", lock.ContractSHA256)
	}
}

func TestReadPinnedVersionRejectsPreTrustRelease(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".gitmemo"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".gitmemo", "lock.json"), []byte(`{"lock_version":1,"gitmemo_version":"v0.2.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadPinnedVersion(root); err == nil {
		t.Fatal("expected pre-v0.3 pinned version to be rejected by stable bootstrap")
	}
}
