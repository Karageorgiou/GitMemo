package upgrader

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/runethread/core/internal/buildinfo"
	"github.com/runethread/core/internal/indexer"
	"github.com/runethread/core/internal/starter"
	"github.com/runethread/core/internal/trust"
)

func TestContractV8MigratesFrozenNativeSourcesWithoutChangingCanonicalData(t *testing.T) {
	if buildinfo.ContractVersion < 8 {
		t.Skip("contract-v8 migration proof activates with contract version 8")
	}

	for _, release := range []string{nativeV060ReleaseVersion, nativeV070ReleaseVersion} {
		t.Run(release, func(t *testing.T) {
			root := materializeContractV7Fixture(t, release)
			memoryJSON, memoryMD := writeFixtureMemory(t, root, false)
			projectOverview := filepath.Join(root, "projects", "test", "overview.md")
			if err := indexer.Write(root); err != nil {
				t.Fatal(err)
			}

			beforeJSON := mustRead(t, memoryJSON)
			beforeMD := mustRead(t, memoryMD)
			beforeProject := mustRead(t, projectOverview)
			beforeCatalog := mustRead(t, filepath.Join(root, "index", "catalog.json"))

			result, err := Apply(root)
			if err != nil {
				t.Fatal(err)
			}
			if result.FromVersion != release || result.ToVersion != buildinfo.ContractReleaseVersion || result.FromContract != 7 || result.ToContract != 8 {
				t.Fatalf("unexpected contract-v8 migration result: %#v", result)
			}
			if result.AlreadyCurrent {
				t.Fatal("historical native source incorrectly reported as current")
			}

			wantChanged := []string{
				".gitattributes",
				".runethread/config.json",
				".runethread/lock.json",
				"MEMORY_PROTOCOL.md",
				"docs/INDEX_FORMAT.md",
				"docs/REPOSITORY_VALIDATION.md",
				"docs/TRUST_MODEL.md",
			}
			gotChanged := append([]string(nil), result.ChangedPaths...)
			sort.Strings(gotChanged)
			if !reflect.DeepEqual(gotChanged, wantChanged) {
				t.Fatalf("contract-v8 changed paths = %v, want %v", gotChanged, wantChanged)
			}

			if got := mustRead(t, memoryJSON); !bytes.Equal(got, beforeJSON) {
				t.Fatal("canonical memory JSON changed during contract migration")
			}
			if got := mustRead(t, memoryMD); !bytes.Equal(got, beforeMD) {
				t.Fatal("canonical memory Markdown changed during contract migration")
			}
			if got := mustRead(t, projectOverview); !bytes.Equal(got, beforeProject) {
				t.Fatal("user-owned project bytes changed during contract migration")
			}
			if got := mustRead(t, filepath.Join(root, "index", "catalog.json")); !bytes.Equal(got, beforeCatalog) {
				t.Fatal("index catalog changed despite unchanged index format and canonical data")
			}
			if got := mustRead(t, filepath.Join(root, managedGitAttributesPath)); !bytes.Equal(got, starter.GitAttributes()) {
				t.Fatalf("managed .gitattributes bytes = %q", got)
			}
			if problems := trust.Check(root); len(problems) != 0 {
				t.Fatalf("target trust check failed: %+v", problems)
			}
			if stale, err := indexer.Check(root); err != nil || len(stale) != 0 {
				t.Fatalf("target index is not fresh: stale=%v err=%v", stale, err)
			}
			if _, err := os.Stat(filepath.Join(root, ".runethread", "config.json")); err != nil {
				t.Fatalf("target config missing: %v", err)
			}
		})
	}
}
