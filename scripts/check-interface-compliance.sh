#!/bin/bash
#
# Check Provider interface compliance
# Ensures that Provider implementations follow the correct 4-method interface
#

set -euo pipefail

# Colors for output
readonly RED='\033[0;31m'
readonly YELLOW='\033[1;33m'
readonly GREEN='\033[0;32m'
readonly BLUE='\033[0;34m'
readonly NC='\033[0m' # No Color

echo -e "${GREEN}Checking Provider interface compliance...${NC}"

violations_found=0
providers_found=0

# Required Provider interface methods with correct signatures
readonly REQUIRED_METHODS=(
    "Configure.*context\.Context.*map\[string\]interface\{\}.*error"
    "Schema.*\*.*Schema.*error"
    "CallFunction.*context\.Context.*string.*\[\]byte.*\[\]byte.*error"
    "Close.*error"
)

# Method descriptions for error reporting
readonly METHOD_DESCRIPTIONS=(
    "Configure(ctx context.Context, config map[string]interface{}) error"
    "Schema() (*Schema, error)"
    "CallFunction(ctx context.Context, function string, input []byte) ([]byte, error)"
    "Close() error"
)

# Find all Go files
FILES=$(find . -name "*.go" -not -path "./vendor/*" -not -path "./.git/*")

# Function to check if a file contains a Provider interface definition
check_provider_interface() {
    local file="$1"
    
    echo -e "${BLUE}Checking $file for Provider interface...${NC}"
    
    # Look for Provider interface definition
    if grep -n "type.*Provider.*interface" "$file" >/dev/null 2>&1; then
        echo -e "${GREEN}  ✅ Found Provider interface definition${NC}"
        
        # Extract the interface block
        local interface_content
        interface_content=$(awk '/type.*Provider.*interface/,/^}/' "$file")
        
        # Check each required method
        local missing_methods=0
        for i in "${!REQUIRED_METHODS[@]}"; do
            local pattern="${REQUIRED_METHODS[$i]}"
            local description="${METHOD_DESCRIPTIONS[$i]}"
            
            if echo "$interface_content" | grep -E "$pattern" >/dev/null 2>&1; then
                echo -e "${GREEN}    ✅ $description${NC}"
            else
                echo -e "${RED}    ❌ Missing or incorrect: $description${NC}"
                ((missing_methods++))
            fi
        done
        
        if [ $missing_methods -gt 0 ]; then
            ((violations_found++))
            echo -e "${RED}  ❌ Provider interface is not compliant${NC}"
        else
            echo -e "${GREEN}  ✅ Provider interface is compliant${NC}"
        fi
        
        ((providers_found++))
    fi
}

# Function to check if a file contains a Provider implementation
check_provider_implementation() {
    local file="$1"
    
    # Look for struct types that might implement Provider
    local structs
    structs=$(grep -n "type.*struct" "$file" 2>/dev/null | cut -d: -f2 | sed 's/type \([A-Za-z0-9_]*\).*/\1/' || true)
    
    if [[ -z "$structs" ]]; then
        return
    fi
    
    echo -e "${BLUE}Checking $file for Provider implementations...${NC}"
    
    for struct in $structs; do
        echo -e "${BLUE}  Checking struct: $struct${NC}"
        
        # Check if this struct has methods that suggest it implements Provider
        local method_count=0
        local found_methods=()
        
        for i in "${!REQUIRED_METHODS[@]}"; do
            local method_name
            method_name=$(echo "${METHOD_DESCRIPTIONS[$i]}" | cut -d'(' -f1)
            
            # Look for method implementation
            if grep -E "func.*\*?$struct\).*$method_name" "$file" >/dev/null 2>&1; then
                found_methods+=("$method_name")
                ((method_count++))
            fi
        done
        
        if [ $method_count -gt 0 ]; then
            echo -e "${BLUE}    Found $method_count Provider methods:${NC}"
            for method in "${found_methods[@]}"; do
                echo -e "${GREEN}      ✅ $method${NC}"
            done
            
            if [ $method_count -eq 4 ]; then
                echo -e "${GREEN}    ✅ Complete Provider implementation${NC}"
                ((providers_found++))
            else
                echo -e "${YELLOW}    ⚠️  Partial Provider implementation ($method_count/4 methods)${NC}"
                echo -e "${YELLOW}    Missing methods:${NC}"
                
                for i in "${!METHOD_DESCRIPTIONS[@]}"; do
                    local method_name
                    method_name=$(echo "${METHOD_DESCRIPTIONS[$i]}" | cut -d'(' -f1)
                    
                    if [[ ! " ${found_methods[*]} " =~ " $method_name " ]]; then
                        echo -e "${YELLOW}      - ${METHOD_DESCRIPTIONS[$i]}${NC}"
                    fi
                done
                ((violations_found++))
            fi
        fi
    done
}

# Check each file
for file in $FILES; do
    if [[ ! -f "$file" ]]; then
        continue
    fi
    
    # Skip test files
    if [[ $file =~ _test\.go$ ]]; then
        continue
    fi
    
    # Check for interface definitions
    check_provider_interface "$file"
    
    # Check for implementations
    check_provider_implementation "$file"
done

echo
echo -e "${GREEN}📊 Interface Compliance Summary:${NC}"
echo -e "   Provider interfaces/implementations found: $providers_found"
echo -e "   Compliance violations: $violations_found"

if [ $violations_found -gt 0 ]; then
    echo
    echo -e "${RED}❌ INTERFACE COMPLIANCE VIOLATIONS FOUND!${NC}"
    echo -e "${YELLOW}All Provider implementations must include exactly these 4 methods:${NC}"
    for description in "${METHOD_DESCRIPTIONS[@]}"; do
        echo -e "   - $description"
    done
    echo
    echo -e "${YELLOW}Key requirements:${NC}"
    echo -e "   - Configure() must accept map[string]interface{} (not custom Config interface)"
    echo -e "   - CallFunction() must support unified dispatch pattern"
    echo -e "   - Schema() must return *Schema with all provider information"
    echo -e "   - Close() must properly cleanup resources"
    echo
    echo -e "${YELLOW}This ensures 100% compatibility with Kolumn core.${NC}"
    exit 1
fi

if [ $providers_found -eq 0 ]; then
    echo -e "${YELLOW}⚠️  No Provider interfaces or implementations found${NC}"
    echo -e "${YELLOW}This is normal for SDK-only packages without example implementations${NC}"
    exit 0
fi

echo -e "${GREEN}✅ All Provider implementations are compliant!${NC}"
exit 0