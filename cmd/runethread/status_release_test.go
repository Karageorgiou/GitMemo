package main

import (
	"testing"

	"github.com/runethread/core/internal/buildinfo"
	"github.com/runethread/core/internal/memoryservice"
)

func TestStatusCLIReportsRuntimeAndContractReleaseSeparately(t *testing.T) {
	root, _ := makeCLIServiceRepo(t)
	code, stdout, stderr := runCLIWithCapturedServiceIO(t, []string{"status", "--json", "--root", root})
	if code != 0 {
		t.Fatalf("status exit=%d stderr=%s", code, stderr)
	}
	var status memoryservice.StatusResponse
	decodeCLIOutput(t, stdout, &status)
	if status.ReleaseVersion != buildinfo.ReleaseVersion {
		t.Fatalf("release_version = %q, want runtime %q", status.ReleaseVersion, buildinfo.ReleaseVersion)
	}
	if status.ContractReleaseVersion != buildinfo.ContractReleaseVersion {
		t.Fatalf("contract_release_version = %q, want contract release %q", status.ContractReleaseVersion, buildinfo.ContractReleaseVersion)
	}
}
