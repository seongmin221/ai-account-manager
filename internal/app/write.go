package app

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/seongmin221/ai-account-manager/internal/config"
	"github.com/seongmin221/ai-account-manager/internal/errors"
	"github.com/seongmin221/ai-account-manager/internal/execution"
	"github.com/seongmin221/ai-account-manager/internal/providers"
)

type writeOptions struct {
	profile  config.ProfileID
	scope    providers.Scope
	host     string
	activate bool
	shell    string
}

func parseWriteOptions(args []string) (writeOptions, error) {
	var options writeOptions
	var selected []config.ProviderID
	for index := 0; index < len(args); index++ {
		switch args[index] {
		case "--work", "--personal":
			profile := config.ProfileID(strings.TrimPrefix(args[index], "--"))
			if options.profile != "" && options.profile != profile {
				return writeOptions{}, fmt.Errorf("only one target profile may be selected")
			}
			options.profile = profile
		case "--profile":
			if index+1 >= len(args) {
				return writeOptions{}, fmt.Errorf("--profile requires a value")
			}
			index++
			profile := config.ProfileID(args[index])
			if options.profile != "" && options.profile != profile {
				return writeOptions{}, fmt.Errorf("only one target profile may be selected")
			}
			options.profile = profile
		case "--github":
			selected = append(selected, "github")
		case "--codex":
			selected = append(selected, "codex")
		case "--only":
			if index+1 >= len(args) {
				return writeOptions{}, fmt.Errorf("--only requires a comma-separated provider list")
			}
			index++
			for _, value := range strings.Split(args[index], ",") {
				if value == "" {
					return writeOptions{}, fmt.Errorf("--only contains an empty provider")
				}
				selected = append(selected, config.ProviderID(value))
			}
		case "--host":
			if index+1 >= len(args) {
				return writeOptions{}, fmt.Errorf("--host requires a value")
			}
			index++
			options.host = args[index]
		case "--activate":
			options.activate = true
		case "--shell":
			if index+1 >= len(args) {
				return writeOptions{}, fmt.Errorf("--shell requires a value")
			}
			index++
			options.shell = args[index]
		default:
			return writeOptions{}, fmt.Errorf("unknown option %q", args[index])
		}
	}
	if options.profile == "" {
		return writeOptions{}, fmt.Errorf("a target profile is required")
	}
	if err := options.profile.Validate(); err != nil {
		return writeOptions{}, err
	}
	scope, err := providers.NewScope(selected...)
	if err != nil {
		return writeOptions{}, err
	}
	options.scope = scope
	return options, nil
}

func (c *CLI) add(args []string) error {
	options, err := parseWriteOptions(args)
	if err != nil {
		return newUsageError(err.Error())
	}
	if options.shell != "" {
		return apperrors.New(apperrors.ProviderConfigInvalid, "--shell is supported only by change")
	}
	document, err := c.loadForAdd()
	if err != nil {
		return err
	}
	registry, github, _ := c.providerRegistry()
	profile := document.Profiles[options.profile]
	if profile.Providers == nil {
		profile.Providers = make(map[config.ProviderID]config.ProviderConfig)
	}
	selected, err := options.scope.Resolve(registry.IDs())
	if err != nil {
		return apperrors.New(apperrors.UnknownProvider, err.Error())
	}
	for _, providerID := range selected {
		provider, _ := registry.Get(providerID)
		if providerID == "github" {
			if options.host != "" {
				github.CurrentHost = options.host
			} else if existing := profile.Providers[providerID]["host"]; existing != "" {
				github.CurrentHost = existing
			} else {
				return apperrors.New(apperrors.GitHubHostRequired, "GitHub host is required for the first registration")
			}
		}
		settings, err := provider.Capture(context.Background(), options.profile)
		if err != nil {
			return apperrors.New(apperrors.ProviderConfigInvalid, fmt.Sprintf("provider %q registration failed: %v", providerID, err))
		}
		profile.Providers[providerID] = settings
		if _, active := document.Active[providerID]; !active {
			if document.Active == nil {
				document.Active = make(map[config.ProviderID]config.ProfileID)
			}
			document.Active[providerID] = options.profile
		}
	}
	document.Profiles[options.profile] = profile
	if err := c.Config.Save(document); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(c.Out, "Registered profile %q providers: %s\n", options.profile, stringsOrNone(selected))
	if options.activate {
		return c.changeWithOptions(options)
	}
	return nil
}

