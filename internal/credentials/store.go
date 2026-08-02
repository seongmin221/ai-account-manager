package credentials

import "context"

// Store abstracts OS-backed secret storage. Implementations must never expose
// secret values through errors or diagnostic output.
type Store interface {
	Put(ctx context.Context, ref string, secret []byte) error
	Get(ctx context.Context, ref string) ([]byte, error)
	Exists(ctx context.Context, ref string) (bool, error)
	Delete(ctx context.Context, ref string) error
}
