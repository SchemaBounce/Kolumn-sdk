#!/bin/bash
#
# Validate provider schemas for completeness and correctness
# Ensures schema definitions follow Kolumn SDK patterns
#

set -euo pipefail

# Colors for output
readonly RED='\033[0;31m'
readonly YELLOW='\033[1;33m'
readonly GREEN='\033[0;32m'
readonly BLUE='\033[0;34m'
readonly NC='\033[0m' # No Color

echo -e "${GREEN}Validating provider schemas...${NC}"

violations_found=0
schemas_found=0

# Required schema fields for core compatibility
readonly REQUIRED_SCHEMA_FIELDS=(
    "SupportedFunctions"
    "ResourceTypes"
    "ConfigSchema"
)

# Supported unified functions
readonly UNIFIED_FUNCTIONS=(
    "CreateResource"
    "ReadResource"
    "UpdateResource"
    "DeleteResource"
    "DiscoverResources"
    "Ping"
)

# Function to check schema structure in Go files
check_schema_structure() {
    local file="$1"
    
    echo -e "${BLUE}Checking $file for schema definitions...${NC}"
    
    # Look for Schema struct definitions
    if grep -n "type.*Schema.*struct" "$file" >/dev/null 2>&1; then
        echo -e "${GREEN}  ✅ Found Schema struct definition${NC}"
        ((schemas_found++))
        
        # Extract the struct definition
        local schema_content
        schema_content=$(awk '/type.*Schema.*struct/,/^}/' "$file")
        
        # Check for required fields
        local missing_fields=0
        for field in "${REQUIRED_SCHEMA_FIELDS[@]}"; do
            if echo "$schema_content" | grep -q "$field"; then
                echo -e "${GREEN}    ✅ $field field present${NC}"
            else
                echo -e "${RED}    ❌ Missing required field: $field${NC}"
                ((missing_fields++))
            fi
        done
        
        if [ $missing_fields -gt 0 ]; then
            ((violations_found++))
            echo -e "${RED}  ❌ Schema struct is incomplete${NC}"
        else
            echo -e "${GREEN}  ✅ Schema struct has all required fields${NC}"
        fi
    fi
    
    # Look for Schema() method implementations
    if grep -n "func.*Schema.*\*.*Schema.*error" "$file" >/dev/null 2>&1; then
        echo -e "${GREEN}  ✅ Found Schema() method implementation${NC}"
        
        # Check if it returns proper schema with required functions
        local schema_method_content
        schema_method_content=$(awk '/func.*Schema.*\*.*Schema.*error/,/^}/' "$file")
        
        # Check for SupportedFunctions assignment
        if echo "$schema_method_content" | grep -q "SupportedFunctions"; then
            echo -e "${GREEN}    ✅ Sets SupportedFunctions${NC}"
            
            # Check for unified functions
            local functions_found=0
            for func in "${UNIFIED_FUNCTIONS[@]}"; do
                if echo "$schema_method_content" | grep -q "\"$func\""; then
                    echo -e "${GREEN}      ✅ Supports $func${NC}"
                    ((functions_found++))
                fi
            done
            
            if [ $functions_found -lt 2 ]; then
                echo -e "${YELLOW}    ⚠️  Only $functions_found unified functions found${NC}"
                echo -e "${YELLOW}       Consider supporting more functions: ${UNIFIED_FUNCTIONS[*]}${NC}"
            fi
        else
            echo -e "${YELLOW}    ⚠️  No SupportedFunctions assignment found${NC}"
        fi
        
        # Check for ResourceTypes assignment
        if echo "$schema_method_content" | grep -q "ResourceTypes"; then
            echo -e "${GREEN}    ✅ Sets ResourceTypes${NC}"
        else
            echo -e "${YELLOW}    ⚠️  No ResourceTypes assignment found${NC}"
        fi
        
        # Check for ConfigSchema assignment
        if echo "$schema_method_content" | grep -q "ConfigSchema"; then
            echo -e "${GREEN}    ✅ Sets ConfigSchema${NC}"
        else
            echo -e "${YELLOW}    ⚠️  No ConfigSchema assignment found${NC}"
        fi
    fi
}

