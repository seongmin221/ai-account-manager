package providers

import (
	"testing"

	"github.com/seongmin221/ai-account-manager/internal/config"
)

func TestScopeResolvesAllAndPartialSelections(t *testing.T) {
	available := []config.ProviderID{"codex", "github"}
	all, err := NewScope()
	if err != nil {
		t.Fatal(err)
	}
	if got, err := all.Resolve(available); err != nil || len(got) != 2 {
		t.Fatalf("all scope = %v, %v; want both providers", got, err)
	}

	partial, err := NewScope("github")
	if err != nil {
		t.Fatal(err)
	}
	got, err := partial.Resolve(available)
	if err != nil || len(got) != 1 || got[0] != "github" {
		t.Fatalf("partial scope = %v, %v; want [github]", got, err)
	}
	if _, err := NewScope("github", "github"); err == nil {
		t.Fatal("duplicate scope provider succeeded")
	}
	if _, err := partial.Resolve([]config.ProviderID{"codex"}); err == nil {
		t.Fatal("unknown scoped provider succeeded")
	}
}
