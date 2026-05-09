package types

// ProxyStore is the interface for managing a collection of proxies.
// Implemented by proxy.Manager — allows pool.Manager to be decoupled
// from the specific proxy implementation.
type ProxyStore interface {
	List() []Proxy
	Add(proxy Proxy)
	Remove(id string) bool
	Get(id string) (Proxy, bool)
	Count() int
	UpdateStatus(id string, status string)
}
