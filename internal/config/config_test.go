package config

import "testing"

func TestConfigValidateSuccess(t *testing.T) {
	cfg := &Config{
		Backends:  []string{"http://localhost:8081"},
		Port:      8080,
		RateLimit: 100,
		Burst:     10,
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected config to be valid, got error: %v", err)
	}
}

func TestConfigValidateNoBackends(t *testing.T) {
	cfg := &Config{
		Backends:  []string{},
		Port:      8080,
		RateLimit: 100,
		Burst:     10,
	}

	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected error when no backends configured")
	}
}

func TestConfigValidateInvalidPort(t *testing.T) {
	cfg := &Config{
		Backends:  []string{"http://localhost:8081"},
		Port:      70000,
		RateLimit: 100,
		Burst:     10,
	}

	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected error for invalid port")
	}
}

func TestConfigValidateInvalidBackendURL(t *testing.T) {
	cfg := &Config{
		Backends:  []string{":://bad-url"},
		Port:      8080,
		RateLimit: 100,
		Burst:     10,
	}

	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected error for invalid backend URL")
	}
}
