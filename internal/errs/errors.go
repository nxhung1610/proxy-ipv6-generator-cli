// Package errs defines typed errors for proxy-ipv6-generator-cli.
package errs

import "errors"

// Sentinel errors for common failure conditions.
// These allow callers to use errors.Is() for precise error checking.
var (
	// ErrNoPrefix indicates the IPv6 prefix is required but not provided.
	ErrNoPrefix = errors.New("ipv6_prefix_required")

	// ErrInvalidPrefix indicates the provided IPv6 prefix is malformed.
	ErrInvalidPrefix = errors.New("invalid_ipv6_prefix")

	// Err3proxyNotFound indicates the 3proxy binary could not be located.
	Err3proxyNotFound = errors.New("3proxy_binary_not_found")

	// ErrPortInUse indicates the requested port is already bound.
	ErrPortInUse = errors.New("port_already_in_use")

	// ErrPoolNotFound indicates the specified pool does not exist.
	ErrPoolNotFound = errors.New("pool_not_found")

	// ErrPoolAlreadyRunning indicates the pool is already running.
	ErrPoolAlreadyRunning = errors.New("pool_already_running")

	// ErrPoolExists indicates a pool with the given name already exists.
	ErrPoolExists = errors.New("pool_already_exists")

	// ErrInvalidPort indicates the port number is out of valid range.
	ErrInvalidPort = errors.New("invalid_port_range")

	// ErrPermDenied indicates insufficient permissions to perform the operation.
	ErrPermDenied = errors.New("permission_denied")

	// ErrConfigNotFound indicates the config file could not be located.
	ErrConfigNotFound = errors.New("config_file_not_found")

	// ErrConfigInvalid indicates the config file exists but is malformed.
	ErrConfigInvalid = errors.New("config_invalid")

	// ErrCredentialNotFound indicates the credential reference could not be resolved.
	ErrCredentialNotFound = errors.New("credential_not_found")
)
