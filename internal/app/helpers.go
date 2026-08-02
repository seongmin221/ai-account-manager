package app

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/seongmin221/ai-account-manager/internal/config"
	"github.com/seongmin221/ai-account-manager/internal/errors"
)

type usageError struct{ message string }

func (e *usageError) Error() string { return e.message }

func newUsageError(message string) error { return &usageError{message: message} }

func errorExitCode(err error) int {
	var usage *usageError
	if errors.As(err, &usage) {
		return 2
	}
	var coded *apperrors.Error
	if errors.As(err, &coded) {
		return coded.Code.ExitCode()
	}
	return 7
}

func stringsOrNone[T ~string](values []T) string {
	if len(values) == 0 {
		return "none"
	}
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, string(value))
	}
	return strings.Join(parts, ", ")
}

func activeSummary(document config.Document) string {
	if len(document.Active) == 0 {
		return "none"
	}
	providerIDs := document.ProviderIDs()
	parts := make([]string, 0, len(providerIDs))
	for _, providerID := range providerIDs {
		parts = append(parts, fmt.Sprintf("%s=%s", providerID, document.Active[providerID]))
	}
	return strings.Join(parts, ", ")
}

func sortProfileIDs(ids []config.ProfileID) {
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
}

func sortProviderIDs(ids []config.ProviderID) {
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
}
