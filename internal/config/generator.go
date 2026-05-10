package config

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/nxhung/proxy-ipv6-generator-cli/pkg/types"
)

// Default values for 3proxy configuration.
const (
	DefaultMaxConn     = 500
	DefaultDNSServer   = "8.8.8.8"
	DefaultNSCache     = 65536
	DefaultTimeouts    = "1 5 30 5 180"
)

// threeProxyTemplate is parsed once at init for performance.
var threeProxyTemplate = template.Must(template.New("3proxy").Parse(`maxconn {{ .MaxConn }}
nserver {{ .DNSServer }}
nscache {{ .NSCache }}
timeouts {{ .Timeouts }}
auth strong
{{- if .Proxies }}
users {{ range $i, $p := .Proxies }}{{ if $i }} {{ end }}{{ $p.Username }}:{{ $p.Password }}{{ end }}
{{ end }}{{ range $p := .Proxies }}
allow {{ $p.Username }}
proxy -6 -n{{ $.MaxConn }} -a {{ $p.Address }} {{ $p.Port }} {{ $p.Username }} {{ $p.Password }}
{{ end }}
log /var/log/3proxy.log D
rotate 7
`))

// sanitizeCredential removes characters dangerous for 3proxy user credentials.
// 3proxy user/password cannot contain: \n, \r, :, #, ;, space, or backslash.
func sanitizeCredential(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r < 32 || r == '\\' || r == '\n' || r == '\r' ||
			r == ':' || r == '#' || r == ';' || r == ' ':
			return -1
		default:
			return r
		}
	}, s)
}

// sanitizeAddress removes control characters and backslash from addresses.
// IPv6 addresses legitimately contain colons, so colons are preserved.
func sanitizeAddress(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r < 32 || r == '\\' || r == '\n' || r == '\r':
			return -1
		default:
			return r
		}
	}, s)
}

// sanitizeDNSServer removes newlines and control characters from DNS server string.
func sanitizeDNSServer(s string) string {
	return strings.Map(func(r rune) rune {
		if r < 32 || r == '\n' || r == '\r' {
			return -1
		}
		return r
	}, s)
}

type ProxyData struct {
	Username string
	Password string
	Address  string
	Port     int
	Protocol string
}

func Generate3proxyConfig(pool types.Pool, proxies []types.Proxy, maxConn int, dnssrv string) (string, error) {
	if maxConn <= 0 {
		maxConn = DefaultMaxConn
	}

	// Sanitize DNSServer to prevent config injection
	dnssrv = sanitizeDNSServer(dnssrv)
	if dnssrv == "" {
		dnssrv = DefaultDNSServer
	}

	sanitizedProxies := make([]ProxyData, len(proxies))
	for i, p := range proxies {
		sanitizedProxies[i] = ProxyData{
			Username: sanitizeCredential(p.Username),
			Password: sanitizeCredential(p.Password),
			Address:  sanitizeAddress(p.Address),
			Port:     p.Port,
			Protocol: string(p.Protocol),
		}
	}

	data := map[string]interface{}{
		"MaxConn":   maxConn,
		"DNSServer": dnssrv,
		"NSCache":   DefaultNSCache,
		"Timeouts":  DefaultTimeouts,
		"Proxies":   sanitizedProxies,
	}

	var buf bytes.Buffer
	if err := threeProxyTemplate.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// generateIPv6FromPrefix generates an IPv6 address from a prefix and index.
// It handles prefixes with or without trailing colons.
func generateIPv6FromPrefix(prefix string, index int) string {
	// Remove trailing colons from prefix
	for len(prefix) > 0 && prefix[len(prefix)-1] == ':' {
		prefix = prefix[:len(prefix)-1]
	}
	return fmt.Sprintf("%s:%04x::1", prefix, index)
}

// PoolToProxies converts a pool to a list of ProxyData for testing.
// It uses default credentials and sanitizes all generated values.
func PoolToProxies(pool types.Pool) []ProxyData {
	proxies := make([]ProxyData, 0, pool.Count)
	for i := 0; i < pool.Count; i++ {
		addr := generateIPv6FromPrefix(pool.Prefix, i)
		proxies = append(proxies, ProxyData{
			Username: sanitizeCredential("user"),
			Password: sanitizeCredential("pass"),
			Address:  sanitizeAddress(addr),
			Port:     pool.BasePort + i,
			Protocol: string(pool.Protocol),
		})
	}
	return proxies
}