func (c *CLI) change(args []string) error {
	options, err := parseWriteOptions(args)
	if err != nil {
		return newUsageError(err.Error())
	}
	if options.host != "" || options.activate {
		return apperrors.New(apperrors.ProviderConfigInvalid, "--host and --activate are supported only by add")
	}
	if options.shell != "" && options.shell != "zsh" {
		return apperrors.New(apperrors.ProviderConfigInvalid, fmt.Sprintf("unsupported shell %q", options.shell))
	}
	return c.changeWithOptions(options)
}

func (c *CLI) changeWithOptions(options writeOptions) error {
	document, err := c.Config.Load()
	if err != nil {
		return err
	}
	registry, _, _ := c.providerRegistry()
	updated, result, err := (providers.Transaction{Registry: registry}).Change(context.Background(), document, options.profile, options.scope)
	if err != nil {
		return err
	}
	if err := c.Config.Save(updated); err != nil {
		if rollbackErr := result.Rollback(context.Background(), registry); rollbackErr != nil {
			return rollbackErr
		}
		return err
	}
	_, _ = fmt.Fprintf(c.Out, "Changed profile %q providers: %s\n", options.profile, stringsOrNone(result.Selected))
	if options.shell == "zsh" {
		_, _ = fmt.Fprint(c.Out, renderProviderPatch(updated, result.Selected))
	} else {
		_, _ = fmt.Fprintln(c.Out, "Warning: current shell environment was not patched; use the zsh wrapper when shell integration is installed.")
	}
	return nil
}

func (c *CLI) env(args []string) error {
	if len(args) != 2 || args[0] != "--shell" || args[1] != "zsh" {
		return newUsageError("usage: account-manager env --shell zsh")
	}
	document, err := c.Config.Load()
	if err != nil {
		return err
	}
	_, _ = fmt.Fprint(c.Out, renderProviderPatch(document, document.ProviderIDs()))
	return nil
}

func renderProviderPatch(document config.Document, providerIDs []config.ProviderID) string {
	patch := execution.NewEnvPatch()
	for _, providerID := range providerIDs {
		profileID, ok := document.Active[providerID]
		if !ok {
			continue
		}
		settings, ok := document.Profiles[profileID].Providers[providerID]
		if !ok {
			continue
		}
		switch providerID {
		case "github":
			if host := settings["host"]; host != "" {
				patch.SetValue("GH_HOST", host)
			}
			patch.UnsetValue("GH_TOKEN", "GITHUB_TOKEN", "GH_ENTERPRISE_TOKEN", "GITHUB_ENTERPRISE_TOKEN")
		case "codex":
			if home := settings["codex_home"]; home != "" {
				patch.SetValue("CODEX_HOME", home)
			}
		}
	}
	return patch.RenderZsh()
}

func (c *CLI) loadForAdd() (config.Document, error) {
	document, err := c.Config.Load()
	if err == nil {
		return document, nil
	}
	var coded *apperrors.Error
	if errors.As(err, &coded) && coded.Code == apperrors.ConfigNotFound {
		return config.NewDocument(), nil
	}
	return config.Document{}, err
}

func (c *CLI) providerRegistry() (providers.Registry, *providers.GitHubProvider, *providers.CodexProvider) {
	registry := providers.NewMemoryRegistry()
	github := providers.NewGitHubProvider(c.Runner)
	codex := providers.NewCodexProvider(c.Runner, c.Credentials)
	if c.ManagedRoot != "" {
		codex.ManagedRoot = c.ManagedRoot
	}
	_ = registry.Register(github)
	_ = registry.Register(codex)
	return registry, github, codex
}
