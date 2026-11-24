// Package config handles loading and parsing of the application configuration
// from a YAML file.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"

	"gopkg.in/yaml.v3"
)

// Config holds all application configuration sections.
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Seed     SeedConfig     `yaml:"seed"`
	Log      LogConfig      `yaml:"log"`
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
	// SecureCookies marks the session cookie Secure. Turn it on when the
	// server is reached over HTTPS, for example behind a TLS proxy.
	SecureCookies bool `yaml:"secure_cookies"`
	// PublicURL is the address phones use to reach the server, for example
	// the https address the server is published at, such as https://household.example.com. It is
	// only shown by "household-app device add"; the server does not use it.
	PublicURL string `yaml:"public_url"`
}

// DatabaseConfig holds SQLite database settings.
type DatabaseConfig struct {
	Path string `yaml:"path"`
}

// SeedConfig holds the location of the CSV files that are imported on
// startup and exported back after sync.
type SeedConfig struct {
	Dir string `yaml:"dir"`
}

// LogConfig holds logging settings.
type LogConfig struct {
	// Level is one of debug, info, warn or error.
	Level string `yaml:"level"`
}

// Load reads the YAML configuration file at the given path and returns
// a parsed Config. Missing optional values are filled with sensible defaults,
// and an unknown key is an error, so a typo cannot silently do nothing.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file %s: %w", path, err)
	}

	cfg := &Config{}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	// An empty file is a valid configuration: everything takes its default.
	if err := decoder.Decode(cfg); err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("parse config file %s: %w", path, err)
	}

	setDefaults(cfg)
	if err := validate(cfg); err != nil {
		return nil, fmt.Errorf("check config file %s: %w", path, err)
	}

	return cfg, nil
}

func validate(cfg *Config) error {
	if raw := cfg.Server.PublicURL; raw != "" {
		u, err := url.Parse(raw)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return fmt.Errorf("server.public_url %q must be an http or https address", raw)
		}
	}
	return nil
}

func setDefaults(cfg *Config) {
	if cfg.Server.Host == "" {
		cfg.Server.Host = "127.0.0.1"
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if cfg.Database.Path == "" {
		cfg.Database.Path = "household.db"
	}
	if cfg.Seed.Dir == "" {
		cfg.Seed.Dir = "seed"
	}
	if cfg.Log.Level == "" {
		cfg.Log.Level = "info"
	}
}
