package trust

import (
	"testing"

	"github.com/runethread/core/internal/buildinfo"
)

func TestExpectedLockPinsContractRelease(t *testing.T) {
	lock, err := ExpectedLock()
	if err != nil {
		t.Fatal(err)
	}
	if lock.RunethreadVersion != buildinfo.ContractReleaseVersion {
		t.Fatalf("lock pins %q, want contract release %q", lock.RunethreadVersion, buildinfo.ContractReleaseVersion)
	}
}
