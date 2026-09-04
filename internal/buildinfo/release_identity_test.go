package buildinfo_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/runethread/core/internal/starter"
	"github.com/runethread/core/internal/trust"
)

func TestRepositoryPinGeneratorsUseContractReleaseIdentity(t *testing.T) {
	for _, rel := range []string{
		filepath.Join("..", "starter", "init.go"),
		filepath.Join("..", "trust", "lock.go"),
	} {
		data, err := os.ReadFile(rel)
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		text := string(data)
		if strings.Contains(text, "RunethreadVersion: buildinfo.ReleaseVersion") {
			t.Fatalf("%s couples repository pinning directly to runtime ReleaseVersion", rel)
		}
		if !strings.Contains(text, "RunethreadVersion: buildinfo.ContractReleaseVersion") {
			t.Fatalf("%s does not pin repositories through ContractReleaseVersion", rel)
		}
	}
}

func TestIdentitySplitPreservesExactPublishedV07MetadataBytes(t *testing.T) {
	fixtureRoot := filepath.Join("..", "upgrader", "testdata", "runethread-v0.7.0", ".runethread")

	wantConfig, err := os.ReadFile(filepath.Join(fixtureRoot, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	gotConfig, err := starter.ConfigJSON()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotConfig, wantConfig) {
		t.Fatalf("config bytes changed during identity-only split\nwant: %s\ngot:  %s", wantConfig, gotConfig)
	}

	wantLock, err := os.ReadFile(filepath.Join(fixtureRoot, "lock.json"))
	if err != nil {
		t.Fatal(err)
	}
	gotLock, err := trust.JSON()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotLock, wantLock) {
		t.Fatalf("lock bytes changed during identity-only split")
	}
}
