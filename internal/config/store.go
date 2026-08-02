package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/seongmin221/ai-account-manager/internal/errors"
)

const (
	configDirMode  os.FileMode = 0o700
	configFileMode os.FileMode = 0o600
)

// DefaultPath resolves the documented config location for a supplied home
// directory. Supplying home makes the path deterministic in tests.
func DefaultPath(home string) string {
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	return filepath.Join(home, ".config", "account-manager", "config.toml")
}

// FileStore persists configuration using TOML and atomic replacement.
type FileStore struct {
	Path string
}

func NewFileStore(path string) *FileStore {
	return &FileStore{Path: path}
}

func (s *FileStore) Load() (Document, error) {
	file, err := os.Open(s.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return Document{}, apperror(apperrors.ConfigNotFound, fmt.Sprintf("configuration file %q was not found", s.Path), err)
		}
		return Document{}, apperror(apperrors.PermissionError, fmt.Sprintf("cannot open configuration file %q", s.Path), err)
	}
	defer file.Close()

	var raw rawDocument
	metadata, err := toml.DecodeReader(file, &raw)
	if err != nil {
		return Document{}, apperror(apperrors.ConfigSyntax, "configuration file is not valid TOML", err)
	}
	if !metadata.IsDefined("version") {
		return Document{}, apperror(apperrors.UnsupportedConfigVersion, "configuration version is required", nil)
	}
	document := fromRaw(raw)
	if err := document.Validate(); err != nil {
		return Document{}, err
	}
	return document, nil
}

// Save validates the complete document before writing it. The temporary file
// lives beside the destination so rename is atomic on the same filesystem.
func (s *FileStore) Save(document Document) error {
	if err := document.Validate(); err != nil {
		return err
	}
	directory := filepath.Dir(s.Path)
	if err := os.MkdirAll(directory, configDirMode); err != nil {
		return apperror(apperrors.PermissionError, fmt.Sprintf("cannot create configuration directory %q", directory), err)
	}
	if err := os.Chmod(directory, configDirMode); err != nil {
		return apperror(apperrors.PermissionError, fmt.Sprintf("cannot set configuration directory permissions for %q", directory), err)
	}

	temporary, err := os.CreateTemp(directory, ".config.toml.tmp-*")
	if err != nil {
		return apperror(apperrors.PermissionError, "cannot create temporary configuration file", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)

	if err := temporary.Chmod(configFileMode); err != nil {
		temporary.Close()
		return apperror(apperrors.PermissionError, "cannot set temporary configuration permissions", err)
	}
	if err := toml.NewEncoder(temporary).Encode(toRaw(document)); err != nil {
		temporary.Close()
		return apperror(apperrors.ConfigSyntax, "cannot encode configuration as TOML", err)
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return apperror(apperrors.PermissionError, "cannot flush temporary configuration file", err)
	}
	if err := temporary.Close(); err != nil {
		return apperror(apperrors.PermissionError, "cannot close temporary configuration file", err)
	}
	if err := os.Rename(temporaryPath, s.Path); err != nil {
		return apperror(apperrors.PermissionError, fmt.Sprintf("cannot replace configuration file %q", s.Path), err)
	}
	return syncDirectory(directory)
}

func syncDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return apperror(apperrors.PermissionError, "cannot open configuration directory for sync", err)
	}
	defer directory.Close()
	if err := directory.Sync(); err != nil {
		return apperror(apperrors.PermissionError, "cannot sync configuration directory", err)
	}
	return nil
}

func apperror(code apperrors.Code, message string, cause error) error {
	return &apperrors.Error{Code: code, Message: message, Cause: cause}
}

// The TOML library works reliably with built-in string map keys. These wire
// types keep that limitation at the persistence boundary while the rest of
// the application uses typed IDs.
type rawDocument struct {
	Version  int                   `toml:"version"`
	Active   map[string]string     `toml:"active"`
	Profiles map[string]rawProfile `toml:"profiles"`
}

type rawProfile struct {
	DisplayName string                       `toml:"display_name"`
	Providers   map[string]map[string]string `toml:"providers"`
}

func fromRaw(raw rawDocument) Document {
	document := NewDocument()
	document.Version = raw.Version
	for provider, profile := range raw.Active {
		document.Active[ProviderID(provider)] = ProfileID(profile)
	}
	for profileID, rawProfile := range raw.Profiles {
		profile := Profile{
			DisplayName: rawProfile.DisplayName,
			Providers:   make(map[ProviderID]ProviderConfig, len(rawProfile.Providers)),
		}
		for providerID, settings := range rawProfile.Providers {
			profile.Providers[ProviderID(providerID)] = ProviderConfig(settings)
		}
		document.Profiles[ProfileID(profileID)] = profile
	}
	return document
}

func toRaw(document Document) rawDocument {
	raw := rawDocument{
		Version:  document.Version,
		Active:   make(map[string]string, len(document.Active)),
		Profiles: make(map[string]rawProfile, len(document.Profiles)),
	}
	for provider, profile := range document.Active {
		raw.Active[string(provider)] = string(profile)
	}
	for profileID, profile := range document.Profiles {
		rawProfile := rawProfile{
			DisplayName: profile.DisplayName,
			Providers:   make(map[string]map[string]string, len(profile.Providers)),
		}
		for providerID, settings := range profile.Providers {
			rawProfile.Providers[string(providerID)] = map[string]string(settings)
		}
		raw.Profiles[string(profileID)] = rawProfile
	}
	return raw
}
