package execution

import "context"

// Command describes an external process invocation without using a shell.
type Command struct {
	Name  string
	Args  []string
	Env   map[string]string
	Unset []string
	Dir   string
}

type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// CommandRunner isolates gh/codex execution from provider logic.
type CommandRunner interface {
	Run(ctx context.Context, command Command) (Result, error)
	Exists(ctx context.Context, name string) bool
}
