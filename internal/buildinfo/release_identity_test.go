package buildinfo_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
