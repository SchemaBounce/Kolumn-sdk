// Package rpc provides RPC server functionality for the Kolumn Provider SDK
package rpc

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-hclog"
)

// ProviderServer implements the RPC server side for UniversalProvider
type ProviderServer struct {
	Provider UniversalProvider
	Logger   hclog.Logger
}

// Configure handles the Configure RPC call
func (s *ProviderServer) Configure(req *ConfigureRequest, resp *ConfigureResponse) error {
	s.Logger.Debug("Configure called", "config_keys", getConfigKeys(req.Config))

	ctx := context.Background()
	err := s.Provider.Configure(ctx, req.Config)
	if err != nil {
		s.Logger.Error("Configure failed", "error", err)
		resp.Error = &RPCError{
			Message: "Configuration failed",
			Details: err.Error(),
		}
		return nil
	}

	s.Logger.Debug("Configure completed successfully")
	return nil
}

// GetSchema handles the GetSchema RPC call
func (s *ProviderServer) GetSchema(req *GetSchemaRequest, resp *GetSchemaResponse) error {
	s.Logger.Debug("GetSchema called")

	schema, err := s.Provider.GetSchema()
	if err != nil {
		s.Logger.Error("GetSchema failed", "error", err)
		resp.Error = &RPCError{
			Message: "Failed to get provider schema",
			Details: err.Error(),
		}
		return nil
	}

	resp.Schema = schema
	s.Logger.Debug("GetSchema completed", "provider", schema.Provider.Name, "version", schema.Provider.Version)
	return nil
}

// CallFunction handles the CallFunction RPC call
func (s *ProviderServer) CallFunction(req *CallFunctionRequest, resp *CallFunctionResponse) error {
	s.Logger.Debug("CallFunction called", "function", req.Function)

	ctx := context.Background()
	output, err := s.Provider.CallFunction(ctx, req.Function, req.Input)
	if err != nil {
		s.Logger.Error("CallFunction failed", "function", req.Function, "error", err)
		resp.Error = &RPCError{
			Message: fmt.Sprintf("Function '%s' failed", req.Function),
			Details: err.Error(),
		}
		return nil
	}

	resp.Output = output
	s.Logger.Debug("CallFunction completed", "function", req.Function)
	return nil
}

// Close handles the Close RPC call
func (s *ProviderServer) Close(req *CloseRequest, resp *CloseResponse) error {
	s.Logger.Debug("Close called")

	err := s.Provider.Close()
	if err != nil {
		s.Logger.Error("Close failed", "error", err)
		resp.Error = &RPCError{
			Message: "Failed to close provider",
			Details: err.Error(),
		}
		return nil
	}

	s.Logger.Debug("Close completed successfully")
	return nil
}

// getConfigKeys extracts the keys from a configuration map for logging
func getConfigKeys(config map[string]interface{}) []string {
	keys := make([]string, 0, len(config))
	for k := range config {
		keys = append(keys, k)
	}
	return keys
}
