package app

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/seongmin221/ai-account-manager/internal/config"
	"github.com/seongmin221/ai-account-manager/internal/credentials"
	"github.com/seongmin221/ai-account-manager/internal/execution"
)

// CLI contains the application dependencies so read-only commands can be
// tested without touching the real home directory or external tools.
type CLI struct {
	Config      *config.FileStore
	Runner      execution.CommandRunner
	Credentials credentials.Store
	ManagedRoot string
	In          io.Reader
	ZshrcPath   string
	Out         io.Writer
	ErrOut      io.Writer
}

func NewDefaultCLI() *CLI {
	return &CLI{
		Config:      config.NewFileStore(config.DefaultPath("")),
		Runner:      execution.OSRunner{},
		Credentials: credentials.NewDefaultStore(),
		In:          os.Stdin,
		ZshrcPath:   defaultZshrcPath(),
		Out:         os.Stdout,
		ErrOut:      os.Stderr,
	}
}

func NewCLI(store *config.FileStore, runner execution.CommandRunner, out, errOut io.Writer) *CLI {
	if runner == nil {
		runner = execution.OSRunner{}
	}
	if store == nil {
		store = config.NewFileStore(config.DefaultPath(""))
	}
	if out == nil {
		out = io.Discard
	}
	if errOut == nil {
		errOut = io.Discard
	}
	return &CLI{
		Config:      store,
		Runner:      runner,
		Credentials: credentials.NewMemoryStore(),
		In:          os.Stdin,
		ZshrcPath:   defaultZshrcPath(),
		Out:         out,
		ErrOut:      errOut,
	}
}

// Run executes the command and returns its documented process exit code.
func Run(args []string) int {
	return NewDefaultCLI().Run(args)
}

func (c *CLI) Run(args []string) int {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		c.printHelp()
		return 0
	}

	var err error
	switch args[0] {
	case "init":
		err = c.initConfig(args[1:])
	case "current":
		err = c.current()
	case "list":
		err = c.list()
	case "doctor":
		err = c.doctor()
	case "add":
		err = c.add(args[1:])
	case "change":
		err = c.change(args[1:])
	case "env":
		err = c.env(args[1:])
	case "config":
		if len(args) != 2 || args[1] != "validate" {
			return c.usage("usage: account-manager config validate")
		}
		err = c.validateConfig()
	default:
		return c.usage(fmt.Sprintf("unknown command %q", args[0]))
	}
	if err == nil {
		return 0
	}
	_, _ = fmt.Fprintln(c.ErrOut, err)
	return errorExitCode(err)
}

func (c *CLI) initConfig(args []string) error {
	if len(args) > 1 || (len(args) == 1 && args[0] != "zsh") {
		return newUsageError("usage: account-manager init [zsh]")
	}
	if _, err := os.Stat(c.Config.Path); err == nil {
		_, _ = fmt.Fprintf(c.Out, "Configuration already exists: %s\n", c.Config.Path)
		if len(args) == 1 {
			return c.configureZshIntegration()
		}
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := c.Config.Save(config.NewDocument()); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(c.Out, "Initialized configuration: %s\n", c.Config.Path)
	if len(args) == 1 {
		return c.configureZshIntegration()
	}
	return nil
}

func (c *CLI) validateConfig() error {
	document, err := c.Config.Load()
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintln(c.Out, "Configuration is valid.")
	_, _ = fmt.Fprintf(c.Out, "Profiles: %d\n", len(document.Profiles))
	_, _ = fmt.Fprintf(c.Out, "Providers: %s\n", stringsOrNone(document.ProviderIDs()))
	_, _ = fmt.Fprintf(c.Out, "Active: %s\n", activeSummary(document))
	return nil
}

func (c *CLI) current() error {
	document, err := c.Config.Load()
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(c.Out, "Overall mode: %s\n", document.OverallMode())
	for _, providerID := range document.ProviderIDs() {
		profileID := document.Active[providerID]
		settings := document.Profiles[profileID].Providers[providerID]
		_, _ = fmt.Fprintf(c.Out, "\n%s\n  profile: %s\n", providerID, profileID)
		switch providerID {
		case "github":
			_, _ = fmt.Fprintf(c.Out, "  host: %s\n  account: %s\n  source: %s\n", settings["host"], settings["account"], settings["auth_source"])
		case "codex":
			available := "missing"
			if ref := settings["credential_ref"]; ref != "" {
				if exists, err := c.Credentials.Exists(context.Background(), ref); err == nil && exists {
					available = "available"
				}
			}
			_, _ = fmt.Fprintf(c.Out, "  home: %s\n  auth: %s\n", settings["codex_home"], available)
		}
	}
	return nil
}

func (c *CLI) list() error {
	document, err := c.Config.Load()
	if err != nil {
		return err
	}
	profileIDs := make([]config.ProfileID, 0, len(document.Profiles))
	for profileID := range document.Profiles {
		profileIDs = append(profileIDs, profileID)
	}
	sortProfileIDs(profileIDs)
	for _, profileID := range profileIDs {
		profile := document.Profiles[profileID]
		providerIDs := make([]config.ProviderID, 0, len(profile.Providers))
		for providerID := range profile.Providers {
			providerIDs = append(providerIDs, providerID)
		}
		sortProviderIDs(providerIDs)
		_, _ = fmt.Fprintf(c.Out, "%s", profileID)
		if profile.DisplayName != "" {
			_, _ = fmt.Fprintf(c.Out, " (%s)", profile.DisplayName)
		}
		_, _ = fmt.Fprintf(c.Out, ": %s\n", stringsOrNone(providerIDs))
	}
	return nil
}

func (c *CLI) doctor() error {
	return c.runDoctor()
}

func (c *CLI) printHelp() {
	_, _ = fmt.Fprintln(c.Out, "account-manager: profile-based account switcher")
	_, _ = fmt.Fprintln(c.Out, "usage: account-manager <init|current|list|doctor|add|change|env|config validate>")
}

func (c *CLI) printZshIntegration() {
	_, _ = fmt.Fprintln(c.Out, "Add this wrapper to ~/.zshrc:")
	_, _ = fmt.Fprint(c.Out, zshWrapperSnippet())
}

func (c *CLI) usage(message string) int {
	_, _ = fmt.Fprintln(c.ErrOut, message)
	return 2
}
