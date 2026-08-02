package execution

import (
	"context"
	"testing"
)

func TestFakeRunnerRecordsStructuredInvocation(t *testing.T) {
	runner := NewFakeRunner()
	runner.Results["gh\x00auth\x00status"] = Result{Stdout: "logged in", ExitCode: 0}

	result, err := runner.Run(context.Background(), Command{
		Name: "gh",
		Args: []string{"auth", "status"},
		Env:  map[string]string{"GH_HOST": "github.com"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Stdout != "logged in" || result.ExitCode != 0 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if len(runner.Invocations) != 1 {
		t.Fatalf("recorded %d invocations, want 1", len(runner.Invocations))
	}
	runner.Invocations[0].Command.Args[0] = "changed"
	if runner.Invocations[0].Command.Env["GH_HOST"] != "github.com" {
		t.Fatal("recorded command was not retained")
	}
}
