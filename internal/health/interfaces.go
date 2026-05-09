package health

import "github.com/nxhung/proxy-ipv6-generator-cli/pkg/types"

// ProxyProvider is the interface for accessing proxy state.
// It abstracts the proxy storage backend from the health checking logic.
type ProxyProvider interface {
	List() []types.Proxy
	UpdateStatus(id string, status string)
}

// proxyManagerAdapter wraps a types.ProxyStore to implement ProxyProvider.
type proxyManagerAdapter struct {
	store types.ProxyStore
}

func (a *proxyManagerAdapter) List() []types.Proxy {
	return a.store.List()
}

func (a *proxyManagerAdapter) UpdateStatus(id string, status string) {
	a.store.UpdateStatus(id, status)
}

// NewProxyProviderAdapter creates a ProxyProvider from a ProxyStore implementation.
// This allows existing implementations like proxy.Manager to be used with health.Checker.
func NewProxyProviderAdapter(store types.ProxyStore) ProxyProvider {
	return &proxyManagerAdapter{store: store}
}
