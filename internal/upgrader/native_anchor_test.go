package upgrader

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	runethread "github.com/runethread/core"
	"github.com/runethread/core/internal/trust"
)

func TestNativeSourceAnchorsMatchFrozenPublishedMetadata(t *testing.T) {
	for _, release := range []string{nativeV060ReleaseVersion, nativeV070ReleaseVersion} {
		t.Run(release, func(t *testing.T) {
			lockPath := filepath.Join("testdata", "runethread-"+release, ".runethread", "lock.json")
			data, err := os.ReadFile(lockPath)
			if err != nil {
				t.Fatal(err)
			}
			var locked trust.Lock
			if err := json.Unmarshal(data, &locked); err != nil {
				t.Fatal(err)
			}
			anchor, ok := nativeSourceAnchorFor(release)
			if !ok {
				t.Fatalf("missing source anchor for %s", release)
			}
			if locked.RunethreadVersion != anchor.ReleaseVersion ||
				locked.RepositoryFormat != anchor.RepositoryFormat ||
				locked.SchemaVersion != anchor.SchemaVersion ||
				locked.ContractVersion != anchor.ContractVersion ||
				locked.LockVersion != anchor.LockVersion ||
				locked.SourceRepository != anchor.SourceRepository ||
				locked.ContractSHA256 != anchor.ContractSHA256 {
				t.Fatalf("frozen %s metadata does not match source anchor: lock=%#v anchor=%#v", release, locked, anchor)
			}
			if !reflect.DeepEqual(locked.FilesSHA256, anchor.FilesSHA256) {
				t.Fatalf("frozen %s per-file digests do not match source anchor", release)
			}
			if got := legacyAggregateDigest(anchor.FilesSHA256); got != anchor.ContractSHA256 {
				t.Fatalf("source anchor aggregate digest = %s, want %s", got, anchor.ContractSHA256)
			}
		})
	}
}

func TestInspectSourceAcceptsExactFrozenNativeV060(t *testing.T) {
	root := materializeContractV7Fixture(t, nativeV060ReleaseVersion)
	state, err := inspectSource(root)
	if err != nil {
		t.Fatal(err)
	}
	if state.Version != nativeV060ReleaseVersion || state.ContractVersion != 7 || state.Kind != sourceNative {
		t.Fatalf("unexpected source state: %#v", state)
	}
}

func TestInspectSourceAcceptsExactFrozenNativeV070(t *testing.T) {
	root := materializeContractV7Fixture(t, nativeV070ReleaseVersion)
	state, err := inspectSource(root)
	if err != nil {
		t.Fatal(err)
	}
	if state.Version != nativeV070ReleaseVersion || state.ContractVersion != 7 || state.Kind != sourceNative {
		t.Fatalf("unexpected source state: %#v", state)
	}
}

func TestInspectSourceRefusesTamperedFrozenNativeV060(t *testing.T) {
	root := materializeContractV7Fixture(t, nativeV060ReleaseVersion)
	configBefore := mustRead(t, filepath.Join(root, ".runethread", "config.json"))
	lockBefore := mustRead(t, filepath.Join(root, ".runethread", "lock.json"))
	path := filepath.Join(root, "docs", "TRUST_MODEL.md")
	mustWrite(t, path, append(mustRead(t, path), []byte("\ntampered\n")...))

	if _, err := inspectSource(root); err == nil || !strings.Contains(err.Error(), "digest") {
		t.Fatalf("expected trusted source digest refusal, got %v", err)
	}
	if got := mustRead(t, filepath.Join(root, ".runethread", "config.json")); !reflect.DeepEqual(got, configBefore) {
		t.Fatal("config changed during read-only source verification")
	}
	if got := mustRead(t, filepath.Join(root, ".runethread", "lock.json")); !reflect.DeepEqual(got, lockBefore) {
		t.Fatal("lock changed during read-only source verification")
	}
}

func materializeContractV7Fixture(t *testing.T, release string) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "memory")
	fixtureRoot := filepath.Join("testdata", "runethread-"+release)

	for _, rel := range []string{
		".runethread/config.json",
		".runethread/lock.json",
		".github/workflows/validate.yml",
	} {
		data, err := os.ReadFile(filepath.Join(fixtureRoot, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("read frozen %s %s: %v", release, rel, err)
		}
		mustWrite(t, filepath.Join(root, filepath.FromSlash(rel)), data)
	}

	var locked trust.Lock
	lockData := mustRead(t, filepath.Join(root, ".runethread", "lock.json"))
	if err := json.Unmarshal(lockData, &locked); err != nil {
		t.Fatal(err)
	}
	if len(locked.FilesSHA256) != 19 {
		t.Fatalf("frozen %s lock contains %d contract paths, want 19", release, len(locked.FilesSHA256))
	}

	for _, rel := range sortedStringMapKeys(locked.FilesSHA256) {
		frozen := filepath.Join("testdata", "runethread-contract-v7", filepath.FromSlash(rel))
		data, err := os.ReadFile(frozen)
		if os.IsNotExist(err) {
			data, err = fs.ReadFile(runethread.ContractFS, rel)
		}
		if err != nil {
			t.Fatalf("materialize historical contract %s: %v", rel, err)
		}
		if got := sha256Hex(data); got != locked.FilesSHA256[rel] {
			t.Fatalf("historical contract %s digest = %s, want %s; current bytes may not be reused for this path", rel, got, locked.FilesSHA256[rel])
		}
		mustWrite(t, filepath.Join(root, filepath.FromSlash(rel)), data)
	}

	if err := os.MkdirAll(filepath.Join(root, "memories"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "projects"), 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}
