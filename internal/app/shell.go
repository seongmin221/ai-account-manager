package app

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	zshWrapperStart = "# >>> account-manager zsh integration >>>"
	zshWrapperEnd   = "# <<< account-manager zsh integration <<<"
)

func defaultZshrcPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".zshrc"
	}
	return filepath.Join(home, ".zshrc")
}

func zshWrapperSnippet() string {
	return zshWrapperStart + "\n" +
		"account-manager() {\n" +
		"  if [[ \"$1\" == \"change\" ]]; then\n" +
		"    eval \"$(command account-manager \"$@\" --shell zsh)\" || return\n" +
		"  else\n" +
		"    command account-manager \"$@\"\n" +
		"  fi\n" +
		"}\n" +
		zshWrapperEnd + "\n"
}

func (c *CLI) configureZshIntegration() error {
	path := c.ZshrcPath
	if path == "" {
		path = defaultZshrcPath()
	}
	contents, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("cannot read zsh config %q", path)
	}
	if strings.Contains(string(contents), zshWrapperStart) || strings.Contains(string(contents), zshWrapperEnd) {
		_, _ = fmt.Fprintf(c.Out, "zsh wrapper is already installed in %s\n", path)
		return nil
	}
	if !c.confirm(fmt.Sprintf("Install account-manager wrapper in %s? [y/N] ", path)) {
		_, _ = fmt.Fprintln(c.Out, "zsh wrapper was not installed.")
		c.printZshIntegration()
		return nil
	}
	if len(contents) > 0 && contents[len(contents)-1] != '\n' {
		contents = append(contents, '\n')
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("cannot open zsh config %q", path)
	}
	if len(contents) == 0 {
		_, err = io.WriteString(file, zshWrapperSnippet())
	} else {
		_, err = io.WriteString(file, "\n"+zshWrapperSnippet())
	}
	closeErr := file.Close()
	if err != nil {
		return fmt.Errorf("cannot install zsh wrapper")
	}
	if closeErr != nil {
		return fmt.Errorf("cannot close zsh config")
	}
	_, _ = fmt.Fprintf(c.Out, "Installed zsh wrapper in %s\n", path)
	return nil
}

func (c *CLI) confirm(prompt string) bool {
	if c.In == nil {
		return false
	}
	_, _ = fmt.Fprint(c.Out, prompt)
	answer, err := bufio.NewReader(c.In).ReadString('\n')
	if err != nil && len(answer) == 0 {
		return false
	}
	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer == "y" || answer == "yes"
}
