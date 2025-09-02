// Package rpc provides plugin serving functionality for the Kolumn Provider SDK
package rpc

import (
	"log"
	"net/rpc"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/schemabounce/kolumn/sdk"
)

// ServeProvider serves a provider as an RPC plugin
// This is the main entry point for provider plugins
func ServeProvider(config *ServeConfig) {
	if config == nil {
		log.Fatal("ServeConfig cannot be nil")
	}

	if config.Provider == nil {
		log.Fatal("Provider cannot be nil in ServeConfig")
	}

	// Setup logger
	logger := getLogger(config.Logger, config.Debug)

	// Create provider info
	schema, err := config.Provider.GetSchema()
	if err != nil {
		logger.Error("failed to get provider schema", "error", err)
		log.Fatalf("Failed to get provider schema: %v", err)
	}

	providerInfo := &ProviderInfo{
		Name:       schema.Provider.Name,
		Version:    schema.Provider.Version,
		SDKVersion: sdk.Version,
		APIVersion: sdk.APIVersion,
	}

	logger.Info("starting provider plugin",
		"name", providerInfo.Name,
		"version", providerInfo.Version,
		"sdk_version", providerInfo.SDKVersion,
	)

	// Plugin handshake configuration
	handshakeConfig := plugin.HandshakeConfig{
		ProtocolVersion:  uint(sdk.ProtocolVersion),
		MagicCookieKey:   "KOLUMN_PLUGIN",
		MagicCookieValue: "kolumn-provider-plugin",
	}

	// Plugin map
	pluginMap := map[string]plugin.Plugin{
		"provider": &KolumnProviderPlugin{
			Provider: config.Provider,
			Logger:   logger,
		},
	}

	// Serve the plugin
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: handshakeConfig,
		Plugins:         pluginMap,
		GRPCServer:      plugin.DefaultGRPCServer,
		Logger:          logger,
	})
}

// getLogger creates or converts a logger for plugin use
func getLogger(logger interface{}, debug bool) hclog.Logger {
	if logger == nil {
		level := hclog.Info
		if debug {
			level = hclog.Debug
		}

		return hclog.New(&hclog.LoggerOptions{
			Name:  "kolumn-provider",
			Level: level,
		})
	}

	// If it's already an hclog.Logger, return it
	if hclogger, ok := logger.(hclog.Logger); ok {
		return hclogger
	}

	// Convert other logger types if needed
	// For now, create a new one if it's not hclog
	level := hclog.Info
	if debug {
		level = hclog.Debug
	}

	return hclog.New(&hclog.LoggerOptions{
		Name:  "kolumn-provider",
		Level: level,
	})
}

// KolumnProviderPlugin implements the plugin.Plugin interface
type KolumnProviderPlugin struct {
	Provider UniversalProvider
	Logger   hclog.Logger
}

// Server returns the RPC server for this plugin
func (p *KolumnProviderPlugin) Server(*plugin.MuxBroker) (interface{}, error) {
	return &ProviderServer{
		Provider: p.Provider,
		Logger:   p.Logger,
	}, nil
}

// Client returns the RPC client for this plugin
func (p *KolumnProviderPlugin) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &ProviderClient{
		Client: c,
		Logger: p.Logger,
	}, nil
}
