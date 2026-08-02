package providers

import (
	"context"
	"testing"

	"github.com/seongmin221/ai-account-manager/internal/config"
	"github.com/seongmin221/ai-account-manager/internal/execution"
)

func TestGitHubProviderValidatesSafeMetadata(t *testing.T) {
	runner := execution.NewFakeRunner()
	runner.Available["gh"] = true
	provider := &GitHubProvider{Runner: runner, CurrentHost: "github.com"}
	if err := provider.Validate(context.Background(), config.ProviderConfig{"host": "oss.navercorp.com", "account": "work-user", "auth_source": "gh"}); err != nil {
		t.Fatal(err)
	}
	for _, host := range []string{"https://github.com", "github.com/path", "github.com:443"} {
		if err := provider.Validate(context.Background(), config.ProviderConfig{"host": host, "account": "user"}); err == nil {
			t.Fatalf("Validate accepted invalid host %q", host)
		}
	}
}

func TestGitHubProviderPlansSwitchAndRollbackWithoutTokens(t *testing.T) {
	runner := execution.NewFakeRunner()
	runner.Available["gh"] = true
	runner.Results["gh\x00api\x00user\x00--hostname\x00github.com\x00--jq\x00.login"] = execution.Result{Stdout: "personal-user\n"}
	runner.Results["gh\x00auth\x00switch\x00--hostname\x00oss.navercorp.com\x00--user\x00work-user"] = execution.Result{}
	runner.Results["gh\x00auth\x00switch\x00--hostname\x00github.com\x00--user\x00personal-user"] = execution.Result{}
	provider := &GitHubProvider{Runner: runner, CurrentHost: "github.com"}
	plan, err := provider.PlanActivate(context.Background(), config.ProviderConfig{"host": "oss.navercorp.com", "account": "work-user", "auth_source": "gh"})
	if err != nil {
		t.Fatal(err)
	}
	rollback, err := provider.Apply(context.Background(), plan)
	if err != nil {
		t.Fatal(err)
	}
	if err := provider.Rollback(context.Background(), rollback); err != nil {
		t.Fatal(err)
	}
	if len(runner.Invocations) != 3 {
		t.Fatalf("recorded %d commands, want inspect/apply/rollback", len(runner.Invocations))
	}
	for _, invocation := range runner.Invocations {
		for _, arg := range invocation.Command.Args {
			if arg == "token" || arg == "secret" {
				t.Fatal("credential material appeared in GitHub argv")
			}
		}
	}
}
