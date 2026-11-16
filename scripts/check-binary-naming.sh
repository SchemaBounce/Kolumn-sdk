#!/bin/bash
#
# Validate the provider binary naming convention: kolumn-provider-{name}.
#
# Kolumn discovers provider binaries by that prefix (cmd/kolumn-docs-gen strips
# it the same way), and README.md tells provider authors to name the project
# directory, the module and the binary kolumn-provider-{name}. This SDK repo
# ships no provider of its own, so the check guards what it does contain:
#
#   1. Provider modules: a go.mod outside examples/ and templates/ whose
#      module name contains "provider" must be named kolumn-provider-{name}.
#   2. Provider binaries: a `go build -o <out>` in a build file or doc whose
#      output name contains "provider" must be kolumn-provider-{name}.
#      Rules 1 and 2 only see names that say "provider": a provider binary
#      named plainly `mydb` is indistinguishable from any other tool here.
#   3. Provider names in docs and build files: every concrete
#      kolumn-provider-<x> must be a valid name. The placeholder
#      kolumn-provider-{name}, shell-templated names (kolumn-provider-$NAME,
#      kolumn-provider-${NAME}) and sentence-ending dots are not violations.
#   4. README.md must still state the rule, and its `go mod init` example
#      must use a valid kolumn-provider-{name} module.
#
# Example directories (examples/*) are demos built with `go run`, not
# provider binaries, so their directory names are not checked.
#

set -euo pipefail

readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly BLUE='\033[0;34m'
readonly NC='\033[0m' # No Color

# Always run from the repository root so relative paths resolve.
cd "$(dirname "$0")/.."

readonly REQUIRED_PATTERN='^kolumn-provider-[a-z0-9]([a-z0-9-]*[a-z0-9])?$'
readonly PLACEHOLDER='kolumn-provider-{name}'

violations_found=0
names_checked=0

fail() {
    echo -e "${RED}  ❌ $*${NC}"
    violations_found=$((violations_found + 1))
}

check_name() {
    local where="$1" name="$2"
    names_checked=$((names_checked + 1))
    if [[ $name =~ $REQUIRED_PATTERN ]]; then
        echo -e "${GREEN}  ✅ $where: $name${NC}"
    else
        fail "$where: '$name' must match kolumn-provider-{name} (lowercase letters, digits, hyphens)"
    fi
}

echo -e "${GREEN}Checking provider binary naming convention...${NC}"

# Docs and build files that name binaries. This script is excluded: it quotes
# bad names on purpose.
doc_and_build_files=()
while IFS= read -r f; do
    doc_and_build_files+=("$f")
done < <(find . \( -path ./.git -o -path '*/vendor' \) -prune -o -type f \
    \( -name '*.md' -o -name Makefile -o -name '*.mk' -o -name '*.sh' \
       -o -name '*.yml' -o -name '*.yaml' \) \
    -not -path ./scripts/check-binary-naming.sh -print | sort)

# 1. Provider modules.
echo -e "${BLUE}Provider modules (go.mod outside examples/ and templates/)...${NC}"
while IFS= read -r mod; do
    module=$(awk '$1 == "module" { print $2; exit }' "$mod")
    last="${module##*/}"
    if [[ $last == *provider* ]]; then
        check_name "$mod" "$last"
    fi
done < <(find . \( -path ./.git -o -path ./examples -o -path ./templates -o -path '*/vendor' \) -prune \
    -o -name go.mod -print | sort)

# 2. Provider binaries built with go build -o.
echo -e "${BLUE}Provider binaries (go build -o)...${NC}"
for f in "${doc_and_build_files[@]}"; do
    while IFS=: read -r line_num out; do
        out="${out//[\`\"\']/}"           # drop quotes and backticks around the path
        [[ $out == *'$'* ]] && continue   # shell-templated output name
        name=$(basename "$out" .exe)
        while [[ $name == *. ]]; do name="${name%.}"; done
        if [[ $name == *provider* ]]; then
            check_name "$f:$line_num" "$name"
        fi
    done < <(grep -nE 'go build[^#]*[[:space:]]-o[[:space:]=]+[^[:space:]]+' "$f" 2>/dev/null |
        sed -E 's/^([0-9]+):.*go build[^#]*[[:space:]]-o[[:space:]=]+([^[:space:]]+).*/\1:\2/' || true)
done

# 3. Provider names mentioned in docs and build files.
echo -e "${BLUE}Provider names in docs and build files...${NC}"
for f in "${doc_and_build_files[@]}"; do
    while IFS=: read -r line_num token; do
        [[ $token == "$PLACEHOLDER" ]] && continue
        [[ $token == *'$'* ]] && continue
        while [[ $token == *. ]]; do token="${token%.}"; done
        check_name "$f:$line_num" "$token"
    done < <(grep -noE 'kolumn-provider-[^][:space:]`"'"'"'(),;:<>|*/\\$]*\$?' "$f" 2>/dev/null || true)
done

# 4. What README.md requires of provider authors.
echo -e "${BLUE}README.md naming rule...${NC}"
if ! grep -qF "$PLACEHOLDER" README.md; then
    fail "README.md no longer states the $PLACEHOLDER naming rule"
fi
init_lines=$(grep -nE 'go mod init[[:space:]]+[^[:space:]]+' README.md || true)
if [[ -z $init_lines ]]; then
    fail "README.md has no 'go mod init' example for a provider module"
else
    while IFS=: read -r line_num rest; do
        module=$(sed -E 's/.*go mod init[[:space:]]+([^[:space:]]+).*/\1/' <<< "$rest")
        check_name "README.md:$line_num (go mod init)" "${module##*/}"
    done <<< "$init_lines"
fi

echo
echo -e "${GREEN}📊 Binary Naming Summary:${NC}"
echo -e "   Names checked: $names_checked"
echo -e "   Naming violations: $violations_found"

if [ $violations_found -gt 0 ]; then
    echo
    echo -e "${RED}❌ BINARY NAMING VIOLATIONS FOUND!${NC}"
    echo "Provider modules and binaries must be named kolumn-provider-{name},"
    echo "e.g. kolumn-provider-postgres. Kolumn discovers providers by this prefix."
    exit 1
fi

echo -e "${GREEN}✅ Provider naming follows kolumn-provider-{name}${NC}"
exit 0
