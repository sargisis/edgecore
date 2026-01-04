package balancer

import (
	"edgecore/internal/backend"
	"log"
	"net"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
)

// ServerPool holds information about reachable backends
type ServerPool struct {
	backends []*backend.Backend
	current  uint64
	mux      sync.RWMutex
}

// AddBackend adds a new backend to the pool
func (s *ServerPool) AddBackend(b *backend.Backend) {
	s.mux.Lock()
	defer s.mux.Unlock()
	s.backends = append(s.backends, b)
}

// Clear removes all backends from the pool
func (s *ServerPool) Clear() {
	s.mux.Lock()
	defer s.mux.Unlock()
	s.backends = []*backend.Backend{}
}

// GetNextPeer returns the next active peer to take a connection (Round Robin)
func (s *ServerPool) GetNextPeer() *backend.Backend {
	s.mux.RLock()
	defer s.mux.RUnlock()

	if len(s.backends) == 0 {
		return nil
	}

	// Loop over the list to find an alive backend
	next := s.nextIndex()
	l := len(s.backends) + int(next)
	for i := next; i < uint64(l); i++ {
		idx := int(i % uint64(len(s.backends)))

		// Check if the backend is alive (skipping dead ones)
		if s.backends[idx].IsAlive() {
			if i != next {
				// We had to skip some, meaning we should update 'current'
				// to point to this one to start from here next time
				atomic.StoreUint64(&s.current, uint64(idx))
			}
			return s.backends[idx]
		}
	}
	return nil
}

// GetLeastConnections returns the backend with the least number of active connections
func (s *ServerPool) GetLeastConnections() *backend.Backend {
	s.mux.RLock()
	defer s.mux.RUnlock()

	var leastConnPeer *backend.Backend
	for _, b := range s.backends {
		if b.IsAlive() {
			if leastConnPeer == nil || b.GetConnections() < leastConnPeer.GetConnections() {
				leastConnPeer = b
			}
		}
	}
	return leastConnPeer
}

// nextIndex atomically increases the counter and returns the index
func (s *ServerPool) nextIndex() uint64 {
	return atomic.AddUint64(&s.current, 1) % uint64(len(s.backends))
}

// HealthCheck pings the backends and updates their status
func (s *ServerPool) HealthCheck() {
	s.mux.RLock()
	defer s.mux.RUnlock()

	for _, b := range s.backends {
		status := "up"
		alive := isBackendAlive(b.URL)
		b.SetAlive(alive)
		if !alive {
			status = "down"
		}
		log.Printf("%s [%s]\n", b.URL, status)
	}
}

// isBackendAlive checks whether a backend is responsive by attempting a TCP connection
func isBackendAlive(u *url.URL) bool {
	conn, err := net.DialTimeout("tcp", u.Host, 2*time.Second)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}
