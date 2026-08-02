package app

import "fmt"

// Run executes the account-manager command and returns a process exit code.
// Command behavior will be implemented incrementally by provider and storage
// packages while keeping the single-binary entrypoint stable.
func Run(args []string) int {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		fmt.Println("account-manager: profile-based account switcher")
		return 0
	}

	fmt.Printf("account-manager: command %q is not implemented yet\n", args[0])
	return 2
}
