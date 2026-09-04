package memoryservice

import (
	"context"
	"testing"

	"github.com/runethread/core/internal/buildinfo"
	"github.com/runethread/core/internal/repository"
)

func TestStatusSeparatesRuntimeAndContractReleaseIdentity(t *testing.T) {
	root := makeServiceFixture(t)
	svc := New(fakeRepository{root: root, state: repository.State{Revision: "abc", Branch: "main", Clean: true}})

	got, err := svc.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.ReleaseVersion != buildinfo.ReleaseVersion {
		t.Fatalf("release_version = %q, want runtime %q", got.ReleaseVersion, buildinfo.ReleaseVersion)
	}
	if got.ContractReleaseVersion != buildinfo.ContractReleaseVersion {
		t.Fatalf("contract_release_version = %q, want contract release %q", got.ContractReleaseVersion, buildinfo.ContractReleaseVersion)
	}
}
