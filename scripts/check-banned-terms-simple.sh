#!/bin/bash

set -euo pipefail

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${GREEN}Checking for banned terms (simple version)...${NC}"

violations_found=0
total_files=0

# Simple find command
files=$(find . -name "*.go" -o -name "*.md" | grep -v "./CLAUDE.md" | grep -v "./scripts/")

for file in $files; do
    if [[ -f "$file" ]]; then
        ((total_files++))
        echo "Checking: $file"
        
        # Check for terraform (case insensitive)
        if grep -iq "terraform" "$file" 2>/dev/null; then
            echo -e "${RED}❌ Found banned term in $file${NC}"
            grep -n -i "terraform" "$file" || true
            ((violations_found++))
        fi
    fi
done

echo
echo "Files scanned: $total_files"
echo "Violations found: $violations_found"

if [ $violations_found -gt 0 ]; then
    echo -e "${RED}❌ BANNED TERMS FOUND!${NC}"
    exit 1
else
    echo -e "${GREEN}✅ No banned terms found!${NC}"
    exit 0
fi