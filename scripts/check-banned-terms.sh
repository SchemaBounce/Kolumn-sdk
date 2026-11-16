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

# Direct approach - check files one by one using find
echo -e "${GREEN}Scanning for Go and Markdown files...${NC}"

find . -name "*.go" -o -name "*.md" | while read -r file; do
    # Skip excluded files
    case "$file" in
        */.git/*|*/vendor/*|./CLAUDE.md|./scripts/check-banned-terms.sh)
            continue
            ;;
    esac
    
    if [[ -f "$file" ]]; then
        ((total_files++))
        echo -e "${GREEN}Checking file $total_files: $file${NC}"
        
        # Check for banned terms (case insensitive)
        if grep -iq "terraform" "$file" 2>/dev/null; then
            echo -e "${RED}❌ BANNED TERM FOUND in $file${NC}"
            echo -e "${YELLOW}   Matches:${NC}"
            grep -nHi "terraform" "$file" 2>/dev/null || true
            ((violations_found++))
        fi
    fi
done

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