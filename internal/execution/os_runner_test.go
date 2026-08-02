package execution

import (
	"context"
	"testing"
)

func TestOSRunnerUsesArgvAndAppliesEnvironmentPatch(t *testing.T) {
	result, err := (OSRunner{}).Run(context.Background(), Command{
		Name:  "/usr/bin/printf",
		Args:  []string{"%s", "safe-output"},
		Env:   map[string]string{"ACCOUNT_MANAGER_TEST": "present"},
		Unset: []string{"ACCOUNT_MANAGER_REMOVED"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ExitCode != 0 || result.Stdout != "safe-output" {
		t.Fatalf("unexpected result: %+v", result)
	}
}
