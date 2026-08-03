package execution

import (
	"context"
	"fmt"
	"sync"
)

type Invocation struct {
	Command Command
}

// FakeRunner is deterministic and records argv/env boundaries for tests.
type FakeRunner struct {
	mu          sync.Mutex
	Available   map[string]bool
	Results     map[string]Result
	RunErrors   map[string]error
	Invocations []Invocation
}

func NewFakeRunner() *FakeRunner {
	return &FakeRunner{
		Available: make(map[string]bool),
		Results:   make(map[string]Result),
		RunErrors: make(map[string]error),
	}
}

func (r *FakeRunner) Exists(ctx context.Context, name string) bool {
	select {
	case <-ctx.Done():
		return false
	default:
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.Available[name]
}

func (r *FakeRunner) Run(ctx context.Context, command Command) (Result, error) {
	select {
	case <-ctx.Done():
		return Result{}, ctx.Err()
	default:
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	key := commandKey(command)
	r.Invocations = append(r.Invocations, Invocation{Command: cloneCommand(command)})
	if err := r.RunErrors[key]; err != nil {
		return Result{}, err
	}
	if result, ok := r.Results[key]; ok {
		return result, nil
	}
	return Result{}, fmt.Errorf("fake runner has no result for %q", key)
}

func commandKey(command Command) string {
	key := command.Name
	for _, arg := range command.Args {
		key += "\x00" + arg
	}
	return key
}

func cloneCommand(command Command) Command {
	args := append([]string(nil), command.Args...)
	env := make(map[string]string, len(command.Env))
	for key, value := range command.Env {
		env[key] = value
	}
	command.Args = args
	command.Env = env
	command.Unset = append([]string(nil), command.Unset...)
	return command
}
