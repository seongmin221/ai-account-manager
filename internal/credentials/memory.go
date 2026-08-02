package credentials

import (
	"context"
	"errors"
	"sync"
)

var ErrNotFound = errors.New("credential not found")

// MemoryStore is a test double that copies byte slices at its boundary so
// callers cannot mutate stored secrets accidentally.
type MemoryStore struct {
	mu      sync.RWMutex
	secrets map[string][]byte
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{secrets: make(map[string][]byte)}
}

func (s *MemoryStore) Put(ctx context.Context, ref string, secret []byte) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.secrets[ref] = append([]byte(nil), secret...)
	return nil
}

func (s *MemoryStore) Get(ctx context.Context, ref string) ([]byte, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	secret, ok := s.secrets[ref]
	if !ok {
		return nil, ErrNotFound
	}
	return append([]byte(nil), secret...), nil
}

func (s *MemoryStore) Exists(ctx context.Context, ref string) (bool, error) {
	if err := contextError(ctx); err != nil {
		return false, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.secrets[ref]
	return ok, nil
}

func (s *MemoryStore) Delete(ctx context.Context, ref string) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.secrets, ref)
	return nil
}

func contextError(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}
