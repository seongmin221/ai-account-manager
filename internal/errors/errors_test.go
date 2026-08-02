package apperrors

import "testing"

func TestErrorFormattingAndExitCode(t *testing.T) {
	err := New(CredentialMissing, "credential is unavailable")
	if got, want := err.Error(), "error[AM010]: credential is unavailable"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
	if got, want := CredentialMissing.ExitCode(), 4; got != want {
		t.Fatalf("ExitCode() = %d, want %d", got, want)
	}
}
