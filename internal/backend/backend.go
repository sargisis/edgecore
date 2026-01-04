package backend

import (
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"
)

// Backend holds the data for a backend server
type Backend struct {
	URL          *url.URL
	Alive        bool
	mux          sync.RWMutex
	ReverseProxy *httputil.ReverseProxy
	Connections  int64
}

// NewBackend creates a new Backend
func NewBackend(u *url.URL, rp *httputil.ReverseProxy) *Backend {
	return &Backend{
		URL:          u,
		Alive:        true,
		ReverseProxy: rp,
	}
}

// SetAlive for this backend
func (b *Backend) SetAlive(alive bool) {
	b.mux.Lock()
	b.Alive = alive
	b.mux.Unlock()
}

// IsAlive returns true when backend is alive
func (b *Backend) IsAlive() (alive bool) {
	b.mux.RLock()
	alive = b.Alive
	b.mux.RUnlock()
	return
}

// GetConnections returns active connections count
func (b *Backend) GetConnections() int64 {
	return atomic.LoadInt64(&b.Connections)
}

// IncConnections increases active connections count
func (b *Backend) IncConnections() {
	atomic.AddInt64(&b.Connections, 1)
}

// DecConnections decreases active connections count
func (b *Backend) DecConnections() {
	atomic.AddInt64(&b.Connections, -1)
}
