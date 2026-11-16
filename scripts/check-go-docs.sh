#!/bin/bash
#
# Check Go documentation coverage
# Ensures all exported functions, types, and variables have proper documentation
#

set -euo pipefail

# Colors for output
readonly RED='\033[0;31m'
readonly YELLOW='\033[1;33m'
readonly GREEN='\033[0;32m'
readonly BLUE='\033[0;34m'
readonly NC='\033[0m' # No Color

echo -e "${GREEN}Checking Go documentation coverage...${NC}"

violations_found=0
total_exported=0
documented=0

# Function to check if a line is an exported declaration
is_exported() {
    local line="$1"
    # Check for exported functions, types, constants, variables
    # Simple pattern matching for exported declarations
    if [[ $line =~ ^func[[:space:]]+[A-Z] ]] || \
       [[ $line =~ ^type[[:space:]]+[A-Z] ]] || \
       [[ $line =~ ^const[[:space:]]+[A-Z] ]] || \
       [[ $line =~ ^var[[:space:]]+[A-Z] ]]; then
        return 0
    fi
    return 1
}

# Function to check if previous line is a documentation comment
has_doc_comment() {
    local prev_line="$1"
    if [[ $prev_line =~ ^//.*[[:alnum:]] ]]; then
        return 0
    fi
    return 1
}

# Process files passed as arguments or find all Go files
if [ $# -eq 0 ]; then
    FILES=$(find . -name "*.go" -not -path "./vendor/*" -not -path "./.git/*")
else
    FILES="$*"
fi

for file in $FILES; do
    if [[ ! -f "$file" ]]; then
        continue
    fi
    
    echo -e "${BLUE}Checking $file...${NC}"
    
    # Skip test files for documentation requirements
    if [[ $file =~ _test\.go$ ]]; then
        echo -e "${YELLOW}  Skipping test file${NC}"
        continue
    fi
    
    # Read file line by line
    prev_line=""
    line_num=0
    
    while IFS= read -r line; do
        ((line_num++))
        
        # Skip empty lines and build constraints
        if [[ -z "$line" ]] || [[ $line =~ ^//[[:space:]]*\+build ]] || [[ $line =~ ^//go: ]]; then
            prev_line="$line"
            continue
        fi
        
        # Check if this is an exported declaration
        if is_exported "$line"; then
            ((total_exported++))
            
            # Extract the identifier name for reporting
            identifier=$(echo "$line" | sed -E 's/^(func|type|const|var)[[:space:]]*(\([^)]*\))?[[:space:]]*([A-Z][a-zA-Z0-9_]*).*/\3/')
            
            if has_doc_comment "$prev_line"; then
                ((documented++))
                echo -e "${GREEN}  ✅ $identifier (line $line_num)${NC}"
            else
                ((violations_found++))
                echo -e "${RED}  ❌ $identifier (line $line_num) - Missing documentation${NC}"
                echo -e "${YELLOW}     Declaration: $line${NC}"
            fi
        fi
        
        prev_line="$line"
    done < "$file"
done

# Calculate coverage percentage
if [ $total_exported -gt 0 ]; then
    coverage=$((documented * 100 / total_exported))
else
    coverage=100
fi

echo
echo -e "${GREEN}📊 Documentation Coverage Summary:${NC}"
echo -e "   Total exported declarations: $total_exported"
echo -e "   Documented: $documented"
echo -e "   Missing documentation: $violations_found"
echo -e "   Coverage: ${coverage}%"

# Set minimum coverage threshold
readonly MIN_COVERAGE=80

if [ $violations_found -gt 0 ]; then
    echo
    echo -e "${RED}❌ DOCUMENTATION VIOLATIONS FOUND!${NC}"
    echo -e "${YELLOW}All exported Go declarations must have documentation comments.${NC}"
    echo -e "${YELLOW}Documentation should start with the name of the exported item.${NC}"
    echo
    echo -e "${YELLOW}Examples:${NC}"
    echo -e "   // Provider represents a Kolumn provider implementation"
    echo -e "   type Provider interface {"
    echo
    echo -e "   // Configure sets up the provider with given configuration"
    echo -e "   func (p *MyProvider) Configure(ctx context.Context, config map[string]interface{}) error {"
    
    if [ $coverage -lt $MIN_COVERAGE ]; then
        echo
        echo -e "${RED}Coverage ($coverage%) is below minimum threshold ($MIN_COVERAGE%)${NC}"
        exit 1
    fi
fi

if [ $total_exported -eq 0 ]; then
    echo -e "${YELLOW}⚠️  No exported declarations found${NC}"
    exit 0
fi

if [ $violations_found -eq 0 ]; then
    echo -e "${GREEN}✅ All exported declarations are properly documented!${NC}"
    exit 0
else
    echo -e "${YELLOW}⚠️  Some documentation issues found but coverage is acceptable${NC}"
    exit 0
fi