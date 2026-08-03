package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/seongmin221/ai-account-manager/internal/config"
	"github.com/seongmin221/ai-account-manager/internal/execution"
	"github.com/seongmin221/ai-account-manager/internal/providers"
)

func (c *CLI) runDoctor() error {
	document, err := c.Config.Load()
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintln(c.Out, "config: valid")
	_, _ = fmt.Fprintf(c.Out, "config permissions: %s\n", c.permissionStatus())
	_, _ = fmt.Fprintf(c.Out, "zsh wrapper: %s\n", c.wrapperStatus())
	_, _ = fmt.Fprintf(c.Out, "gh: %s\n", commandStatus(c.Runner, "gh"))
	_, _ = fmt.Fprintf(c.Out, "codex: %s\n", commandStatus(c.Runner, "codex"))

	registry, github, codex := c.providerRegistry()
	for _, profileID := range sortedProfiles(document) {
		profile := document.Profiles[profileID]
		for _, providerID := range sortedProfileProviders(profile) {
			settings := profile.Providers[providerID]
			provider, registered := registry.Get(providerID)
			if !registered {
				_, _ = fmt.Fprintf(c.Out, "%s/%s: unknown provider\n", profileID, providerID)
				continue
			}
			if err := provider.Validate(context.Background(), settings); err != nil {
				_, _ = fmt.Fprintf(c.Out, "%s/%s: registered / invalid (%s)\n", profileID, providerID, safeDiagnostic(err))
				continue
			}
			switch providerID {
			case "github":
				_, _ = fmt.Fprintf(c.Out, "%s/%s: registered / credential available / %s\n", profileID, providerID, c.githubAuthStatus(settings["host"]))
			case "codex":
				_, _ = fmt.Fprintf(c.Out, "%s/%s: registered / credential available / %s\n", profileID, providerID, c.codexAuthStatus(codex, settings["codex_home"]))
			default:
				_, _ = fmt.Fprintf(c.Out, "%s/%s: registered / configuration valid\n", profileID, providerID)
			}
		}
	}
	_ = github
	return nil
}

func (c *CLI) permissionStatus() string {
	fileInfo, fileErr := os.Stat(c.Config.Path)
	dirInfo, dirErr := os.Stat(filepath.Dir(c.Config.Path))
	if fileErr != nil || dirErr != nil {
		return "unavailable"
	}
	if fileInfo.Mode().Perm() != 0o600 || dirInfo.Mode().Perm() != 0o700 {
		return "fix required"
	}
	return "secure"
}

func (c *CLI) wrapperStatus() string {
	path := c.ZshrcPath
	if path == "" {
		path = defaultZshrcPath()
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		return "not installed"
	}
	if strings.Contains(string(contents), zshWrapperStart) && strings.Contains(string(contents), zshWrapperEnd) {
		return "installed"
	}
	return "not installed"
}

func commandStatus(runner execution.CommandRunner, name string) string {
	if runner.Exists(context.Background(), name) {
		return "available"
	}
	return "unavailable"
}

func (c *CLI) githubAuthStatus(host string) string {
	if host == "" || !c.Runner.Exists(context.Background(), "gh") {
		return "auth unavailable"
	}
	result, err := c.Runner.Run(context.Background(), execution.Command{
		Name:  "gh",
		Args:  []string{"auth", "status", "--hostname", host},
		Unset: []string{"GH_TOKEN", "GITHUB_TOKEN", "GH_ENTERPRISE_TOKEN", "GITHUB_ENTERPRISE_TOKEN"},
	})
	if err != nil || result.ExitCode != 0 {
		return "auth unavailable"
	}
	return "auth valid"
}

func (c *CLI) codexAuthStatus(provider *providers.CodexProvider, home string) string {
	if home == "" || !c.Runner.Exists(context.Background(), "codex") {
		return "auth unavailable"
	}
	result, err := c.Runner.Run(context.Background(), execution.Command{
		Name: "codex",
		Args: []string{"login", "status"},
		Env:  map[string]string{"CODEX_HOME": home},
	})
	if err != nil || result.ExitCode != 0 {
		return "auth unavailable"
	}
	_ = provider
	return "auth valid"
}

func safeDiagnostic(err error) string {
	message := err.Error()
	message = strings.ReplaceAll(message, "\n", " ")
	message = strings.ReplaceAll(message, "\r", " ")
	return message
}

func sortedProfiles(document config.Document) []config.ProfileID {
	ids := make([]config.ProfileID, 0, len(document.Profiles))
	for id := range document.Profiles {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func sortedProfileProviders(profile config.Profile) []config.ProviderID {
	ids := make([]config.ProviderID, 0, len(profile.Providers))
	for id := range profile.Providers {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}
