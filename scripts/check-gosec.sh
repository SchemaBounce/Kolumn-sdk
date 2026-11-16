#!/bin/bash
#
# Pre-commit hook: run the gosec security scanner over the root module.
#
# Runs `gosec -quiet ./...`, as `make security` (part of verify-all) does,
# and fails when gosec reports an issue or cannot run.
#
# When gosec is not installed the check did not run: the hook prints
# SKIPPED and exits 0 so other hooks can still be used. That is the only
# case in which it exits 0 without checking.

set -euo pipefail

if ! command -v gosec >/dev/null 2>&1; then
    echo "SKIPPED: gosec is not installed, so this check did not run."
    echo "Install it with: go install github.com/securego/gosec/v2/cmd/gosec@latest"
    exit 0
fi

exec gosec -quiet ./...
