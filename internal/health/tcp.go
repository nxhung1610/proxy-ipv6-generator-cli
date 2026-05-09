package health

import (
	"context"
	"fmt"
	"net"
	"time"
)

// DialProxy establishes a TCP connection to a proxy endpoint.
// Returns the connection or an error with address context.
func DialProxy(ctx context.Context, addr string, timeout time.Duration) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", addr, err)
	}
	return conn, nil
}
