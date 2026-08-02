package credentials

import (
	"context"
	"testing"
)

func TestMemoryStoreCopiesSecretsAndImplementsContract(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	input := []byte("opaque-auth-cache")
	if err := store.Put(ctx, "codex/work", input); err != nil {
		t.Fatal(err)
	}
	input[0] = 'X'

	got, err := store.Get(ctx, "codex/work")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "opaque-auth-cache" {
		t.Fatalf("Get() = %q, want original secret", got)
	}
	got[0] = 'Y'
	gotAgain, err := store.Get(ctx, "codex/work")
	if err != nil {
		t.Fatal(err)
	}
	if string(gotAgain) != "opaque-auth-cache" {
		t.Fatalf("stored secret was exposed to caller mutation: %q", gotAgain)
	}

	exists, err := store.Exists(ctx, "codex/work")
	if err != nil || !exists {
		t.Fatalf("Exists() = %v, %v; want true, nil", exists, err)
	}
	if err := store.Delete(ctx, "codex/work"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(ctx, "codex/work"); err != ErrNotFound {
		t.Fatalf("Get() error = %v, want ErrNotFound", err)
	}
}
