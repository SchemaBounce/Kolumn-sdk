#!/bin/bash
#
# Validate provider binary naming convention
# Ensures all provider binaries follow the kolumn-provider-{name} pattern
#

set -euo pipefail

# Colors for output
readonly RED='\033[0;31m'
readonly YELLOW='\033[1;33m'
readonly GREEN='\033[0;32m'
readonly BLUE='\033[0;34m'
readonly NC='\033[0m' # No Color

echo -e "${GREEN}Checking provider binary naming convention...${NC}"

violations_found=0
providers_checked=0

# Required binary naming pattern
readonly REQUIRED_PATTERN="^kolumn-provider-[a-z0-9][a-z0-9-]*[a-z0-9]$"

# Function to check binary name in go build or go install commands
check_binary_names_in_file() {
    local file="$1"
    
    echo -e "${BLUE}Checking $file for binary naming...${NC}"
    
    # Look for go build commands with -o flag
    local build_commands
    build_commands=$(grep -n "go build.*-o" "$file" 2>/dev/null || true)
    
    if [[ -n "$build_commands" ]]; then
        echo -e "${BLUE}  Found go build commands:${NC}"
        while IFS= read -r line; do
            local line_num
            line_num=$(echo "$line" | cut -d: -f1)
            local command
            command=$(echo "$line" | cut -d: -f2-)
            
            echo -e "${BLUE}    Line $line_num: $command${NC}"
            
            # Extract binary name from -o flag
            local binary_name
            binary_name=$(echo "$command" | sed -n 's/.*-o[[:space:]]*\([^[:space:]]*\).*/\1/p')
            
            if [[ -n "$binary_name" ]]; then
                # Remove path components and extensions
                binary_name=$(basename "$binary_name" .exe)
                
                if [[ $binary_name =~ $REQUIRED_PATTERN ]]; then
                    echo -e "${GREEN}      ✅ Binary name: $binary_name${NC}"
                else
                    echo -e "${RED}      ❌ Invalid binary name: $binary_name${NC}"
                    echo -e "${YELLOW}         Must match pattern: kolumn-provider-{name}${NC}"
                    ((violations_found++))
                fi
                ((providers_checked++))
            fi
        done <<< "$build_commands"
    fi
    
    # Look for main package declarations (suggests this will be a binary)
    if grep -q "^package main" "$file" 2>/dev/null; then
        echo -e "${BLUE}  Found main package - checking directory/file naming${NC}"
        
        # Get the directory name
        local dir_name
        dir_name=$(dirname "$file")
        dir_name=$(basename "$dir_name")
        
        # Get the file name without extension
        local file_name
        file_name=$(basename "$file" .go)
        
        # List of allowed non-provider binaries
        local ALLOWED_NON_PROVIDERS="kolumn-docs-gen kolumn-schema-gen"
        
        # Check if this is an allowed non-provider binary
        if [[ " $ALLOWED_NON_PROVIDERS " =~ " $dir_name " ]]; then
            echo -e "${GREEN}    ✅ Directory name: $dir_name (allowed non-provider binary)${NC}"
        # Check if directory follows naming convention
        elif [[ $dir_name =~ $REQUIRED_PATTERN ]]; then
            echo -e "${GREEN}    ✅ Directory name: $dir_name${NC}"
        elif [[ "$dir_name" == "." ]] || [[ "$dir_name" == "examples" ]] || [[ "$dir_name" == "cmd" ]]; then
            echo -e "${YELLOW}    ⚠️  Generic directory name: $dir_name${NC}"
        else
            echo -e "${RED}    ❌ Directory name should match: kolumn-provider-{name}${NC}"
            echo -e "${YELLOW}       Current: $dir_name${NC}"
            ((violations_found++))
        fi
        
        # For main.go files, check if they're in properly named directories
        if [[ "$file_name" == "main" ]]; then
            local parent_dir
            parent_dir=$(basename "$(dirname "$file")")
            
            # Allow main.go in provider directories, examples, or allowed non-provider binaries
            if [[ ! $parent_dir =~ $REQUIRED_PATTERN ]] && \
               [[ "$parent_dir" != "examples" ]] && \
               [[ ! " $ALLOWED_NON_PROVIDERS " =~ " $parent_dir " ]]; then
                echo -e "${YELLOW}    ⚠️  main.go should be in a kolumn-provider-{name} directory${NC}"
                echo -e "${YELLOW}       Current directory: $parent_dir${NC}"
            fi
        fi
        
        ((providers_checked++))
    fi
}

