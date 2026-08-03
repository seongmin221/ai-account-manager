package providers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/seongmin221/ai-account-manager/internal/config"
	"github.com/seongmin221/ai-account-manager/internal/credentials"
	"github.com/seongmin221/ai-account-manager/internal/execution"
)

const codexProviderID config.ProviderID = "codex"

type CodexProvider struct {
	Runner      execution.CommandRunner
	Credentials credentials.Store
	CurrentHome string
	ManagedRoot string
}

func NewCodexProvider(runner execution.CommandRunner, store credentials.Store) *CodexProvider {
	home, _ := os.UserHomeDir()
	currentHome := os.Getenv("CODEX_HOME")
	if currentHome == "" {
		currentHome = filepath.Join(home, ".codex")
	}
	return &CodexProvider{
		Runner:      runner,
		Credentials: store,
		CurrentHome: currentHome,
		ManagedRoot: filepath.Join(home, ".local", "share", "account-manager", "codex"),
	}
}

func (p *CodexProvider) ID() config.ProviderID { return codexProviderID }

// Login starts the interactive Codex login flow in a profile-managed home.
func (p *CodexProvider) Login(ctx context.Context, target config.ProfileID) error {
	home := filepath.Join(p.ManagedRoot, string(target))
	result, err := p.Runner.Run(ctx, execution.Command{
		Name:        "codex",
		Args:        []string{"login"},
		Env:         map[string]string{"CODEX_HOME": home},
		Interactive: true,
	})
	if err != nil {
		return fmt.Errorf("cannot start Codex login: %w", err)
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("Codex login failed with exit code %d", result.ExitCode)
	}
	p.CurrentHome = home
	return nil
}

func (p *CodexProvider) InspectCurrent(ctx context.Context) (Identity, error) {
	home := p.CurrentHome
	if home == "" {
		return Identity{}, fmt.Errorf("CODEX_HOME is not configured")
	}
	if err := p.checkLogin(ctx, home); err != nil {
		return Identity{}, err
	}
	return Identity{Provider: p.ID(), Host: home, Account: "authenticated"}, nil
}

func (p *CodexProvider) Capture(ctx context.Context, target config.ProfileID) (config.ProviderConfig, error) {
	home := p.CurrentHome
	authPath := filepath.Join(home, "auth.json")
	authCache, err := os.ReadFile(authPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("Codex auth cache was not found; run CODEX_HOME=%q codex login", home)
		}
		return nil, fmt.Errorf("cannot read Codex auth cache")
	}
	if err := p.checkLogin(ctx, home); err != nil {
		return nil, err
	}
	ref := "codex/" + string(target)
	if err := p.Credentials.Put(ctx, ref, authCache); err != nil {
		return nil, fmt.Errorf("cannot store Codex credential %q: %w", ref, err)
	}
	return config.ProviderConfig{
		"credential_ref": ref,
		"codex_home":     filepath.Join(p.ManagedRoot, string(target)),
	}, nil
}

func (p *CodexProvider) Validate(ctx context.Context, settings config.ProviderConfig) error {
	ref := settings["credential_ref"]
	home := settings["codex_home"]
	if ref == "" {
		return fmt.Errorf("Codex credential_ref is required")
	}
	if home == "" {
		return fmt.Errorf("Codex codex_home is required")
	}
	path, err := p.expandManagedPath(home)
	if err != nil {
		return err
	}
	if !p.CredentialsExists(ctx, ref) {
		return fmt.Errorf("Codex credential %q is not available", ref)
	}
	if !p.Runner.Exists(ctx, "codex") {
		return fmt.Errorf("codex is not installed")
	}
	if path == "" {
		return fmt.Errorf("Codex home is invalid")
	}
	return nil
}

func (p *CodexProvider) PlanActivate(ctx context.Context, settings config.ProviderConfig) (ActivationPlan, error) {
	if err := p.Validate(ctx, settings); err != nil {
		return ActivationPlan{}, err
	}
	home, err := p.expandManagedPath(settings["codex_home"])
	if err != nil {
		return ActivationPlan{}, err
	}
	authPath := filepath.Join(home, "auth.json")
	previous, existed, err := readOptionalFile(authPath)
	if err != nil {
		return ActivationPlan{}, fmt.Errorf("cannot snapshot Codex auth cache")
	}
	return ActivationPlan{
		Provider: p.ID(),
		Data: codexActivationPlan{
			CredentialRef: settings["credential_ref"],
			Home:          home,
			AuthPath:      authPath,
			Previous:      previous,
			PreviousExist: existed,
		},
	}, nil
}

