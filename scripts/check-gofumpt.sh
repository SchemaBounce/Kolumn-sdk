#!/bin/bash
#
# Pre-commit hook: report Go files that gofumpt would reformat.
#
# Runs `gofumpt -l .` over the repository, as `make fmt` formats it, and
# fails when gofumpt lists any file. This hook only reports; run
# `gofumpt -w .` (or `make fmt`) to apply the changes.
#
# When gofumpt is not installed the check did not run: the hook prints
# SKIPPED and exits 0 so other hooks can still be used. That is the only
# case in which it exits 0 without checking.

set -euo pipefail

if ! command -v gofumpt >/dev/null 2>&1; then
    echo "SKIPPED: gofumpt is not installed, so this check did not run."
    echo "Install it with: go install mvdan.cc/gofumpt@latest"
    exit 0
fi

# gofumpt exits non-zero on parse errors; set -e propagates that.
files=$(gofumpt -l .)

if [[ -n "$files" ]]; then
    echo "gofumpt would reformat these files (run: gofumpt -w .):"
    echo "$files"
    exit 1
fi

echo "gofumpt: all Go files are formatted."
