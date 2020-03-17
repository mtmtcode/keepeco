package keychain

import (
	"errors"

	"github.com/zalando/go-keyring"
)

var (
	// ErrorItemNotFound is an error occurs when no matching item found in the Keychain
	ErrorItemNotFound = errors.New("No matching item found in the Keychain")

	// ErrorUnsupportedPlatform is an error used when a platform is unsupported
	ErrorUnsupportedPlatform = errors.New("This platform in unsupported")
)

var defaultAccountName = "keepeco"

// GetData returns password of specified service from the keychain
func GetData(serviceName string) (string, error) {
	password, err := keyring.Get(serviceName, defaultAccountName)
	if err == keyring.ErrNotFound {
		return "", ErrorItemNotFound
	}
	if err == keyring.ErrUnsupportedPlatform {
		return "", ErrorUnsupportedPlatform
	}
	return password, err
}

// Save saves the password of specified service to the keychain
func Save(serviceName string, password string) error {
	err := keyring.Set(serviceName, defaultAccountName, password)
	if err == keyring.ErrUnsupportedPlatform {
		return ErrorUnsupportedPlatform
	}
	return err
}
