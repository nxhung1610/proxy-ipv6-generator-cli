package generator

import (
	"fmt"
	"math/big"
	"net"
	"strings"

	"github.com/google/uuid"
	"github.com/nxhung/proxy-ipv6-generator-cli/pkg/types"
)

const (
	// MinPrefixSize is the minimum IPv6 prefix size required for proxy generation.
	MinPrefixSize = 64
	// IPv6AddressLength is the length of an IPv6 address in bytes.
	IPv6AddressLength = 16
)

type IPv6Generator struct {
	prefix   *net.IPNet
	basePort int
	count    int
}

func New(prefix string, basePort, count int) (*IPv6Generator, error) {
	_, ipNet, err := net.ParseCIDR(prefix)
	if err != nil {
		return nil, fmt.Errorf("invalid IPv6 prefix: %w", err)
	}

	if !strings.Contains(prefix, ":") {
		return nil, fmt.Errorf("not an IPv6 prefix")
	}

	// Check that prefix is /64 or narrower (not wider)
	ones, _ := ipNet.Mask.Size()
	if ones < MinPrefixSize {
		return nil, fmt.Errorf("prefix too wide (%s); need /64 or narrower for proxy generation", prefix)
	}

	return &IPv6Generator{
		prefix:   ipNet,
		basePort: basePort,
		count:    count,
	}, nil
}

func (g *IPv6Generator) Generate(count int) ([]types.Proxy, error) {
	if count <= 0 {
		count = g.count
	}

	ones, bits := g.prefix.Mask.Size()
	hostBits := bits - ones

	if hostBits < 64 {
		return nil, fmt.Errorf("prefix too small: need at least /64 for /64 subnet, got /%d", ones)
	}

	prefixIP := g.prefix.IP.To16()
	if prefixIP == nil {
		return nil, fmt.Errorf("invalid IPv6 prefix IP")
	}

	proxies := make([]types.Proxy, 0, count)
	counter := big.NewInt(0)

	for i := 0; i < count; i++ {
		ip := make(net.IP, IPv6AddressLength)
		for j := 0; j < IPv6AddressLength; j++ {
			ip[j] = prefixIP[j]
		}

		hostPart := counter.Bytes()
		hostLen := len(hostPart)

		offset := IPv6AddressLength - hostLen
		for j := 0; j < hostLen; j++ {
			ip[offset+j] = hostPart[j]
		}

		port := g.basePort + i

		proxy := types.Proxy{
			ID:       uuid.New().String(),
			Address:  ip.String(),
			Port:     port,
			Protocol: types.ProtocolSOCKS5,
			Status:   string(types.ProxyStatusUnknown),
		}

		proxies = append(proxies, proxy)
		counter.Add(counter, big.NewInt(1))
	}

	return proxies, nil
}

func ParsePrefix(prefix string) (string, error) {
	_, ipNet, err := net.ParseCIDR(prefix)
	if err != nil {
		return "", fmt.Errorf("invalid prefix: %w", err)
	}

	return ipNet.String(), nil
}

func ValidatePrefix(prefix string) error {
	// Strip scope ID (e.g., "%eth0") if present
	if idx := strings.Index(prefix, "%"); idx != -1 {
		prefix = prefix[:idx]
	}

	// First check: is it valid CIDR notation?
	_, ipNet, err := net.ParseCIDR(prefix)
	if err != nil {
		return fmt.Errorf("invalid IPv6 prefix: %w", err)
	}

	// Second check: must be IPv6
	if !strings.Contains(prefix, ":") {
		return fmt.Errorf("not an IPv6 prefix")
	}

	// Third check: prefix must be /64 or narrower (not wider)
	ones, _ := ipNet.Mask.Size()
	if ones < MinPrefixSize {
		return fmt.Errorf("prefix too wide (%s); need /64 or narrower for proxy generation", prefix)
	}

	// Fourth check: skip IPv4-mapped (::ffff:x.x.x.x)
	ip := ipNet.IP.To4()
	if ip != nil {
		return fmt.Errorf("IPv4-mapped IPv6 addresses not supported")
	}

	return nil
}
