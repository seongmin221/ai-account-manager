package providers

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/seongmin221/ai-account-manager/internal/config"
	"github.com/seongmin221/ai-account-manager/internal/execution"
)

const githubProviderID config.ProviderID = "github"

type GitHubProvider struct {
	Runner      execution.CommandRunner
	CurrentHost string
}

func NewGitHubProvider(runner execution.CommandRunner) *GitHubProvider {
	return &GitHubProvider{Runner: runner, CurrentHost: os.Getenv("GH_HOST")}
}

func (p *GitHubProvider) ID() config.ProviderID { return githubProviderID }

func (p *GitHubProvider) InspectCurrent(ctx context.Context) (Identity, error) {
	host := p.CurrentHost
	if host == "" {
		host = "github.com"
	}
	result, err := p.Runner.Run(ctx, execution.Command{
		Name:  "gh",
		Args:  []string{"api", "user", "--hostname", host, "--jq", ".login"},
		Unset: githubTokenEnvironmentNames,
	})
	if err != nil {
		return Identity{}, fmt.Errorf("cannot inspect GitHub account: %w", err)
	}
	if result.ExitCode != 0 {
		return Identity{}, fmt.Errorf("GitHub account inspection failed with exit code %d", result.ExitCode)
	}
	account := strings.TrimSpace(result.Stdout)
	if account == "" {
		return Identity{}, fmt.Errorf("GitHub account inspection returned no account")
	}
	return Identity{Provider: p.ID(), Host: host, Account: account}, nil
}

func (p *GitHubProvider) Capture(ctx context.Context, target config.ProfileID) (config.ProviderConfig, error) {
	identity, err := p.InspectCurrent(ctx)
	if err != nil {
		return nil, err
	}
	return config.ProviderConfig{
		"host":        identity.Host,
		"account":     identity.Account,
		"auth_source": "gh",
	}, nil
}

func (p *GitHubProvider) Validate(ctx context.Context, settings config.ProviderConfig) error {
	host := settings["host"]
	account := settings["account"]
	if err := validateGitHubHost(host); err != nil {
		return err
	}
	if account == "" {
		return fmt.Errorf("GitHub account is required")
	}
	if source := settings["auth_source"]; source != "" && source != "gh" {
		return fmt.Errorf("unsupported GitHub auth source %q", source)
	}
	if !p.Runner.Exists(ctx, "gh") {
		return fmt.Errorf("gh is not installed")
	}
	return nil
}

func (p *GitHubProvider) PlanActivate(ctx context.Context, settings config.ProviderConfig) (ActivationPlan, error) {
	if err := p.Validate(ctx, settings); err != nil {
		return ActivationPlan{}, err
	}
	previous, err := p.InspectCurrent(ctx)
	if err != nil {
		return ActivationPlan{}, err
	}
	return ActivationPlan{
		Provider: p.ID(),
		Data: githubActivationPlan{
			Target:   Identity{Provider: p.ID(), Host: settings["host"], Account: settings["account"]},
			Previous: previous,
		},
	}, nil
}

func (p *GitHubProvider) Apply(ctx context.Context, plan ActivationPlan) (RollbackPlan, error) {
	activation, ok := plan.Data.(githubActivationPlan)
	if !ok || plan.Provider != p.ID() {
		return RollbackPlan{}, fmt.Errorf("invalid GitHub activation plan")
	}
	result, err := p.Runner.Run(ctx, execution.Command{
		Name:  "gh",
		Args:  []string{"auth", "switch", "--hostname", activation.Target.Host, "--user", activation.Target.Account},
		Unset: githubTokenEnvironmentNames,
	})
	if err != nil {
		return RollbackPlan{}, fmt.Errorf("cannot switch GitHub account: %w", err)
	}
	if result.ExitCode != 0 {
		return RollbackPlan{}, fmt.Errorf("GitHub account switch failed with exit code %d", result.ExitCode)
	}
	return RollbackPlan{Provider: p.ID(), Data: activation.Previous}, nil
}

func (p *GitHubProvider) Rollback(ctx context.Context, plan RollbackPlan) error {
	previous, ok := plan.Data.(Identity)
	if !ok || plan.Provider != p.ID() {
		return fmt.Errorf("invalid GitHub rollback plan")
	}
	if previous.Host == "" || previous.Account == "" {
		return nil
	}
	result, err := p.Runner.Run(ctx, execution.Command{
		Name:  "gh",
		Args:  []string{"auth", "switch", "--hostname", previous.Host, "--user", previous.Account},
		Unset: githubTokenEnvironmentNames,
	})
	if err != nil {
		return fmt.Errorf("cannot restore GitHub account: %w", err)
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("GitHub account rollback failed with exit code %d", result.ExitCode)
	}
	return nil
}

var githubTokenEnvironmentNames = []string{
	"GH_TOKEN",
	"GITHUB_TOKEN",
	"GH_ENTERPRISE_TOKEN",
	"GITHUB_ENTERPRISE_TOKEN",
}

type githubActivationPlan struct {
	Target   Identity
	Previous Identity
}

func validateGitHubHost(host string) error {
	if host == "" {
		return fmt.Errorf("GitHub host is required")
	}
	if strings.ContainsAny(host, "/:@") {
		return fmt.Errorf("GitHub host must be a hostname without scheme or path")
	}
	parsed, err := url.Parse("https://" + host)
	if err != nil || parsed.Hostname() != host || parsed.Path != "" {
		return fmt.Errorf("GitHub host %q is invalid", host)
	}
	return nil
}
