// Package rpc - discovery provides high-level client functionality for the Kolumn Provider SDK
package rpc

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/schemabounce/kolumn/sdk"
)

// KolumnProviderHandshake is the handshake configuration for provider plugins
var KolumnProviderHandshake = plugin.HandshakeConfig{
	ProtocolVersion:  uint(sdk.ProtocolVersion),
	MagicCookieKey:   "KOLUMN_PLUGIN",
	MagicCookieValue: "kolumn-provider-plugin",
}

// Client provides high-level RPC provider discovery and management
type Client struct {
	logger hclog.Logger
	// plugins maps provider names to their plugin clients
	plugins map[string]*plugin.Client
	// providers maps provider names to their RPC clients
	providers map[string]UniversalProvider
	// mutex protects concurrent access
	mutex sync.RWMutex
}

// NewClient creates a new RPC client for provider discovery and management
func NewClient() *Client {
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "kolumn-rpc-client",
		Level:  hclog.Info,
		Output: os.Stderr,
	})

	return &Client{
		logger:    logger,
		plugins:   make(map[string]*plugin.Client),
		providers: make(map[string]UniversalProvider),
	}
}

// DiscoverProviders discovers available provider plugins in standard paths
func (c *Client) DiscoverProviders() ([]string, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	var providers []string
	searchPaths := []string{
		"./bin",
		".",
		"/usr/local/bin",
	}

	// Add PATH directories
	if path := os.Getenv("PATH"); path != "" {
		searchPaths = append(searchPaths, strings.Split(path, string(os.PathListSeparator))...)
	}

	// Look for kolumn-provider-* binaries
	for _, searchPath := range searchPaths {
		if _, err := os.Stat(searchPath); os.IsNotExist(err) {
			continue
		}

		entries, err := os.ReadDir(searchPath)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			name := entry.Name()
			if strings.HasPrefix(name, "kolumn-provider-") {
				// Extract provider name
				providerName := strings.TrimPrefix(name, "kolumn-provider-")

				// Validate binary is executable
				binaryPath := filepath.Join(searchPath, name)
				if isExecutable(binaryPath) {
					providers = append(providers, providerName)
				}
			}
		}
	}

	// Remove duplicates
	seen := make(map[string]bool)
	var unique []string
	for _, provider := range providers {
		if !seen[provider] {
			seen[provider] = true
			unique = append(unique, provider)
		}
	}

	return unique, nil
}

// Connect connects to a specific provider plugin
func (c *Client) Connect(providerName string) (UniversalProvider, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Check if already connected
	if provider, exists := c.providers[providerName]; exists {
		return provider, nil
	}

	// Find the binary
	binaryName := fmt.Sprintf("kolumn-provider-%s", providerName)
	binaryPath, err := c.findProviderBinary(binaryName)
	if err != nil {
		return nil, fmt.Errorf("failed to find provider binary '%s': %w", binaryName, err)
	}

	// Create plugin client
	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: KolumnProviderHandshake,
		Plugins: map[string]plugin.Plugin{
			"provider": &KolumnProviderPlugin{},
		},
		Cmd:    exec.Command(binaryPath),
		Logger: c.logger.Named(providerName),
	})

	// Connect to the plugin
	rpcClient, err := client.Client()
	if err != nil {
		client.Kill()
		return nil, fmt.Errorf("failed to connect to provider plugin: %w", err)
	}

	// Get the provider
	raw, err := rpcClient.Dispense("provider")
	if err != nil {
		client.Kill()
		return nil, fmt.Errorf("failed to dispense provider: %w", err)
	}

	provider := raw.(UniversalProvider)

	// Store references
	c.plugins[providerName] = client
	c.providers[providerName] = provider

	return provider, nil
}

// Close closes all provider connections
func (c *Client) Close() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	for name, client := range c.plugins {
		c.logger.Debug("closing provider connection", "provider", name)
		client.Kill()
	}

	c.plugins = make(map[string]*plugin.Client)
	c.providers = make(map[string]UniversalProvider)
}

// findProviderBinary finds a provider binary in standard search paths
func (c *Client) findProviderBinary(binaryName string) (string, error) {
	searchPaths := []string{
		"./bin",
		".",
		"/usr/local/bin",
	}

	// Add PATH directories
	if path := os.Getenv("PATH"); path != "" {
		searchPaths = append(searchPaths, strings.Split(path, string(os.PathListSeparator))...)
	}

	for _, searchPath := range searchPaths {
		binaryPath := filepath.Join(searchPath, binaryName)
		if isExecutable(binaryPath) {
			return binaryPath, nil
		}
	}

	return "", fmt.Errorf("binary '%s' not found in search paths", binaryName)
}

// isExecutable checks if a file exists and is executable
func isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	// Check if it's a regular file and executable
	return info.Mode().IsRegular() && (info.Mode().Perm()&0111) != 0
}
