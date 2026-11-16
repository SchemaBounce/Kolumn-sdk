#!/bin/bash
#
# Pre-commit hook: run the gosec security scanner over the root module.
#
# Runs `gosec -quiet ./...`, the command `make security` (part of
# verify-all) runs, and fails when gosec reports an issue or cannot run.
# Unlike `make security`, it fails on findings.
#
# gosec skips a file it cannot parse and still exits 0, so the hook first
# checks that every Go file parses (`gofmt -e -l .` exits non-zero on a
# syntax error) and fails if one does not.
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

if ! gofmt -e -l . >/dev/null; then
    echo "gosec: a Go file failed to parse (errors above); gosec would skip it, so failing."
    exit 1
fi

exec gosec -quiet ./...
