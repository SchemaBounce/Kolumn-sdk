#!/bin/bash
#
# Check Provider interface compliance against the real core.Provider.
#
# - core.Provider must still be the 4-method contract (Configure, Schema,
#   CallFunction, Close with their current signatures).
# - Every type that claims to be a provider must implement core.Provider. A
#   type claims it when it has a CallFunction method (any signature), or at
#   least two of the four method names with Configure or Schema matching
#   core.Provider exactly. Types explicitly declared with
#   `var _ sdktesting.SchemaProvider = (*T)(nil)` are skipped (listed in the
#   output): that interface cannot be satisfied together with core.Provider.
#
# The rules live in scripts/providercheck/main.go, which type-checks every
# module in the repository (the root SDK module and each nested example
# module). Interfaces that merely have "Provider" in their name, and structs
# that merely have a Schema method, are not providers and are not checked.
#

set -euo pipefail

readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly NC='\033[0m' # No Color

# Always run from the repository root so relative paths resolve.
cd "$(dirname "$0")/.."

echo -e "${GREEN}Checking Provider interface compliance...${NC}"

modules=()
while IFS= read -r mod; do
    modules+=("$(dirname "$mod")")
done < <(find . -name go.mod -not -path './.git/*' -not -path '*/vendor/*' | sort)

if ! go run scripts/providercheck/main.go interfaces "${modules[@]}"; then
    echo
    echo -e "${RED}❌ INTERFACE COMPLIANCE VIOLATIONS FOUND!${NC}"
    echo "Providers must implement core.Provider exactly:"
    echo "   - Configure(ctx context.Context, config map[string]interface{}) error"
    echo "   - Schema() (*core.Schema, error)"
    echo "   - CallFunction(ctx context.Context, function string, input []byte) ([]byte, error)"
    echo "   - Close() error"
    exit 1
fi

echo -e "${GREEN}✅ Provider interface compliance passed${NC}"
