package providers

import (
	"context"

	"github.com/seongmin221/ai-account-manager/internal/config"
)

type Identity struct {
	Provider config.ProviderID
	Host     string
	Account  string
}

// ActivationPlan and RollbackPlan intentionally carry provider-owned data.
// The transaction engine treats them as opaque values.
type ActivationPlan struct {
	Provider config.ProviderID
	Data     any
}

type RollbackPlan struct {
	Provider config.ProviderID
	Data     any
}

// Provider is the common lifecycle contract for account domains.
type Provider interface {
	ID() config.ProviderID
	InspectCurrent(ctx context.Context) (Identity, error)
	Capture(ctx context.Context, target config.ProfileID) (config.ProviderConfig, error)
	Validate(ctx context.Context, settings config.ProviderConfig) error
	PlanActivate(ctx context.Context, settings config.ProviderConfig) (ActivationPlan, error)
	Apply(ctx context.Context, plan ActivationPlan) (RollbackPlan, error)
	Rollback(ctx context.Context, plan RollbackPlan) error
}
