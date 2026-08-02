package app

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/seongmin221/ai-account-manager/internal/config"
	"github.com/seongmin221/ai-account-manager/internal/credentials"
	"github.com/seongmin221/ai-account-manager/internal/execution"
)

func TestReadOnlyCLIInitializesAndReportsConfiguration(t *testing.T) {
	store := config.NewFileStore(t.TempDir() + "/config.toml")
	var output, errors bytes.Buffer
	cli := NewCLI(store, execution.NewFakeRunner(), &output, &errors)
	if got := cli.Run([]string{"init"}); got != 0 {
		t.Fatalf("init exit code = %d, errors = %s", got, errors.String())
	}
	if got := cli.Run([]string{"init"}); got != 0 {
		t.Fatalf("second init exit code = %d, errors = %s", got, errors.String())
	}
	if !strings.Contains(output.String(), "Initialized configuration") || !strings.Contains(output.String(), "already exists") {
		t.Fatalf("unexpected init output: %s", output.String())
	}
}

func TestReadOnlyCLICommandsUseSafeSummaries(t *testing.T) {
	store := config.NewFileStore(t.TempDir() + "/config.toml")
	document := config.Document{
		Version: 1,
		Active:  map[config.ProviderID]config.ProfileID{"github": "personal", "codex": "work"},
		Profiles: map[config.ProfileID]config.Profile{
			"work":     {Providers: map[config.ProviderID]config.ProviderConfig{"codex": {"credential_ref": "codex/work", "secret": "must-not-print"}}},
			"personal": {Providers: map[config.ProviderID]config.ProviderConfig{"github": {"host": "github.com"}}},
		},
	}
	if err := store.Save(document); err != nil {
		t.Fatal(err)
	}
	var output, errors bytes.Buffer
	runner := execution.NewFakeRunner()
	runner.Available["gh"] = true
	cli := NewCLI(store, runner, &output, &errors)
	if got := cli.Run([]string{"current"}); got != 0 {
		t.Fatalf("current exit code = %d, errors = %s", got, errors.String())
	}
	if got := cli.Run([]string{"list"}); got != 0 {
		t.Fatalf("list exit code = %d, errors = %s", got, errors.String())
	}
	if got := cli.Run([]string{"doctor"}); got != 0 {
		t.Fatalf("doctor exit code = %d, errors = %s", got, errors.String())
	}
	if strings.Contains(output.String(), "must-not-print") || !strings.Contains(output.String(), "Overall mode: mixed") || !strings.Contains(output.String(), "gh: available") {
		t.Fatalf("unsafe or incomplete output: %s", output.String())
	}
}

func TestReadOnlyCLIValidatesAndRejectsUnknownCommands(t *testing.T) {
	store := config.NewFileStore(t.TempDir() + "/config.toml")
	if err := store.Save(config.NewDocument()); err != nil {
		t.Fatal(err)
	}
	var output, errors bytes.Buffer
	cli := NewCLI(store, execution.NewFakeRunner(), &output, &errors)
	if got := cli.Run([]string{"config", "validate"}); got != 0 || !strings.Contains(output.String(), "Configuration is valid") {
		t.Fatalf("validate exit=%d output=%s errors=%s", got, output.String(), errors.String())
	}
	if got := cli.Run([]string{"unknown-command"}); got != 2 {
		t.Fatalf("unknown command exit code = %d, want 2", got)
	}
}

func TestChangeOnlyActivatesSelectedCodexProvider(t *testing.T) {
	root := t.TempDir()
	managedRoot := filepath.Join(root, "managed")
	codexHome := filepath.Join(managedRoot, "work")
	store := config.NewFileStore(filepath.Join(root, "config.toml"))
	document := config.Document{
		Version: 1,
		Active:  map[config.ProviderID]config.ProfileID{"github": "personal", "codex": "personal"},
		Profiles: map[config.ProfileID]config.Profile{
			"work": {Providers: map[config.ProviderID]config.ProviderConfig{
				"codex":  {"credential_ref": "codex/work", "codex_home": codexHome},
				"github": {"host": "oss.navercorp.com", "account": "work-user", "auth_source": "gh"},
			}},
			"personal": {Providers: map[config.ProviderID]config.ProviderConfig{
				"codex":  {"credential_ref": "codex/personal", "codex_home": filepath.Join(managedRoot, "personal")},
				"github": {"host": "github.com", "account": "personal-user", "auth_source": "gh"},
			}},
		},
	}
	if err := store.Save(document); err != nil {
		t.Fatal(err)
	}
	runner := execution.NewFakeRunner()
	runner.Available["codex"] = true
	runner.Results["codex\x00login\x00status"] = execution.Result{}
	credentialsStore := credentials.NewMemoryStore()
	if err := credentialsStore.Put(context.Background(), "codex/work", []byte("opaque-work-auth")); err != nil {
		t.Fatal(err)
	}
	var output, errors bytes.Buffer
	cli := NewCLI(store, runner, &output, &errors)
	cli.Credentials = credentialsStore
	cli.ManagedRoot = managedRoot
	if got := cli.Run([]string{"change", "--work", "--codex"}); got != 0 {
		t.Fatalf("change exit=%d output=%s errors=%s", got, output.String(), errors.String())
	}
	updated, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if updated.Active["codex"] != "work" || updated.Active["github"] != "personal" {
		t.Fatalf("partial change active state = %v", updated.Active)
	}
	auth, err := os.ReadFile(filepath.Join(codexHome, "auth.json"))
	if err != nil || string(auth) != "opaque-work-auth" {
		t.Fatalf("Codex auth cache = %q, %v", auth, err)
	}
	if len(runner.Invocations) != 1 || runner.Invocations[0].Command.Name != "codex" {
		t.Fatalf("non-target provider was accessed: %+v", runner.Invocations)
	}
}
