package config

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
)

type Config struct {
	Backends  []string `json:"backends"`
	Port      int      `json:"port"`
	RateLimit float64  `json:"rate_limit"`
	Burst     float64  `json:"burst"`
}

func LoadConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var cfg Config
	err = json.NewDecoder(file).Decode(&cfg)
	return &cfg, err
}

// Validate performs basic sanity checks on the configuration.
func (c *Config) Validate() error {
	if len(c.Backends) == 0 {
		return fmt.Errorf("no backends configured")
	}

	for _, rawURL := range c.Backends {
		u, err := url.Parse(rawURL)
		if err != nil || u.Scheme == "" || u.Host == "" {
			return fmt.Errorf("invalid backend URL %q", rawURL)
		}
	}

	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("invalid port %d: must be between 1 and 65535", c.Port)
	}

	if c.RateLimit < 0 {
		return fmt.Errorf("rate_limit must be >= 0")
	}
	if c.Burst < 0 {
		return fmt.Errorf("burst must be >= 0")
	}

	return nil
}
