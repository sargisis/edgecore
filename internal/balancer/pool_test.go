package balancer

import (
	"net/http/httputil"
	"net/url"
	"testing"

	"github.com/sargisis/edgecore/internal/backend"
)

func newTestBackend(t *testing.T, rawURL string) *backend.Backend {
	t.Helper()

	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("failed to parse URL %q: %v", rawURL, err)
	}
	return backend.NewBackend(u, &httputil.ReverseProxy{})
}

func TestServerPoolGetLeastConnections(t *testing.T) {
	var pool ServerPool

	b1 := newTestBackend(t, "http://backend1")
	b2 := newTestBackend(t, "http://backend2")
	b3 := newTestBackend(t, "http://backend3")

	pool.AddBackend(b1)
	pool.AddBackend(b2)
	pool.AddBackend(b3)

	// Simulate different number of active connections.
	b1.IncConnections() // 1
	b2.IncConnections() // 1
	b2.IncConnections() // 2
	// b3 remains with 0 connections

	least := pool.GetLeastConnections()
	if least != b3 {
		t.Fatalf("expected backend3 to have least connections, got %v", least.URL)
	}
}

func TestServerPoolGetLeastConnectionsSkipsDead(t *testing.T) {
	var pool ServerPool

	b1 := newTestBackend(t, "http://backend1")
	b2 := newTestBackend(t, "http://backend2")

	pool.AddBackend(b1)
	pool.AddBackend(b2)

	b1.SetAlive(false) // first backend is "dead"

	least := pool.GetLeastConnections()
	if least != b2 {
		t.Fatalf("expected alive backend2 to be chosen, got %v", least.URL)
	}
}
