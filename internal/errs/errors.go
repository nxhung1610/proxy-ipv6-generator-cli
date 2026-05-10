package errs

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// WrapFileError adds "permission denied" context to file operation errors.
func WrapFileError(action, path string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, os.ErrPermission) || strings.Contains(strings.ToLower(err.Error()), "permission denied") {
		return fmt.Errorf("%s: permission denied: %s", action, path)
	}
	return fmt.Errorf("%s: %w", action, err)
}

// IsPermissionDenied checks if an error is a permission denied error.
func IsPermissionDenied(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, os.ErrPermission) || strings.Contains(strings.ToLower(err.Error()), "permission denied")
}