func (p *CodexProvider) Apply(ctx context.Context, plan ActivationPlan) (RollbackPlan, error) {
	activation, ok := plan.Data.(codexActivationPlan)
	if !ok || plan.Provider != p.ID() {
		return RollbackPlan{}, fmt.Errorf("invalid Codex activation plan")
	}
	authCache, err := p.Credentials.Get(ctx, activation.CredentialRef)
	if err != nil {
		return RollbackPlan{}, fmt.Errorf("cannot read Codex credential %q", activation.CredentialRef)
	}
	if err := writeAtomicSecretFile(activation.AuthPath, authCache); err != nil {
		return RollbackPlan{}, err
	}
	if err := p.checkLogin(ctx, activation.Home); err != nil {
		if restoreErr := restoreFile(activation.AuthPath, activation.Previous, activation.PreviousExist); restoreErr != nil {
			return RollbackPlan{}, fmt.Errorf("Codex login status failed and auth rollback failed")
		}
		return RollbackPlan{}, err
	}
	return RollbackPlan{Provider: p.ID(), Data: activation}, nil
}

func (p *CodexProvider) Rollback(ctx context.Context, plan RollbackPlan) error {
	activation, ok := plan.Data.(codexActivationPlan)
	if !ok || plan.Provider != p.ID() {
		return fmt.Errorf("invalid Codex rollback plan")
	}
	return restoreFile(activation.AuthPath, activation.Previous, activation.PreviousExist)
}

type codexActivationPlan struct {
	CredentialRef string
	Home          string
	AuthPath      string
	Previous      []byte
	PreviousExist bool
}

func (p *CodexProvider) CredentialsExists(ctx context.Context, ref string) bool {
	exists, err := p.Credentials.Exists(ctx, ref)
	return err == nil && exists
}

func (p *CodexProvider) checkLogin(ctx context.Context, home string) error {
	result, err := p.Runner.Run(ctx, execution.Command{
		Name: "codex",
		Args: []string{"login", "status"},
		Env:  map[string]string{"CODEX_HOME": home},
	})
	if err != nil {
		return fmt.Errorf("cannot inspect Codex login status: %w", err)
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("Codex login status is invalid; run CODEX_HOME=%q codex login", home)
	}
	return nil
}

func (p *CodexProvider) expandManagedPath(path string) (string, error) {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("cannot resolve home directory")
		}
		path = filepath.Join(home, strings.TrimPrefix(path, "~/"))
	}
	path, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("Codex home path is invalid")
	}
	root, err := filepath.Abs(p.ManagedRoot)
	if err != nil {
		return "", fmt.Errorf("Codex managed root is invalid")
	}
	relative, err := filepath.Rel(root, path)
	if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("Codex home must be inside the managed account-manager directory")
	}
	return path, nil
}

func readOptionalFile(path string) ([]byte, bool, error) {
	data, err := os.ReadFile(path)
	if err == nil {
		return data, true, nil
	}
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	return nil, false, err
}

func writeAtomicSecretFile(path string, secret []byte) error {
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return fmt.Errorf("cannot create Codex home")
	}
	if err := os.Chmod(directory, 0o700); err != nil {
		return fmt.Errorf("cannot set Codex home permissions")
	}
	temporary, err := os.CreateTemp(directory, ".auth.json.tmp-*")
	if err != nil {
		return fmt.Errorf("cannot create temporary Codex auth cache")
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return fmt.Errorf("cannot set Codex auth cache permissions")
	}
	if _, err := temporary.Write(secret); err != nil {
		temporary.Close()
		return fmt.Errorf("cannot write Codex auth cache")
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return fmt.Errorf("cannot flush Codex auth cache")
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("cannot close Codex auth cache")
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("cannot activate Codex auth cache")
	}
	return nil
}

func restoreFile(path string, data []byte, existed bool) error {
	if !existed {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	return writeAtomicSecretFile(path, data)
}
