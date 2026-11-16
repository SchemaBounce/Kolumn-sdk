#!/bin/bash
#
# Check that every example under examples/ builds, vets and passes its tests.
# This is the single implementation behind `make test-examples` (which
# `make test` runs), `make validate-examples` and the validate-examples
# pre-commit hook.
#
# Two kinds of example exist:
#   - packages inside the root SDK module (no go.mod of their own): vetted
#     from the repository root, which also type-checks them;
#   - nested modules with their own go.mod (which replace the SDK with the
#     local checkout): vetted and tested with -race from their own directory.
#

set -euo pipefail

# Colors for output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly BLUE='\033[0;34m'
readonly NC='\033[0m' # No Color

# Always run from the repository root so relative paths resolve.
cd "$(dirname "$0")/.."

if [[ ! -d examples ]]; then
    echo -e "${RED}❌ examples/ directory not found${NC}"
    exit 1
fi

echo -e "${GREEN}Checking examples in the root module...${NC}"
echo -e "${BLUE}  go vet ./examples/...${NC}"
go vet ./examples/...

echo -e "${GREEN}Checking example modules...${NC}"
modules=0
while IFS= read -r mod; do
    dir=$(dirname "$mod")
    echo -e "${BLUE}  $dir${NC}"
    (cd "$dir" && go vet ./... && go test -race ./...)
    modules=$((modules + 1))
done < <(find examples -name go.mod -not -path '*/vendor/*' | sort)

echo -e "${GREEN}✅ Examples passed (root-module examples + $modules example module(s))${NC}"
