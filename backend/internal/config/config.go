// Package config loads and validates application configuration from
// environment variables.
package config

import "os"

// Config holds all runtime configuration for the server.
type Config struct {
	Port string
}

// Load reads configuration from environment variables, applying defaults
// where appropriate.
func Load() (Config, error) {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	return Config{Port: port}, nil
}
