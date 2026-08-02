package providers

import (
	"context"
	"testing"

	"github.com/seongmin221/ai-account-manager/internal/config"
)

type fakeProvider struct{ id config.ProviderID }

func (p fakeProvider) ID() config.ProviderID { return p.id }
func (p fakeProvider) InspectCurrent(context.Context) (Identity, error) {
	return Identity{Provider: p.id}, nil
}
func (p fakeProvider) Capture(context.Context, config.ProfileID) (config.ProviderConfig, error) {
	return config.ProviderConfig{}, nil
}
func (p fakeProvider) Validate(context.Context, config.ProviderConfig) error { return nil }
func (p fakeProvider) PlanActivate(context.Context, config.ProviderConfig) (ActivationPlan, error) {
	return ActivationPlan{Provider: p.id}, nil
}
func (p fakeProvider) Apply(context.Context, ActivationPlan) (RollbackPlan, error) {
	return RollbackPlan{Provider: p.id}, nil
}
func (p fakeProvider) Rollback(context.Context, RollbackPlan) error { return nil }

func TestMemoryRegistryValidatesAndSortsProviders(t *testing.T) {
	r := NewMemoryRegistry()
	if err := r.Register(fakeProvider{id: "codex"}); err != nil {
		t.Fatal(err)
	}
	if err := r.Register(fakeProvider{id: "github"}); err != nil {
		t.Fatal(err)
	}
	if got, want := r.IDs(), []config.ProviderID{"codex", "github"}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("IDs() = %v, want %v", got, want)
	}
	if err := r.Register(fakeProvider{id: "github"}); err == nil {
		t.Fatal("duplicate registration succeeded")
	}
	if err := r.Register(fakeProvider{id: "BadID"}); err == nil {
		t.Fatal("invalid provider ID registration succeeded")
	}
}
