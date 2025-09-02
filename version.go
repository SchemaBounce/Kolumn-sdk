// Package sdk provides the Kolumn Provider SDK
package sdk

const (
	// Version represents the current SDK version
	Version = "v0.1.0"

	// APIVersion represents the API compatibility version
	APIVersion = "v1"

	// ProtocolVersion represents the RPC protocol version
	ProtocolVersion = 1
)

// SDKInfo provides information about the SDK
type SDKInfo struct {
	Version         string `json:"version"`
	APIVersion      string `json:"api_version"`
	ProtocolVersion int    `json:"protocol_version"`
	GoVersion       string `json:"go_version"`
}

// GetSDKInfo returns information about the current SDK
func GetSDKInfo() *SDKInfo {
	return &SDKInfo{
		Version:         Version,
		APIVersion:      APIVersion,
		ProtocolVersion: ProtocolVersion,
		GoVersion:       "1.24",
	}
}
