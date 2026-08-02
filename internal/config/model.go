package config

import (
	"fmt"
	"regexp"
	"sort"

	"github.com/seongmin221/ai-account-manager/internal/errors"
)

// ProfileID identifies a named collection of provider configurations.
type ProfileID string

// ProviderID identifies an account domain such as github or codex.
type ProviderID string

// ProviderConfig contains provider-specific metadata. Secrets are deliberately
// not represented here; providers keep them in their credential stores.
type ProviderConfig map[string]string

// Profile is a named collection of provider configurations.
type Profile struct {
	DisplayName string
	Providers   map[ProviderID]ProviderConfig
}

// Document is the in-memory representation of account-manager configuration.
// Persistence and TOML encoding are implemented in a later stage.
type Document struct {
	Version  int
	Active   map[ProviderID]ProfileID
	Profiles map[ProfileID]Profile
}

// NewDocument returns an empty v1 configuration document.
func NewDocument() Document {
	return Document{
		Version:  1,
		Active:   make(map[ProviderID]ProfileID),
		Profiles: make(map[ProfileID]Profile),
	}
}

var identifierPattern = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,31}$`)

// ValidateIdentifier applies the common profile/provider identifier rule.
func ValidateIdentifier(value string) error {
	if !identifierPattern.MatchString(value) {
		return fmt.Errorf("identifier %q must match ^[a-z][a-z0-9_-]{0,31}$", value)
	}
	return nil
}

func (id ProfileID) Validate() error {
	return ValidateIdentifier(string(id))
}

func (id ProviderID) Validate() error {
	return ValidateIdentifier(string(id))
}

// ProviderIDs returns provider IDs in deterministic order.
func (d Document) ProviderIDs() []ProviderID {
	ids := make([]ProviderID, 0, len(d.Active))
	for id := range d.Active {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

// OverallMode calculates the user-facing aggregate state from provider-level
// active profiles. An empty state is unconfigured.
func (d Document) OverallMode() string {
	if len(d.Active) == 0 {
		return "unconfigured"
	}

	var first ProfileID
	for _, profile := range d.Active {
		if first == "" {
			first = profile
			continue
		}
		if profile != first {
			return "mixed"
		}
	}
	return string(first)
}

// Validate checks the schema-level invariants that do not require a concrete
// provider implementation. Provider-specific validation happens during
// provider preflight.
func (d Document) Validate() error {
	if d.Version != 1 {
		return apperrors.New(apperrors.UnsupportedConfigVersion, fmt.Sprintf("unsupported configuration version %d", d.Version))
	}
	for profileID, profile := range d.Profiles {
		if err := profileID.Validate(); err != nil {
			return apperrors.New(apperrors.InvalidProfileID, err.Error())
		}
		for providerID, settings := range profile.Providers {
			if err := providerID.Validate(); err != nil {
				return apperrors.New(apperrors.InvalidProviderID, err.Error())
			}
			for key := range settings {
				if key == "" {
					return apperrors.New(apperrors.ProviderConfigInvalid, "provider configuration contains an empty field name")
				}
			}
		}
	}
	for providerID, profileID := range d.Active {
		if err := providerID.Validate(); err != nil {
			return apperrors.New(apperrors.InvalidProviderID, err.Error())
		}
		profile, ok := d.Profiles[profileID]
		if !ok {
			return apperrors.New(apperrors.InvalidActiveReference, fmt.Sprintf("active provider %q refers to missing profile %q", providerID, profileID))
		}
		if _, ok := profile.Providers[providerID]; !ok {
			return apperrors.New(apperrors.InvalidActiveReference, fmt.Sprintf("active provider %q is not configured in profile %q", providerID, profileID))
		}
	}
	return nil
}
