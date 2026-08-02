// Package apperrors contains stable application error codes and exit-code
// mapping shared by the CLI and its adapters.
package apperrors

import "fmt"

type Code string

const (
	ConfigNotFound           Code = "AM001"
	ConfigSyntax             Code = "AM002"
	UnsupportedConfigVersion Code = "AM003"
	InvalidProfileID         Code = "AM004"
	InvalidProviderID        Code = "AM005"
	InvalidActiveReference   Code = "AM006"
	ProviderConfigInvalid    Code = "AM007"
	UnknownProvider          Code = "AM008"
	CredentialReference      Code = "AM009"
	CredentialMissing        Code = "AM010"
	ExternalToolMissing      Code = "AM011"
	ExternalAuthInvalid      Code = "AM012"
	ActivationFailed         Code = "AM013"
	RollbackFailed           Code = "AM014"
	ShellPatchRequired       Code = "AM015"
	PermissionError          Code = "AM016"
	GitHubHostRequired       Code = "AM017"
)

// Error is an actionable, machine-readable application error.
type Error struct {
	Code    Code
	Message string
	Cause   error
}

func (e *Error) Error() string {
	if e.Message == "" {
		return string(e.Code)
	}
	return fmt.Sprintf("error[%s]: %s", e.Code, e.Message)
}

func (e *Error) Unwrap() error { return e.Cause }

func New(code Code, message string) *Error {
	return &Error{Code: code, Message: message}
}

// ExitCode maps the documented application error classes to process status.
func (c Code) ExitCode() int {
	switch c {
	case ConfigNotFound, ConfigSyntax, UnsupportedConfigVersion,
		InvalidProfileID, InvalidProviderID, InvalidActiveReference,
		ProviderConfigInvalid, UnknownProvider:
		return 3
	case CredentialReference, CredentialMissing:
		return 4
	case ActivationFailed:
		return 5
	case RollbackFailed:
		return 6
	case ExternalToolMissing, ExternalAuthInvalid, PermissionError:
		return 7
	case ShellPatchRequired:
		return 15
	default:
		return 1
	}
}
