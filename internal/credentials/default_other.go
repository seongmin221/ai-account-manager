//go:build !darwin

package credentials

func NewDefaultStore() Store { return NewMemoryStore() }
