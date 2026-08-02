package providers

import (
	"fmt"
	"sort"

	"github.com/seongmin221/ai-account-manager/internal/config"
)

// Registry provides provider lookup without coupling the application to
// concrete GitHub/Codex implementations.
type Registry interface {
	Register(provider Provider) error
	Get(id config.ProviderID) (Provider, bool)
	IDs() []config.ProviderID
}

type MemoryRegistry struct {
	providers map[config.ProviderID]Provider
}

func NewMemoryRegistry() *MemoryRegistry {
	return &MemoryRegistry{providers: make(map[config.ProviderID]Provider)}
}

func (r *MemoryRegistry) Register(provider Provider) error {
	if provider == nil {
		return fmt.Errorf("provider must not be nil")
	}
	id := provider.ID()
	if err := id.Validate(); err != nil {
		return fmt.Errorf("invalid provider ID: %w", err)
	}
	if _, exists := r.providers[id]; exists {
		return fmt.Errorf("provider %q is already registered", id)
	}
	r.providers[id] = provider
	return nil
}

func (r *MemoryRegistry) Get(id config.ProviderID) (Provider, bool) {
	provider, ok := r.providers[id]
	return provider, ok
}

func (r *MemoryRegistry) IDs() []config.ProviderID {
	ids := make([]config.ProviderID, 0, len(r.providers))
	for id := range r.providers {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}