# Function to check Makefile or build scripts
check_build_scripts() {
    local makefiles
    makefiles=$(find . -name "Makefile" -o -name "*.mk" -o -name "build.sh" -o -name "*.sh" | grep -v ".git" || true)
    
    if [[ -n "$makefiles" ]]; then
        echo -e "${BLUE}Checking build scripts for binary naming...${NC}"
        
        for makefile in $makefiles; do
            echo -e "${BLUE}  Checking $makefile...${NC}"
            
            # Look for binary names in build targets
            local binary_refs
            binary_refs=$(grep -n "kolumn-provider" "$makefile" 2>/dev/null || true)
            
            if [[ -n "$binary_refs" ]]; then
                while IFS= read -r line; do
                    local line_num
                    line_num=$(echo "$line" | cut -d: -f1)
                    local content
                    content=$(echo "$line" | cut -d: -f2-)
                    
                    echo -e "${BLUE}    Line $line_num: $content${NC}"
                    
                    # Extract potential binary names
                    local binaries
                    binaries=$(echo "$content" | grep -oE "kolumn-provider-[a-z0-9-]+" || true)
                    
                    for binary in $binaries; do
                        if [[ $binary =~ $REQUIRED_PATTERN ]]; then
                            echo -e "${GREEN}      ✅ Binary reference: $binary${NC}"
                        else
                            echo -e "${RED}      ❌ Invalid binary reference: $binary${NC}"
                            ((violations_found++))
                        fi
                        ((providers_checked++))
                    done
                done <<< "$binary_refs"
            fi
        done
    fi
}

# Function to check for README documentation about naming
check_documentation() {
    local readmes
    readmes=$(find . -name "README.md" -o -name "*.md" | grep -v ".git" || true)
    
    if [[ -n "$readmes" ]]; then
        echo -e "${BLUE}Checking documentation for binary naming guidance...${NC}"
        
        for readme in $readmes; do
            if grep -q "kolumn-provider" "$readme" 2>/dev/null; then
                echo -e "${GREEN}  ✅ $readme mentions provider naming${NC}"
                
                # Check if it shows the correct pattern
                if grep -q "kolumn-provider-.*name" "$readme" 2>/dev/null; then
                    echo -e "${GREEN}    ✅ Shows correct naming pattern${NC}"
                else
                    echo -e "${YELLOW}    ⚠️  Should show kolumn-provider-{name} pattern${NC}"
                fi
            fi
        done
    fi
}

# Find all relevant files
FILES=$(find . -name "*.go" -o -name "main.go" | grep -E "(examples|cmd)" | head -20)

if [[ -z "$FILES" ]]; then
    echo -e "${YELLOW}⚠️  No example or cmd Go files found${NC}"
else
    for file in $FILES; do
        if [[ -f "$file" ]]; then
            check_binary_names_in_file "$file"
        fi
    done
fi

# Check build scripts
check_build_scripts

# Check documentation
check_documentation

echo
echo -e "${GREEN}📊 Binary Naming Summary:${NC}"
echo -e "   Providers checked: $providers_checked"
echo -e "   Naming violations: $violations_found"

if [ $violations_found -gt 0 ]; then
    echo
    echo -e "${RED}❌ BINARY NAMING VIOLATIONS FOUND!${NC}"
    echo -e "${YELLOW}All provider binaries MUST follow this pattern:${NC}"
    echo -e "   kolumn-provider-{name}"
    echo
    echo -e "${YELLOW}Examples of correct names:${NC}"
    echo -e "   - kolumn-provider-postgres"
    echo -e "   - kolumn-provider-mysql"
    echo -e "   - kolumn-provider-redis"
    echo -e "   - kolumn-provider-kafka"
    echo -e "   - kolumn-provider-s3"
    echo
    echo -e "${YELLOW}Examples of incorrect names:${NC}"
    echo -e "   - postgres-provider"
    echo -e "   - kolumn-postgres"
    echo -e "   - provider-postgres"
    echo -e "   - mydb-provider"
    echo
    echo -e "${YELLOW}This naming convention enables automatic discovery by Kolumn core.${NC}"
    exit 1
fi

if [ $providers_checked -eq 0 ]; then
    echo -e "${YELLOW}⚠️  No provider binaries found to check${NC}"
    echo -e "${YELLOW}This is normal for SDK-only packages${NC}"
    exit 0
fi

echo -e "${GREEN}✅ All provider binary names follow the correct convention!${NC}"
exit 0