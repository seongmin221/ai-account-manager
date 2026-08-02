//go:build darwin

package credentials

import "testing"

func TestMacOSKeychainStoreUsesStableDefaultService(t *testing.T) {
	store := NewMacOSKeychainStore("")
	if store.service != DefaultKeychainService {
		t.Fatalf("service = %q, want %q", store.service, DefaultKeychainService)
	}
}
