#!/bin/bash
#
# Validate that all example code compiles and follows best practices
# This ensures that examples provided in the SDK are working and up-to-date
#

set -euo pipefail

# Colors for output
readonly RED='\033[0;31m'
readonly YELLOW='\033[1;33m'
readonly GREEN='\033[0;32m'
readonly BLUE='\033[0;34m'
readonly NC='\033[0m' # No Color

echo -e "${GREEN}Validating example code...${NC}"

violations_found=0
examples_tested=0

# Find all example directories
EXAMPLE_DIRS=$(find . -type d -name "examples" -o -path "*/examples/*" -type d | sort)

if [[ -z "$EXAMPLE_DIRS" ]]; then
    echo -e "${YELLOW}⚠️  No example directories found${NC}"
    exit 0
fi

# Function to validate a single Go file
validate_go_file() {
    local file="$1"
    local dir="$2"
    
    echo -e "${BLUE}  Validating $file...${NC}"
    
    # Check if it's a proper Go file
    if ! head -1 "$file" | grep -q "^package "; then
        echo -e "${RED}    ❌ Invalid Go file - missing package declaration${NC}"
        return 1
    fi
    
    # Try to compile the file
    if ! (cd "$dir" && go build "$file" 2>/dev/null); then
        echo -e "${RED}    ❌ Compilation failed${NC}"
        # Show the actual compilation error
        (cd "$dir" && go build "$file")
        return 1
    fi
    
    echo -e "${GREEN}    ✅ Compiles successfully${NC}"
    return 0
}

# Function to validate an example directory
validate_example_dir() {
    local dir="$1"
    
    echo -e "${BLUE}Checking example directory: $dir${NC}"
    
    # Check if directory has go.mod
    if [[ ! -f "$dir/go.mod" ]]; then
        echo -e "${YELLOW}  ⚠️  No go.mod found, checking if it's a simple file example${NC}"
    fi
    
    # Find all Go files in the directory
    local go_files
    go_files=$(find "$dir" -name "*.go" -type f | head -10)  # Limit to prevent runaway
    
    if [[ -z "$go_files" ]]; then
        echo -e "${YELLOW}  ⚠️  No Go files found in $dir${NC}"
        return 0
    fi
    
    local files_in_dir=0
    local failures_in_dir=0
    
    for file in $go_files; do
        ((files_in_dir++))
        ((examples_tested++))
        
        if ! validate_go_file "$file" "$dir"; then
            ((failures_in_dir++))
            ((violations_found++))
        fi
    done
    
    if [[ $failures_in_dir -eq 0 ]]; then
        echo -e "${GREEN}  ✅ All files in $dir validate successfully ($files_in_dir files)${NC}"
    else
        echo -e "${RED}  ❌ $failures_in_dir/$files_in_dir files failed validation in $dir${NC}"
    fi
    
    # Clean up any generated binaries
    find "$dir" -type f -executable -name "*.exe" -delete 2>/dev/null || true
    find "$dir" -type f -executable ! -name "*.sh" ! -name "*.go" -delete 2>/dev/null || true
}

# Function to check example quality
check_example_quality() {
    local dir="$1"
    
    echo -e "${BLUE}  Checking example quality in $dir...${NC}"
    
    # Check for README.md
    if [[ ! -f "$dir/README.md" ]]; then
        echo -e "${YELLOW}    ⚠️  No README.md found - examples should include documentation${NC}"
    else
        echo -e "${GREEN}    ✅ README.md present${NC}"
    fi
    
    # Check for proper Provider interface implementation
    if grep -r "type.*Provider" "$dir"/*.go 2>/dev/null | grep -q "interface"; then
        echo -e "${GREEN}    ✅ Defines Provider interface${NC}"
    else
        # Check if it implements Provider from SDK
        if grep -r "Provider" "$dir"/*.go 2>/dev/null | grep -q "func.*Configure\|func.*Schema\|func.*CallFunction\|func.*Close"; then
            echo -e "${GREEN}    ✅ Implements Provider interface methods${NC}"
        else
            echo -e "${YELLOW}    ⚠️  No clear Provider interface implementation found${NC}"
        fi
    fi
    
    # Check for proper error handling
    if grep -r "if err != nil" "$dir"/*.go >/dev/null 2>&1; then
        echo -e "${GREEN}    ✅ Includes error handling${NC}"
    else
        echo -e "${YELLOW}    ⚠️  No explicit error handling found${NC}"
    fi
    
    # Check for context usage
    if grep -r "context\.Context" "$dir"/*.go >/dev/null 2>&1; then
        echo -e "${GREEN}    ✅ Uses context.Context${NC}"
    else
        echo -e "${YELLOW}    ⚠️  No context.Context usage found${NC}"
    fi
}

# Main validation loop
for dir in $EXAMPLE_DIRS; do
    if [[ ! -d "$dir" ]]; then
        continue
    fi
    
    validate_example_dir "$dir"
    check_example_quality "$dir"
    echo
done

echo -e "${GREEN}📊 Example Validation Summary:${NC}"
echo -e "   Examples tested: $examples_tested"
echo -e "   Failures: $violations_found"

if [[ $violations_found -gt 0 ]]; then
    echo
    echo -e "${RED}❌ EXAMPLE VALIDATION FAILED!${NC}"
    echo -e "${YELLOW}All example code must compile and follow best practices.${NC}"
    echo -e "${YELLOW}Please fix the compilation errors and ensure examples are current.${NC}"
    exit 1
fi

if [[ $examples_tested -eq 0 ]]; then
    echo -e "${YELLOW}⚠️  No examples found to validate${NC}"
    exit 0
fi

echo -e "${GREEN}✅ All examples validate successfully!${NC}"
exit 0