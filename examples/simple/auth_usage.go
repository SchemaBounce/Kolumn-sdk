package main

import (
	"context"
	sdkauth "github.com/schemabounce/kolumn/sdk/core/auth"
)

// This example shows how a provider can read validated auth claims from context
// to gate Pro/Enterprise features or record audit metadata.
func useAuthFromContext(ctx context.Context) {
	if info, ok := sdkauth.FromAuth(ctx); ok {
		_ = info.RawToken // do not log; use for upstream calls if absolutely necessary
		switch info.Claims.Tier {
		case "enterprise", "pro":
			// allow advanced features
		default:
			// community mode: skip Pro/Enterprise-only features
		}
		// Check entitlements, e.g., governance
		for _, e := range info.Claims.Entitlements {
			if e == "governance" {
				// unlock governance feature path
			}
		}
	}
}
