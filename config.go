package main

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Valid proxy types
var validProxyTypes = map[string]bool{
	"http":   true,
	"https":  true,
	"socks5": true,
}

// validateURL checks that a URL string is valid and has an HTTP/HTTPS scheme.
func validateURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("URL must have http or https scheme")
	}
	if u.Host == "" {
		return fmt.Errorf("URL must have a host")
	}
	return nil
}

// validateAddress checks that an address is in host:port format with a non-empty port.
func validateAddress(address string) error {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("address must be in host:port format: %w", err)
	}
	if strings.Contains(host, ":") && !strings.HasPrefix(host, "[") {
		// IPv6 address without brackets
		return fmt.Errorf("IPv6 addresses must be enclosed in brackets")
	}
	if port == "" {
		return fmt.Errorf("port is required")
	}
	return nil
}

// Config represents the full configuration
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Defaults DefaultsConfig `yaml:"defaults"`
	Proxies  []ProxyConfig  `yaml:"proxies"`
}

// ServerConfig holds server settings
type ServerConfig struct {
	Listen   string        `yaml:"listen"`
	Interval time.Duration `yaml:"interval"`
}

// DefaultsConfig holds default values
type DefaultsConfig struct {
	Target  string        `yaml:"target"`
	Timeout time.Duration `yaml:"timeout"`
}

// ProxyConfig represents a single proxy configuration
type ProxyConfig struct {
	Name        string        `yaml:"name"`
	Type        string        `yaml:"type"` // http, https, socks5
	Address     string        `yaml:"address"`
	Target      string        `yaml:"target,omitempty"`
	Timeout     time.Duration `yaml:"timeout,omitempty"`
	Username    string        `yaml:"username,omitempty"`
	Password    string        `yaml:"password,omitempty"`
	ExpectRegex string        `yaml:"expect_regex,omitempty"`

	// compiledRegex is the compiled version of ExpectRegex, set during validation
	compiledRegex *regexp.Regexp
}

// GetCompiledRegex returns the compiled regex pattern, or nil if not set.
// The regex is compiled during configuration validation for efficiency.
func (p *ProxyConfig) GetCompiledRegex() *regexp.Regexp {
	return p.compiledRegex
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate server config
	if c.Server.Listen == "" {
		return fmt.Errorf("server.listen is required")
	}
	if c.Server.Interval <= 0 {
		return fmt.Errorf("server.interval must be positive")
	}
	if c.Server.Interval < time.Second {
		return fmt.Errorf("server.interval must be at least 1 second")
	}

	// Validate defaults
	if c.Defaults.Target == "" {
		return fmt.Errorf("defaults.target is required")
	}
	if err := validateURL(c.Defaults.Target); err != nil {
		return fmt.Errorf("defaults.target is not a valid URL: %w", err)
	}
	if c.Defaults.Timeout <= 0 {
		return fmt.Errorf("defaults.timeout must be positive")
	}

	// Validate proxies
	if len(c.Proxies) == 0 {
		return fmt.Errorf("at least one proxy must be configured")
	}

	// Check for duplicate proxy names
	seen := make(map[string]int)
	for i, p := range c.Proxies {
		if existingIdx, exists := seen[p.Name]; exists {
			return fmt.Errorf("duplicate proxy name %q at indices %d and %d", p.Name, existingIdx, i)
		}
		seen[p.Name] = i

		if err := c.Proxies[i].Validate(); err != nil {
			return fmt.Errorf("proxies[%d]: %w", i, err)
		}
	}

	return nil
}

// Validate validates a proxy configuration and compiles the regex pattern if present.
// It uses a pointer receiver to store the compiled regex for later use.
func (p *ProxyConfig) Validate() error {
	if p.Name == "" {
		return fmt.Errorf("name is required")
	}
	if !validProxyTypes[p.Type] {
		return fmt.Errorf("invalid type %q, must be one of: http, https, socks5", p.Type)
	}
	if p.Address == "" {
		return fmt.Errorf("address is required")
	}
	if err := validateAddress(p.Address); err != nil {
		return err
	}

	// Validate per-proxy target URL if specified
	if p.Target != "" {
		if err := validateURL(p.Target); err != nil {
			return fmt.Errorf("target is not a valid URL: %w", err)
		}
	}

	// Compile regex pattern if specified
	if p.ExpectRegex != "" {
		regex, err := regexp.Compile(p.ExpectRegex)
		if err != nil {
			return fmt.Errorf("invalid expect_regex: %w", err)
		}
		p.compiledRegex = regex
	}

	return nil
}

// LoadConfig reads and parses the configuration file.
// Note: The returned Config is not validated. For most use cases,
// prefer LoadAndValidateConfig which ensures the configuration is valid.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// LoadAndValidateConfig reads, parses, and validates the configuration file.
// This is the recommended way to load configuration as it ensures all required
// fields are present and valid before use.
func LoadAndValidateConfig(path string) (*Config, error) {
	cfg, err := LoadConfig(path)
	if err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}
	return cfg, nil
}
