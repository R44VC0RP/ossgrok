package registry

import (
	"fmt"
	"sync"

	"github.com/R44VC0RP/ossgrok/pkg/logger"
)

// TunnelConnection represents a tunnel connection interface
type TunnelConnection interface {
	Domain() string
	TunnelID() string
	Close() error
}

// Registry manages the mapping of domains to tunnel connections
type Registry struct {
	mu      sync.RWMutex
	tunnels map[string]TunnelConnection
}

// New creates a new tunnel registry
func New() *Registry {
	return &Registry{
		tunnels: make(map[string]TunnelConnection),
	}
}

// Register registers a new tunnel for a domain
func (r *Registry) Register(domain string, conn TunnelConnection) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tunnels[domain]; exists {
		return fmt.Errorf("domain %s is already registered", domain)
	}

	r.tunnels[domain] = conn
	logger.Info("Registered tunnel for domain: %s (tunnel_id: %s)", domain, conn.TunnelID())
	return nil
}

// Unregister removes a tunnel registration
func (r *Registry) Unregister(domain string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if conn, exists := r.tunnels[domain]; exists {
		delete(r.tunnels, domain)
		logger.Info("Unregistered tunnel for domain: %s (tunnel_id: %s)", domain, conn.TunnelID())
	}
}

// GetTunnel retrieves a tunnel connection for a domain
func (r *Registry) GetTunnel(domain string) (TunnelConnection, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	conn, exists := r.tunnels[domain]
	return conn, exists
}

// Count returns the number of registered tunnels
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.tunnels)
}

// List returns all registered domains
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	domains := make([]string, 0, len(r.tunnels))
	for domain := range r.tunnels {
		domains = append(domains, domain)
	}
	return domains
}
