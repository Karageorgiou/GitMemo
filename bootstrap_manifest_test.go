package runethread

import (
	"encoding/json"
	"net/url"
	"os"
	"slices"
	"testing"
)

type bootstrapManifest struct {
	BootstrapProtocol   int    `json:"bootstrap_protocol"`
	Project             string `json:"project"`
	CanonicalRepository string `json:"canonical_repository"`
	SetupInstructions   string `json:"setup_instructions"`
	Template            struct {
		Repository                  string `json:"repository"`
		DefaultMemoryRepositoryName string `json:"default_memory_repository_name"`
		RequiredVisibility          string `json:"required_visibility"`
		CreateURL                   string `json:"create_url"`
	} `json:"template"`
	Discovery struct {
		RepositoryNameHint string   `json:"repository_name_hint"`
		RequiredPaths      []string `json:"required_paths"`
	} `json:"discovery"`
	Release struct {
		Repository                    string `json:"repository"`
		Strategy                      string `json:"strategy"`
		TemplateIsReleasePinned       bool   `json:"template_is_release_pinned"`
		ExistingRepositoriesExplicit bool   `json:"existing_repositories_upgrade_explicitly"`
	} `json:"release"`
	Commands struct {
		Store  string `json:"store"`
		Search string `json:"search"`
	} `json:"commands"`
	Security struct {
		PersonalDataCanonical bool `json:"personal_data_allowed_in_canonical_repository"`
		PersonalDataTemplate  bool `json:"personal_data_allowed_in_template_repository"`
		PrivateBeforeWrites    bool `json:"memory_repository_must_be_private_before_personal_writes"`
		RequestCredentials     bool `json:"request_credentials"`
	} `json:"security"`
}

func TestBootstrapManifest(t *testing.T) {
	data, err := os.ReadFile("runethread-bootstrap.json")
	if err != nil {
		t.Fatalf("read bootstrap manifest: %v", err)
	}
	var manifest bootstrapManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parse bootstrap manifest: %v", err)
	}

	if manifest.BootstrapProtocol != 1 {
		t.Fatalf("bootstrap_protocol = %d, want 1", manifest.BootstrapProtocol)
	}
	if manifest.Project != "Runethread" {
		t.Fatalf("project = %q", manifest.Project)
	}
	if manifest.CanonicalRepository != "runethread/core" {
		t.Fatalf("canonical_repository = %q", manifest.CanonicalRepository)
	}
	if manifest.SetupInstructions != "AI_SETUP.md" {
		t.Fatalf("setup_instructions = %q", manifest.SetupInstructions)
	}
	if _, err := os.Stat(manifest.SetupInstructions); err != nil {
		t.Fatalf("setup instructions path is not present: %v", err)
	}
	if _, err := os.Stat("gitmemo-bootstrap.json"); !os.IsNotExist(err) {
		t.Fatalf("legacy bootstrap manifest must not remain a native current file: %v", err)
	}

	if manifest.Template.Repository != "runethread/memory-template" {
		t.Fatalf("template repository = %q", manifest.Template.Repository)
	}
	if manifest.Template.DefaultMemoryRepositoryName != "runethread-memory" {
		t.Fatalf("default memory repository name = %q", manifest.Template.DefaultMemoryRepositoryName)
	}
	if manifest.Template.RequiredVisibility != "private" {
		t.Fatalf("required visibility = %q, want private", manifest.Template.RequiredVisibility)
	}
	if manifest.Release.Repository != manifest.CanonicalRepository {
		t.Fatalf("release repository %q differs from canonical repository %q", manifest.Release.Repository, manifest.CanonicalRepository)
	}
	if manifest.Release.Strategy != "latest-stable-official-release" {
		t.Fatalf("release strategy = %q", manifest.Release.Strategy)
	}
	if !manifest.Release.TemplateIsReleasePinned {
		t.Fatal("template_is_release_pinned must be true")
	}
	if !manifest.Release.ExistingRepositoriesExplicit {
		t.Fatal("existing_repositories_upgrade_explicitly must be true")
	}

	for _, required := range []string{
		".runethread/config.json",
		".runethread/lock.json",
		"MEMORY_PROTOCOL.md",
		"memories",
		"projects",
	} {
		if !slices.Contains(manifest.Discovery.RequiredPaths, required) {
			t.Errorf("discovery required_paths missing %q", required)
		}
	}

	if manifest.Commands.Store != "Runethread: store ..." || manifest.Commands.Search != "Runethread: search ..." {
		t.Fatalf("unexpected command contract: store=%q search=%q", manifest.Commands.Store, manifest.Commands.Search)
	}
	if manifest.Security.PersonalDataCanonical || manifest.Security.PersonalDataTemplate {
		t.Fatal("public repositories must not allow personal memory data")
	}
	if !manifest.Security.PrivateBeforeWrites {
		t.Fatal("private-before-personal-writes gate must be true")
	}
	if manifest.Security.RequestCredentials {
		t.Fatal("setup must never request pasted credentials")
	}

	createURL, err := url.Parse(manifest.Template.CreateURL)
	if err != nil {
		t.Fatalf("parse template create_url: %v", err)
	}
	if createURL.Scheme != "https" || createURL.Host != "github.com" || createURL.Path != "/new" {
		t.Fatalf("unexpected template create URL destination: %s", createURL.String())
	}
	query := createURL.Query()
	wantQuery := map[string]string{
		"owner":          "@me",
		"name":           "runethread-memory",
		"visibility":     "private",
		"template_owner": "runethread",
		"template_name":  "memory-template",
	}
	for key, want := range wantQuery {
		if got := query.Get(key); got != want {
			t.Errorf("create_url query %s = %q, want %q", key, got, want)
		}
	}
}
