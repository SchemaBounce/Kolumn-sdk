#!/bin/bash
#
# Check for banned terms in the codebase
# This script enforces the "terraform" reference prohibition per CLAUDE.md
#

set -euo pipefail

# Colors for output
readonly RED='\033[0;31m'
readonly YELLOW='\033[1;33m'
readonly GREEN='\033[0;32m'
readonly NC='\033[0m' # No Color

echo -e "${GREEN}Checking for banned terms...${NC}"

violations_found=0
total_files=0

# Narrow exceptions: identifiers that are allowed to contain the banned term,
# and only in the listed files. Each entry is "file|identifier".
#
# helpers/ui keeps FormatTerraformStyle and FormatTerraformStyleWithDuration
# as deprecated aliases of FormatResourceAction and
# FormatResourceActionWithDuration so existing callers (the Kolumn CLI) keep
# compiling. Only those exact identifiers are allowed, only in those files;
# any other mention of the term there still fails. Remove these entries when
# the aliases are deleted.
readonly ALLOWED_IDENTIFIERS=(
    "./helpers/ui/human_output.go|FormatTerraformStyleWithDuration"
    "./helpers/ui/human_output.go|FormatTerraformStyle"
    "./helpers/ui/human_output_test.go|FormatTerraformStyleWithDuration"
    "./helpers/ui/human_output_test.go|FormatTerraformStyle"
)

# banned_lines FILE prints, as file:line:text, the original lines that still
# contain the banned term after removing the identifiers allowed for that
# file.
banned_lines() {
    local file="$1"
    local sed_args=(-e ':a')
    local entry
    for entry in "${ALLOWED_IDENTIFIERS[@]}"; do
        if [[ "${entry%%|*}" == "$file" ]]; then
            # Whole-identifier match only (FormatTerraformStyleX is not
            # allowed), in portable ERE rather than GNU-only \b. One removal
            # per pass; "ta" loops until none are left on the line.
            sed_args+=(-e "s/(^|[^A-Za-z0-9_])${entry#*|}([^A-Za-z0-9_]|\$)/\1\2/" -e 'ta')
        fi
    done
    if [[ ${#sed_args[@]} -eq 1 ]]; then
        grep -nHi "terraform" "$file" 2>/dev/null || true
        return
    fi
    local line_num
    sed -E "${sed_args[@]}" "$file" | grep -ni "terraform" 2>/dev/null | cut -d: -f1 |
        while IFS= read -r line_num; do
            printf '%s:%s:%s\n' "$file" "$line_num" "$(sed -n "${line_num}p" "$file")"
        done || true
}

# Direct approach - check files one by one using find
echo -e "${GREEN}Scanning for Go and Markdown files...${NC}"

while IFS= read -r file; do
    # Skip excluded files
    case "$file" in
        */.git/*|*/vendor/*|./CLAUDE.md|./scripts/check-banned-terms.sh)
            continue
            ;;
    esac

    if [[ -f "$file" ]]; then
        total_files=$((total_files + 1))
        echo -e "${GREEN}Checking file $total_files: $file${NC}"

        # Check for banned terms (case insensitive)
        matches=$(banned_lines "$file")
        if [[ -n "$matches" ]]; then
            echo -e "${RED}❌ BANNED TERM FOUND in $file${NC}"
            echo -e "${YELLOW}   Matches:${NC}"
            echo "$matches"
            violations_found=$((violations_found + 1))
        fi
    fi
done < <(find . \( -name "*.go" -o -name "*.md" \) -type f)

echo
echo -e "${GREEN}📊 Scan Summary:${NC}"
echo -e "   Files scanned: $total_files"
echo -e "   Violations found: $violations_found"

if [ $violations_found -gt 0 ]; then
    echo
    echo -e "${RED}❌ BANNED TERMS DETECTED!${NC}"
    echo -e "${YELLOW}The word 'terraform' or 'Terraform' is prohibited in the Kolumn SDK codebase${NC}"
    echo
    echo -e "${YELLOW}Please use alternative terms:${NC}"
    echo -e "   - infrastructure-as-code"
    echo -e "   - IaC"
    echo -e "   - configuration management"
    echo -e "   - provider SDK"
    echo -e "   - HashiCorp Provider SDK (when referencing similar tools)"
    echo
    echo -e "${YELLOW}This restriction maintains product independence per CLAUDE.md${NC}"
    exit 1
fi

echo -e "${GREEN}✅ No banned terms found!${NC}"
exit 0
