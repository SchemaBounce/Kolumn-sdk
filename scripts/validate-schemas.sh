#!/bin/bash
#
# Validate provider schemas: the struct core.Provider.Schema() returns, and
# every literal of it.
#
# - The schema struct must keep the core compatibility fields
#   (SupportedFunctions, ResourceTypes, ConfigSchema) with their JSON names.
# - A keyed literal of it that is returned directly must set Name and
#   Version (a literal stored in a variable can be filled in afterwards and
#   is not checked).
# - Every function name given to SupportedFunctions (in a literal, a field
#   assignment, an append or a variable in the same package) must be one
#   core.UnifiedDispatcher.Dispatch accepts. Values returned by other
#   function calls are not followed.
#
# The rules live in scripts/providercheck/main.go, which type-checks every
# module in the repository. Other structs whose name contains "Schema" are not
# provider schemas and are not checked.
#

set -euo pipefail

readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly NC='\033[0m' # No Color

# Always run from the repository root so relative paths resolve.
cd "$(dirname "$0")/.."

echo -e "${GREEN}Validating provider schemas...${NC}"

modules=()
while IFS= read -r mod; do
    modules+=("$(dirname "$mod")")
done < <(find . -name go.mod -not -path './.git/*' -not -path '*/vendor/*' | sort)

if ! go run scripts/providercheck/main.go schemas "${modules[@]}"; then
    echo
    echo -e "${RED}❌ SCHEMA VALIDATION VIOLATIONS FOUND!${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Provider schemas are valid${NC}"
