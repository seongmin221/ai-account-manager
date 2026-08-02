package providers

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/seongmin221/ai-account-manager/internal/config"
	"github.com/seongmin221/ai-account-manager/internal/credentials"
	"github.com/seongmin221/ai-account-manager/internal/execution"
)

func TestCodexProviderRestoresOpaqueAuthCache(t *testing.T) {
	root := t.TempDir()
	managedRoot := filepath.Join(root, "managed")
	home := filepath.Join(managedRoot, "work")
	authPath := filepath.Join(home, "auth.json")
	if err := os.MkdirAll(home, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(authPath, []byte("old-auth"), 0o600); err != nil {
		t.Fatal(err)
	}

	runner := execution.NewFakeRunner()
	runner.Available["codex"] = true
	runner.Results["codex\x00login\x00status"] = execution.Result{}
	store := credentials.NewMemoryStore()
	if err := store.Put(context.Background(), "codex/work", []byte("opaque-new-auth")); err != nil {
		t.Fatal(err)
	}
	provider := &CodexProvider{Runner: runner, Credentials: store, CurrentHome: filepath.Join(root, "current"), ManagedRoot: managedRoot}
	settings := config.ProviderConfig{"credential_ref": "codex/work", "codex_home": home}
	plan, err := provider.PlanActivate(context.Background(), settings)
	if err != nil {
		t.Fatal(err)
	}
	rollback, err := provider.Apply(context.Background(), plan)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(authPath)
	if err != nil || string(got) != "opaque-new-auth" {
		t.Fatalf("activated auth cache = %q, %v", got, err)
	}
	info, err := os.Stat(authPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("auth cache mode = %o, want 600", info.Mode().Perm())
	}
	if err := provider.Rollback(context.Background(), rollback); err != nil {
		t.Fatal(err)
	}
	got, err = os.ReadFile(authPath)
	if err != nil || string(got) != "old-auth" {
		t.Fatalf("rolled back auth cache = %q, %v", got, err)
	}
}

func TestCodexProviderRejectsHomesOutsideManagedRoot(t *testing.T) {
	runner := execution.NewFakeRunner()
	runner.Available["codex"] = true
	store := credentials.NewMemoryStore()
	if err := store.Put(context.Background(), "codex/work", []byte("auth")); err != nil {
		t.Fatal(err)
	}
	provider := &CodexProvider{Runner: runner, Credentials: store, ManagedRoot: filepath.Join(t.TempDir(), "managed")}
	if err := provider.Validate(context.Background(), config.ProviderConfig{"credential_ref": "codex/work", "codex_home": t.TempDir()}); err == nil {
		t.Fatal("Validate accepted Codex home outside managed root")
	}
}
