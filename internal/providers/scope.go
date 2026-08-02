package providers

import (
	"fmt"

	"github.com/seongmin221/ai-account-manager/internal/config"
)

// Scope selects providers for an operation. An empty scope means all
// registered providers; explicit selections preserve registry order.
type Scope struct {
	Only []config.ProviderID
}

func NewScope(only ...config.ProviderID) (Scope, error) {
	seen := make(map[config.ProviderID]struct{}, len(only))
	for _, id := range only {
		if err := id.Validate(); err != nil {
			return Scope{}, fmt.Errorf("invalid scope provider %q: %w", id, err)
		}
		if _, exists := seen[id]; exists {
			return Scope{}, fmt.Errorf("provider %q appears more than once in scope", id)
		}
		seen[id] = struct{}{}
	}
	return Scope{Only: append([]config.ProviderID(nil), only...)}, nil
}

func (s Scope) Resolve(available []config.ProviderID) ([]config.ProviderID, error) {
	if len(s.Only) == 0 {
		return append([]config.ProviderID(nil), available...), nil
	}
	allowed := make(map[config.ProviderID]struct{}, len(available))
	for _, id := range available {
		allowed[id] = struct{}{}
	}
	selected := make([]config.ProviderID, 0, len(s.Only))
	requested := make(map[config.ProviderID]struct{}, len(s.Only))
	for _, id := range s.Only {
		requested[id] = struct{}{}
	}
	for _, id := range available {
		if _, ok := requested[id]; ok {
			selected = append(selected, id)
		}
	}
	for _, id := range s.Only {
		if _, ok := allowed[id]; !ok {
			return nil, fmt.Errorf("provider %q is not registered", id)
		}
	}
	return selected, nil
}
