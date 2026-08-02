package providers

import (
	"context"
	"fmt"
	"sort"

	"github.com/seongmin221/ai-account-manager/internal/config"
	"github.com/seongmin221/ai-account-manager/internal/errors"
)

// Transaction coordinates provider changes without knowing provider details.
// It commits active state only after every selected provider has applied.
type Transaction struct {
	Registry Registry
}

type ChangeResult struct {
	Selected   []config.ProviderID
	Applied    []config.ProviderID
	RolledBack []config.ProviderID
	rollback   []appliedProvider
}

type appliedProvider struct {
	id   config.ProviderID
	plan RollbackPlan
}

func (t Transaction) Change(ctx context.Context, document config.Document, target config.ProfileID, scope Scope) (config.Document, ChangeResult, error) {
	profile, ok := document.Profiles[target]
	if !ok {
		return document, ChangeResult{}, apperrors.New(apperrors.InvalidActiveReference, fmt.Sprintf("profile %q is not registered", target))
	}
	if err := target.Validate(); err != nil {
		return document, ChangeResult{}, apperrors.New(apperrors.InvalidProfileID, err.Error())
	}
	available := make([]config.ProviderID, 0, len(profile.Providers))
	for id := range profile.Providers {
		available = append(available, id)
	}
	sort.Slice(available, func(i, j int) bool { return available[i] < available[j] })
	selected, err := scope.Resolve(available)
	if err != nil {
		return document, ChangeResult{}, apperrors.New(apperrors.UnknownProvider, err.Error())
	}
	if len(selected) == 0 {
		return document, ChangeResult{}, apperrors.New(apperrors.ProviderConfigInvalid, fmt.Sprintf("profile %q has no providers in the requested scope", target))
	}

	type prepared struct {
		id   config.ProviderID
		plan ActivationPlan
	}
	preparedPlans := make([]prepared, 0, len(selected))
	for _, id := range selected {
		provider, ok := t.Registry.Get(id)
		if !ok {
			return document, ChangeResult{Selected: selected}, apperrors.New(apperrors.UnknownProvider, fmt.Sprintf("provider %q is not registered", id))
		}
		settings := profile.Providers[id]
		if err := provider.Validate(ctx, settings); err != nil {
			return document, ChangeResult{Selected: selected}, apperrors.New(apperrors.ProviderConfigInvalid, fmt.Sprintf("provider %q preflight failed: %v", id, err))
		}
		plan, err := provider.PlanActivate(ctx, settings)
		if err != nil {
			return document, ChangeResult{Selected: selected}, apperrors.New(apperrors.ActivationFailed, fmt.Sprintf("provider %q activation plan failed: %v", id, err))
		}
		preparedPlans = append(preparedPlans, prepared{id: id, plan: plan})
	}

	result := ChangeResult{Selected: selected}
	rollbackPlans := make([]struct {
		id   config.ProviderID
		plan RollbackPlan
	}, 0, len(preparedPlans))
	for _, item := range preparedPlans {
		provider, _ := t.Registry.Get(item.id)
		rollback, err := provider.Apply(ctx, item.plan)
		if err != nil {
			result.Applied = append(result.Applied, item.id)
			rollbackErr := rollbackApplied(ctx, t.Registry, rollbackPlans, &result)
			if rollbackErr != nil {
				return document, result, rollbackErr
			}
			return document, result, apperrors.New(apperrors.ActivationFailed, fmt.Sprintf("provider %q activation failed: %v", item.id, err))
		}
		rollbackPlans = append(rollbackPlans, struct {
			id   config.ProviderID
			plan RollbackPlan
		}{id: item.id, plan: rollback})
		result.Applied = append(result.Applied, item.id)
	}
	result.rollback = make([]appliedProvider, 0, len(rollbackPlans))
	for _, item := range rollbackPlans {
		result.rollback = append(result.rollback, appliedProvider{id: item.id, plan: item.plan})
	}

	updated := document
	updated.Active = cloneActive(document.Active)
	if updated.Active == nil {
		updated.Active = make(map[config.ProviderID]config.ProfileID)
	}
	for _, id := range selected {
		updated.Active[id] = target
	}
	return updated, result, nil
}

// Rollback restores external provider state after a later commit operation
// (such as config file replacement) fails.
func (r *ChangeResult) Rollback(ctx context.Context, registry Registry) error {
	for index := len(r.rollback) - 1; index >= 0; index-- {
		item := r.rollback[index]
		provider, ok := registry.Get(item.id)
		if !ok {
			return apperrors.New(apperrors.RollbackFailed, fmt.Sprintf("provider %q is unavailable during rollback", item.id))
		}
		if err := provider.Rollback(ctx, item.plan); err != nil {
			return apperrors.New(apperrors.RollbackFailed, fmt.Sprintf("provider %q rollback failed: %v", item.id, err))
		}
		r.RolledBack = append(r.RolledBack, item.id)
	}
	r.rollback = nil
	return nil
}

func rollbackApplied(ctx context.Context, registry Registry, plans []struct {
	id   config.ProviderID
	plan RollbackPlan
}, result *ChangeResult) error {
	for index := len(plans) - 1; index >= 0; index-- {
		item := plans[index]
		provider, _ := registry.Get(item.id)
		if err := provider.Rollback(ctx, item.plan); err != nil {
			return apperrors.New(apperrors.RollbackFailed, fmt.Sprintf("provider %q rollback failed: %v", item.id, err))
		}
		result.RolledBack = append(result.RolledBack, item.id)
	}
	return nil
}

func cloneActive(active map[config.ProviderID]config.ProfileID) map[config.ProviderID]config.ProfileID {
	if active == nil {
		return nil
	}
	cloned := make(map[config.ProviderID]config.ProfileID, len(active))
	for provider, profile := range active {
		cloned[provider] = profile
	}
	return cloned
}
