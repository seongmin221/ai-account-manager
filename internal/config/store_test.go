package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	apperrors "github.com/seongmin221/ai-account-manager/internal/errors"
)

func TestFileStoreRoundTripPreservesUnknownProviders(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "nested", "config.toml")
	store := NewFileStore(path)
	document := Document{
		Version: 1,
		Active: map[ProviderID]ProfileID{
			"github": "personal",
			"codex":  "work",
		},
		Profiles: map[ProfileID]Profile{
			"work": {Providers: map[ProviderID]ProviderConfig{
				"github": {"host": "oss.navercorp.com", "account": "work-user"},
				"codex":  {"credential_ref": "codex/work", "codex_home": "~/.local/share/account-manager/codex/work"},
			}},
			"personal": {Providers: map[ProviderID]ProviderConfig{
				"github": {"host": "github.com", "account": "personal-user"},
				"codex":  {"credential_ref": "codex/personal", "codex_home": "~/.local/share/account-manager/codex/personal"},
				"slack":  {"account": "future-user"},
			}},
		},
	}

	if err := store.Save(document); err != nil {
		t.Fatal(err)
	}
	got, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.OverallMode() != "mixed" || got.Profiles["personal"].Providers["slack"]["account"] != "future-user" {
		t.Fatalf("round-trip lost configuration: %+v", got)
	}

	dirInfo, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if got := dirInfo.Mode().Perm(); got != 0o700 {
		t.Fatalf("config directory mode = %o, want 700", got)
	}
	fileInfo, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := fileInfo.Mode().Perm(); got != 0o600 {
		t.Fatalf("config file mode = %o, want 600", got)
	}
}

func TestFileStoreRejectsInvalidDocumentsWithoutWriting(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "config.toml")
	store := NewFileStore(path)
	invalid := Document{Version: 1, Active: map[ProviderID]ProfileID{"github": "missing"}}
	if err := store.Save(invalid); err == nil {
		t.Fatal("Save() accepted invalid active reference")
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("invalid save created a file: %v", err)
	}
}

func TestFileStoreReturnsStableErrors(t *testing.T) {
	root := t.TempDir()
	store := NewFileStore(filepath.Join(root, "config.toml"))
	if _, err := store.Load(); !hasCode(err, apperrors.ConfigNotFound) {
		t.Fatalf("missing config error = %v, want AM001", err)
	}
	if err := os.WriteFile(store.Path, []byte("version = ["), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Load(); !hasCode(err, apperrors.ConfigSyntax) {
		t.Fatalf("invalid TOML error = %v, want AM002", err)
	}
	if err := os.WriteFile(store.Path, []byte("version = 99"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Load(); !hasCode(err, apperrors.UnsupportedConfigVersion) {
		t.Fatalf("unsupported version error = %v, want AM003", err)
	}
}

func hasCode(err error, want apperrors.Code) bool {
	var coded *apperrors.Error
	return errors.As(err, &coded) && coded.Code == want
}
