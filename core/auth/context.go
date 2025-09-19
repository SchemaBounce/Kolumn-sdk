package auth

import (
	"context"
)

// ContextKey is the key used to store auth info in context.
// Core (Kolumn) injects either a compatible map[string]interface{} or this AuthInfo type
// under the same key. SDK providers should use FromAuth to retrieve claims safely.
const ContextKey = "kolumn.auth"

// Claims contains minimal, stable fields for feature gating.
type Claims struct {
	Subject      string
	Issuer       string
	Scope        string
	Tier         string
	Entitlements []string
	OrgID        string
	WorkspaceID  string
}

// AuthInfo represents authentication details available to providers.
type AuthInfo struct {
	RawToken string
	Claims   Claims
}

// WithAuth stores AuthInfo in context under the well-known key.
func WithAuth(ctx context.Context, info AuthInfo) context.Context {
	return context.WithValue(ctx, ContextKey, info)
}

// FromAuth retrieves AuthInfo from context. It tolerates multiple encodings:
// 1) AuthInfo (preferred)
// 2) map[string]interface{} with optional nested "claims" map written by core
func FromAuth(ctx context.Context) (AuthInfo, bool) {
	v := ctx.Value(ContextKey)
	if v == nil {
		return AuthInfo{}, false
	}
	if info, ok := v.(AuthInfo); ok {
		return info, true
	}
	// Flexible map decoding
	if m, ok := v.(map[string]interface{}); ok {
		var ai AuthInfo
		if tok, ok := m["token"].(string); ok {
			ai.RawToken = tok
		}
		// Flatten claims from nested map if present
		var cm map[string]interface{}
		if c, ok := m["claims"]; ok {
			if mm, ok2 := c.(map[string]interface{}); ok2 {
				cm = mm
			}
		} else {
			cm = m
		}
		if cm != nil {
			if s, ok := cm["sub"].(string); ok {
				ai.Claims.Subject = s
			}
			if s, ok := cm["iss"].(string); ok {
				ai.Claims.Issuer = s
			}
			if s, ok := cm["scope"].(string); ok {
				ai.Claims.Scope = s
			}
			if s, ok := cm["tier"].(string); ok {
				ai.Claims.Tier = s
			}
			if e, ok := cm["entitlements"].([]string); ok {
				ai.Claims.Entitlements = append(ai.Claims.Entitlements, e...)
			} else if arr, ok := cm["entitlements"].([]interface{}); ok {
				for _, it := range arr {
					if s, ok := it.(string); ok {
						ai.Claims.Entitlements = append(ai.Claims.Entitlements, s)
					}
				}
			}
			if s, ok := cm["org_id"].(string); ok {
				ai.Claims.OrgID = s
			}
			if s, ok := cm["workspace_id"].(string); ok {
				ai.Claims.WorkspaceID = s
			}
		}
		return ai, true
	}
	return AuthInfo{}, false
}