# Function to check for object schema definitions (CREATE/DISCOVER)
check_object_schemas() {
    local file="$1"
    
    # Look for CreateObjectSchema or DiscoverObjectSchema
    if grep -n "CreateObjectSchema\|DiscoverObjectSchema" "$file" >/dev/null 2>&1; then
        echo -e "${GREEN}  ✅ Found object schema definitions${NC}"
        
        # Check for required schema properties
        local schema_lines
        schema_lines=$(grep -n "CreateObjectSchema\|DiscoverObjectSchema" "$file")
        
        while IFS= read -r line; do
            local line_num
            line_num=$(echo "$line" | cut -d: -f1)
            echo -e "${BLUE}    Line $line_num: Object schema definition${NC}"
        done <<< "$schema_lines"
        
        # Check for Properties field
        if grep -q "Properties.*map\|Properties.*\[\]" "$file"; then
            echo -e "${GREEN}    ✅ Defines Properties${NC}"
        else
            echo -e "${YELLOW}    ⚠️  No Properties field found${NC}"
        fi
        
        # Check for Examples
        if grep -q "Examples\|Example" "$file"; then
            echo -e "${GREEN}    ✅ Includes Examples${NC}"
        else
            echo -e "${YELLOW}    ⚠️  No Examples found${NC}"
        fi
        
        # Check for Description
        if grep -q "Description" "$file"; then
            echo -e "${GREEN}    ✅ Includes Description${NC}"
        else
            echo -e "${YELLOW}    ⚠️  No Description found${NC}"
        fi
    fi
}

# Function to validate JSON schema files
validate_json_schemas() {
    local schema_files
    schema_files=$(find . -name "*.json" -path "*/schema/*" -o -name "*schema*.json" | head -10)
    
    if [[ -n "$schema_files" ]]; then
        echo -e "${BLUE}Validating JSON schema files...${NC}"
        
        for schema_file in $schema_files; do
            echo -e "${BLUE}  Checking $schema_file...${NC}"
            
            # Validate JSON syntax
            if jq . "$schema_file" >/dev/null 2>&1; then
                echo -e "${GREEN}    ✅ Valid JSON syntax${NC}"
                
                # Check for JSON Schema properties
                if jq -e '.properties' "$schema_file" >/dev/null 2>&1; then
                    echo -e "${GREEN}    ✅ Contains properties definition${NC}"
                else
                    echo -e "${YELLOW}    ⚠️  No properties definition found${NC}"
                fi
                
                if jq -e '.required' "$schema_file" >/dev/null 2>&1; then
                    echo -e "${GREEN}    ✅ Defines required fields${NC}"
                else
                    echo -e "${YELLOW}    ⚠️  No required fields defined${NC}"
                fi
                
            else
                echo -e "${RED}    ❌ Invalid JSON syntax${NC}"
                ((violations_found++))
            fi
            
            ((schemas_found++))
        done
    fi
}

# Function to check for schema documentation
check_schema_documentation() {
    local file="$1"
    
    # Look for schema-related comments
    if grep -n "// Schema\|//.*schema" "$file" >/dev/null 2>&1; then
        echo -e "${GREEN}  ✅ Contains schema documentation${NC}"
    else
        echo -e "${YELLOW}  ⚠️  No schema documentation found${NC}"
    fi
    
    # Check for field documentation in structs
    if grep -A5 -B5 "type.*Schema.*struct" "$file" | grep -q "//"; then
        echo -e "${GREEN}  ✅ Schema fields are documented${NC}"
    else
        echo -e "${YELLOW}  ⚠️  Schema fields lack documentation${NC}"
    fi
}

# Find all Go files
FILES=$(find . -name "*.go" -not -path "./vendor/*" -not -path "./.git/*")

for file in $FILES; do
    if [[ ! -f "$file" ]]; then
        continue
    fi
    
    # Skip test files for schema requirements
    if [[ $file =~ _test\.go$ ]]; then
        continue
    fi
    
    # Check for schema-related content
    if grep -q "Schema\|schema" "$file" 2>/dev/null; then
        check_schema_structure "$file"
        check_object_schemas "$file"
        check_schema_documentation "$file"
        echo
    fi
done

# Validate JSON schema files
validate_json_schemas

echo
echo -e "${GREEN}📊 Schema Validation Summary:${NC}"
echo -e "   Schemas found: $schemas_found"
echo -e "   Validation violations: $violations_found"

if [ $violations_found -gt 0 ]; then
    echo
    echo -e "${RED}❌ SCHEMA VALIDATION VIOLATIONS FOUND!${NC}"
    echo -e "${YELLOW}All provider schemas must include:${NC}"
    for field in "${REQUIRED_SCHEMA_FIELDS[@]}"; do
        echo -e "   - $field"
    done
    echo
    echo -e "${YELLOW}Schema() method should support unified functions:${NC}"
    for func in "${UNIFIED_FUNCTIONS[@]}"; do
        echo -e "   - $func"
    done
    echo
    echo -e "${YELLOW}This ensures compatibility with Kolumn core and proper documentation generation.${NC}"
    exit 1
fi

if [ $schemas_found -eq 0 ]; then
    echo -e "${YELLOW}⚠️  No schemas found${NC}"
    echo -e "${YELLOW}This is normal for SDK-only packages without provider implementations${NC}"
    exit 0
fi

echo -e "${GREEN}✅ All schemas are valid and complete!${NC}"
exit 0