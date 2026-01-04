package config

import (
	"encoding/json"
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
