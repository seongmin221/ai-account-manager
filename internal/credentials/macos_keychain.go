//go:build darwin

package credentials

/*
#cgo darwin LDFLAGS: -framework Security -framework CoreFoundation
#include <CoreFoundation/CoreFoundation.h>
#include <Security/Security.h>
#include <stdlib.h>
#include <string.h>

static CFStringRef am_string(const char *value) {
	return CFStringCreateWithCString(NULL, value, kCFStringEncodingUTF8);
}

static CFMutableDictionaryRef am_query(const char *service, const char *account) {
	CFMutableDictionaryRef query = CFDictionaryCreateMutable(
		NULL, 0, &kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);
	CFStringRef service_value = am_string(service);
	CFStringRef account_value = am_string(account);
	CFDictionarySetValue(query, kSecClass, kSecClassGenericPassword);
	CFDictionarySetValue(query, kSecAttrService, service_value);
	CFDictionarySetValue(query, kSecAttrAccount, account_value);
	CFDictionarySetValue(query, kSecUseDataProtectionKeychain, kCFBooleanTrue);
	CFRelease(service_value);
	CFRelease(account_value);
	return query;
}

static OSStatus am_put(const char *service, const char *account, const void *bytes, size_t length) {
	CFMutableDictionaryRef query = am_query(service, account);
	CFDataRef value = CFDataCreate(NULL, bytes, (CFIndex)length);
	CFMutableDictionaryRef attributes = CFDictionaryCreateMutable(
		NULL, 0, &kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);
	CFDictionarySetValue(attributes, kSecValueData, value);
	OSStatus status = SecItemUpdate(query, attributes);
	if (status == errSecItemNotFound) {
		CFDictionarySetValue(query, kSecValueData, value);
		status = SecItemAdd(query, NULL);
	}
	CFRelease(attributes);
	CFRelease(value);
	CFRelease(query);
	return status;
}

static OSStatus am_get(const char *service, const char *account, void **out_bytes, size_t *out_length) {
	CFMutableDictionaryRef query = am_query(service, account);
	CFDictionarySetValue(query, kSecReturnData, kCFBooleanTrue);
	CFDictionarySetValue(query, kSecMatchLimit, kSecMatchLimitOne);
	CFTypeRef result = NULL;
	OSStatus status = SecItemCopyMatching(query, &result);
	CFRelease(query);
	if (status != errSecSuccess) {
		return status;
	}
	if (result == NULL || CFGetTypeID(result) != CFDataGetTypeID()) {
		if (result != NULL) {
			CFRelease(result);
		}
		return errSecDecode;
	}
	CFDataRef data = (CFDataRef)result;
	CFIndex length = CFDataGetLength(data);
	void *copy = NULL;
	if (length > 0) {
		copy = malloc((size_t)length);
		if (copy == NULL) {
			CFRelease(result);
			return errSecAllocate;
		}
		memcpy(copy, CFDataGetBytePtr(data), (size_t)length);
	}
	*out_bytes = copy;
	*out_length = (size_t)length;
	CFRelease(result);
	return errSecSuccess;
}

static OSStatus am_delete(const char *service, const char *account) {
	CFMutableDictionaryRef query = am_query(service, account);
	OSStatus status = SecItemDelete(query);
	CFRelease(query);
	return status;
}

static OSStatus am_item_not_found(void) { return errSecItemNotFound; }
*/
import "C"

import (
	"context"
	"fmt"
	"unsafe"
)

const DefaultKeychainService = "com.seongmin221.ai-account-manager"

// MacOSKeychainStore stores generic-password items in the user's data
// protection Keychain. It intentionally exposes only opaque bytes to callers.
type MacOSKeychainStore struct {
	service string
}

func NewMacOSKeychainStore(service string) *MacOSKeychainStore {
	if service == "" {
		service = DefaultKeychainService
	}
	return &MacOSKeychainStore{service: service}
}

func (s *MacOSKeychainStore) Put(ctx context.Context, ref string, secret []byte) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	service, account := cStrings(s.service, ref)
	defer freeCString(service)
	defer freeCString(account)
	var data unsafe.Pointer
	if len(secret) > 0 {
		data = C.CBytes(secret)
		defer C.free(data)
	}
	status := C.am_put(service, account, data, C.size_t(len(secret)))
	if status != 0 {
		return keychainError("put", ref, status)
	}
	return nil
}

func (s *MacOSKeychainStore) Get(ctx context.Context, ref string) ([]byte, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	service, account := cStrings(s.service, ref)
	defer freeCString(service)
	defer freeCString(account)
	var data unsafe.Pointer
	var length C.size_t
	status := C.am_get(service, account, &data, &length)
	if status == C.am_item_not_found() {
		return nil, ErrNotFound
	}
	if status != 0 {
		return nil, keychainError("get", ref, status)
	}
	if data == nil || length == 0 {
		return []byte{}, nil
	}
	defer C.free(data)
	secret := C.GoBytes(data, C.int(length))
	return secret, nil
}

func (s *MacOSKeychainStore) Exists(ctx context.Context, ref string) (bool, error) {
	_, err := s.Get(ctx, ref)
	if err == nil {
		return true, nil
	}
	if err == ErrNotFound {
		return false, nil
	}
	return false, err
}

func (s *MacOSKeychainStore) Delete(ctx context.Context, ref string) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	service, account := cStrings(s.service, ref)
	defer freeCString(service)
	defer freeCString(account)
	status := C.am_delete(service, account)
	if status != 0 && status != C.am_item_not_found() {
		return keychainError("delete", ref, status)
	}
	return nil
}

func cStrings(service, account string) (*C.char, *C.char) {
	return C.CString(service), C.CString(account)
}

func freeCString(value *C.char) { C.free(unsafe.Pointer(value)) }

func keychainError(action, ref string, status C.OSStatus) error {
	return fmt.Errorf("keychain %s failed for credential %q (OSStatus %d)", action, ref, int32(status))
}
