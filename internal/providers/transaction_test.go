package providers

import (
	"context"
	"errors"
	"testing"

	"github.com/seongmin221/ai-account-manager/internal/config"
)

type transactionProvider struct {
	id          config.ProviderID
	validateErr error
	planErr     error
	applyErr    error
	rollbackErr error
	state       *[]string
}

func (p *transactionProvider) ID() config.ProviderID { return p.id }
func (p *transactionProvider) InspectCurrent(context.Context) (Identity, error) {
	return Identity{Provider: p.id}, nil
}
func (p *transactionProvider) Capture(context.Context, config.ProfileID) (config.ProviderConfig, error) {
	return config.ProviderConfig{}, nil
}
func (p *transactionProvider) Validate(context.Context, config.ProviderConfig) error {
	return p.validateErr
}
func (p *transactionProvider) PlanActivate(context.Context, config.ProviderConfig) (ActivationPlan, error) {
	if p.planErr != nil {
		return ActivationPlan{}, p.planErr
	}
	return ActivationPlan{Provider: p.id}, nil
}
func (p *transactionProvider) Apply(context.Context, ActivationPlan) (RollbackPlan, error) {
	*p.state = append(*p.state, "apply:"+string(p.id))
	if p.applyErr != nil {
		return RollbackPlan{}, p.applyErr
	}
	return RollbackPlan{Provider: p.id}, nil
}
func (p *transactionProvider) Rollback(context.Context, RollbackPlan) error {
	*p.state = append(*p.state, "rollback:"+string(p.id))
	return p.rollbackErr
}

func TestTransactionPreflightsBeforeApplyingAndCommitsSelectedProviders(t *testing.T) {
	var state []string
	registry := NewMemoryRegistry()
	github := &transactionProvider{id: "github", state: &state}
	codex := &transactionProvider{id: "codex", state: &state}
	if err := registry.Register(github); err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(codex); err != nil {
		t.Fatal(err)
	}
	document := testDocument()
	updated, result, err := (Transaction{Registry: registry}).Change(context.Background(), document, "work", mustScope(t, "codex"))
	if err != nil {
		t.Fatal(err)
	}
	if got := state; len(got) != 1 || got[0] != "apply:codex" {
		t.Fatalf("state changes = %v, want only codex apply", got)
	}
	if document.Active["codex"] != "personal" || updated.Active["codex"] != "work" || updated.Active["github"] != "personal" {
		t.Fatalf("active state changed incorrectly: before=%v after=%v", document.Active, updated.Active)
	}
	if len(result.Applied) != 1 || result.Applied[0] != "codex" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestTransactionRollsBackInReverseOrderAndDoesNotCommitState(t *testing.T) {
	var state []string
	registry := NewMemoryRegistry()
	github := &transactionProvider{id: "github", state: &state, applyErr: errors.New("simulated failure")}
	codex := &transactionProvider{id: "codex", state: &state}
	if err := registry.Register(github); err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(codex); err != nil {
		t.Fatal(err)
	}
	document := testDocument()
	updated, result, err := (Transaction{Registry: registry}).Change(context.Background(), document, "work", mustScope(t))
	if err == nil {
		t.Fatal("transaction unexpectedly succeeded")
	}
	if got := state; len(got) != 3 || got[0] != "apply:codex" || got[1] != "apply:github" || got[2] != "rollback:codex" {
		t.Fatalf("state changes = %v, want reverse rollback", got)
	}
	if updated.Active["github"] != "personal" || updated.Active["codex"] != "personal" {
		t.Fatalf("failed transaction committed active state: %v", updated.Active)
	}
	if len(result.RolledBack) != 1 || result.RolledBack[0] != "codex" {
		t.Fatalf("unexpected rollback result: %+v", result)
	}
}

func TestTransactionStopsBeforeApplyWhenPreflightFails(t *testing.T) {
	var state []string
	registry := NewMemoryRegistry()
	github := &transactionProvider{id: "github", state: &state}
	codex := &transactionProvider{id: "codex", state: &state, validateErr: errors.New("missing credential")}
	if err := registry.Register(github); err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(codex); err != nil {
		t.Fatal(err)
	}
	_, _, err := (Transaction{Registry: registry}).Change(context.Background(), testDocument(), "work", mustScope(t))
	if err == nil || len(state) != 0 {
		t.Fatalf("preflight failure = %v, state changes = %v; want no apply", err, state)
	}
}

func testDocument() config.Document {
	return config.Document{
		Version: 1,
		Active:  map[config.ProviderID]config.ProfileID{"github": "personal", "codex": "personal"},
		Profiles: map[config.ProfileID]config.Profile{
			"work":     {Providers: map[config.ProviderID]config.ProviderConfig{"github": {"account": "work"}, "codex": {"credential_ref": "codex/work"}}},
			"personal": {Providers: map[config.ProviderID]config.ProviderConfig{"github": {"account": "personal"}, "codex": {"credential_ref": "codex/personal"}}},
		},
	}
}

func mustScope(t *testing.T, ids ...config.ProviderID) Scope {
	t.Helper()
	scope, err := NewScope(ids...)
	if err != nil {
		t.Fatal(err)
	}
	return scope
}
