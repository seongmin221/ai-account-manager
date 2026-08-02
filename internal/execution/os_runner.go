package execution

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
)

// OSRunner executes argv directly without invoking a shell.
type OSRunner struct{}

func (OSRunner) Exists(ctx context.Context, name string) bool {
	select {
	case <-ctx.Done():
		return false
	default:
	}
	_, err := exec.LookPath(name)
	return err == nil
}

func (OSRunner) Run(ctx context.Context, command Command) (Result, error) {
	process := exec.CommandContext(ctx, command.Name, command.Args...)
	process.Dir = command.Dir
	process.Env = patchedEnvironment(os.Environ(), command.Env, command.Unset)

	var stdout, stderr bytes.Buffer
	process.Stdout = &stdout
	process.Stderr = &stderr
	err := process.Run()
	result := Result{Stdout: stdout.String(), Stderr: stderr.String()}
	if process.ProcessState != nil {
		result.ExitCode = process.ProcessState.ExitCode()
	}
	if err == nil {
		return result, nil
	}
	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		return result, nil
	}
	return result, err
}

func patchedEnvironment(base []string, set map[string]string, unset []string) []string {
	values := make(map[string]string, len(base)+len(set))
	for _, item := range base {
		for index, char := range item {
			if char == '=' {
				values[item[:index]] = item[index+1:]
				break
			}
		}
	}
	for _, key := range unset {
		delete(values, key)
	}
	for key, value := range set {
		values[key] = value
	}
	result := make([]string, 0, len(values))
	for key, value := range values {
		result = append(result, key+"="+value)
	}
	return result
}
