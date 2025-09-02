# AST RPC Provider Interface

This document describes the AST (Abstract Syntax Tree) RPC provider interface for Kolumn's plugin architecture. The AST provider interface allows providers to expose SQL and HCL parsing, generation, and transformation capabilities through RPC.

## Overview

The AST provider interface extends the base `UniversalProvider` interface with AST-specific capabilities:

- **SQL/HCL Parsing**: Convert source code to Universal AST
- **SQL/HCL Generation**: Convert Universal AST to provider-specific source code
- **AST Transformation**: Transform AST between different dialects
- **AST Analysis**: Extract metadata, dependencies, and complexity metrics
- **AST Optimization**: Apply provider-specific optimizations
- **AST Validation**: Validate AST against provider capabilities
- **Capability Discovery**: Discover what AST operations a provider supports

## Architecture

```
┌─────────────────┐    RPC     ┌──────────────────────┐
│   Core Kolumn   │ ←──────→   │ kolumn-provider-*    │
│  (AST Client)   │            │   (AST Provider)     │
└─────────────────┘            └──────────────────────┘
        │                               │
        │        ASTProvider Interface: │
        │        • ParseSQL/ParseHCL    │
        │        • GenerateSQL/HCL      │
        │        • TransformAST         │
        │        • AnalyzeAST           │
        │        • OptimizeAST          │
        │        • ValidateAST          │
        │        • GetASTCapabilities   │
```

## Core Components

### 1. ASTProvider Interface

The main interface that providers implement:

```go
type ASTProvider interface {
    UniversalProvider
    
    // AST Parsing
    ParseSQL(ctx context.Context, req *ParseSQLRequest) (*ParseSQLResponse, error)
    ParseHCL(ctx context.Context, req *ParseHCLRequest) (*ParseHCLResponse, error)
    
    // AST Generation
    GenerateSQL(ctx context.Context, req *GenerateSQLRequest) (*GenerateSQLResponse, error)
    GenerateHCL(ctx context.Context, req *GenerateHCLRequest) (*GenerateHCLResponse, error)
    
    // AST Operations
    TransformAST(ctx context.Context, req *TransformASTRequest) (*TransformASTResponse, error)
    AnalyzeAST(ctx context.Context, req *AnalyzeASTRequest) (*AnalyzeASTResponse, error)
    OptimizeAST(ctx context.Context, req *OptimizeASTRequest) (*OptimizeASTResponse, error)
    ValidateAST(ctx context.Context, req *ValidateASTRequest) (*ValidateASTResponse, error)
    
    // Capability Discovery
    GetASTCapabilities() (*ASTCapabilities, error)
}
```

### 2. SerializableAST

Optimized AST representation for RPC transmission:

```go
type SerializableAST struct {
    RootID   string                      `json:"root_id"`
    Nodes    map[string]*SerializableNode `json:"nodes"`
    Metadata ASTMetadata                 `json:"metadata"`
    Checksum string                      `json:"checksum,omitempty"`
}
```

### 3. ASTCapabilities

Describes what AST operations a provider supports:

```go
type ASTCapabilities struct {
    SupportedLanguages []universal.Language `json:"supported_languages"`
    SupportedDialects  []string             `json:"supported_dialects"`
    CanParseSQL        bool                 `json:"can_parse_sql"`
    CanGenerateSQL     bool                 `json:"can_generate_sql"`
    CanTransformAST    bool                 `json:"can_transform_ast"`
    TransformTargets   []string             `json:"transform_targets,omitempty"`
    // ... more capabilities
}
```

## Usage Examples

### Basic SQL Parsing

```go
provider := NewExampleASTProvider("postgres")
ctx := context.Background()

// Configure provider
config := map[string]interface{}{
    "dialect": "postgres",
    "cache_size": 100.0,
}
provider.Configure(ctx, config)

// Parse SQL
req := &ParseSQLRequest{
    SQL:     "SELECT id, name FROM users WHERE age > 21",
    Dialect: "postgres",
    Options: &ParseOptions{
        Strict:       false,
        AllowPartial: false,
    },
    EnableCache: true,
}

resp, err := provider.ParseSQL(ctx, req)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Parsed %d nodes in %v\n", len(resp.AST.Nodes), resp.ParseDuration)
```

### AST Transformation Between Dialects

```go
// Parse PostgreSQL SQL
parseReq := &ParseSQLRequest{
    SQL:     "SELECT * FROM users LIMIT 10",
    Dialect: "postgres",
}
parseResp, _ := provider.ParseSQL(ctx, parseReq)

// Transform to MySQL dialect
transformReq := &TransformASTRequest{
    AST:           parseResp.AST,
    SourceDialect: "postgres",
    TargetDialect: "mysql",
    Options: &TransformOptions{
        PreserveFunctionality: true,
        StrictCompatibility:   false,
    },
}

transformResp, _ := provider.TransformAST(ctx, transformReq)

// Generate MySQL SQL
genReq := &GenerateSQLRequest{
    AST:     transformResp.TransformedAST,
    Dialect: "mysql",
}
genResp, _ := provider.GenerateSQL(ctx, genReq)

fmt.Printf("MySQL SQL: %s\n", genResp.SQL)
```

### AST Analysis and Optimization

```go
// Analyze AST
analyzeReq := &AnalyzeASTRequest{
    AST: parseResp.AST,
    AnalysisTypes: []AnalysisType{
        AnalysisTypeDependencies,
        AnalysisTypeComplexity,
        AnalysisTypePerformance,
    },
}

analyzeResp, _ := provider.AnalyzeAST(ctx, analyzeReq)
fmt.Printf("Complexity: %d\n", analyzeResp.Analysis.CyclomaticComplexity)

// Optimize AST
optimizeReq := &OptimizeASTRequest{
    AST: parseResp.AST,
    OptimizationTypes: []OptimizationType{
        OptimizationTypeQueryRewrite,
        OptimizationTypeConstantFolding,
    },
    Options: &OptimizationOptions{
        Level: OptimizationLevelIntermediate,
    },
}

optimizeResp, _ := provider.OptimizeAST(ctx, optimizeReq)
fmt.Printf("Applied %d optimizations\n", len(optimizeResp.AppliedOptimizations))
```

## Performance Features

### Caching

The AST interface supports comprehensive caching for performance:

```go
req := &ParseSQLRequest{
    SQL:         "SELECT * FROM users",
    EnableCache: true,
    CacheKey:    "users_select_all", // Optional - auto-generated if not provided
}

resp, _ := provider.ParseSQL(ctx, req)
fmt.Printf("Cache hit: %v\n", resp.CacheHit)
```

### Checksums for Integrity

All AST objects include SHA-256 checksums for integrity verification:

```go
ast := &SerializableAST{
    RootID: "root",
    Nodes:  nodes,
    // ... other fields
}

serializer := NewASTSerializer(100)
checksum, _ := serializer.generateASTChecksum(ast)
ast.Checksum = checksum
```

## Provider Implementation

### 1. Implement the ASTProvider Interface

```go
type MyASTProvider struct {
    dialect    string
    serializer *ASTSerializer
}

func (p *MyASTProvider) ParseSQL(ctx context.Context, req *ParseSQLRequest) (*ParseSQLResponse, error) {
    // 1. Parse SQL using your provider's parser
    // 2. Convert to SerializableAST
    // 3. Handle caching if enabled
    // 4. Return response with timing and error information
}

func (p *MyASTProvider) GenerateSQL(ctx context.Context, req *GenerateSQLRequest) (*GenerateSQLResponse, error) {
    // 1. Convert SerializableAST to your internal representation
    // 2. Generate SQL using your provider's generator
    // 3. Apply formatting options
    // 4. Return response with timing
}

// ... implement other methods
```

### 2. Register RPC Handlers

```go
server := NewASTProviderServer(myProvider, logger)

// Register with go-plugin
rpcServer.RegisterName("ASTProvider", server)
```

### 3. Declare AST Capabilities

```go
func (p *MyASTProvider) GetASTCapabilities() (*ASTCapabilities, error) {
    return &ASTCapabilities{
        SupportedLanguages: []universal.Language{universal.LanguageSQL},
        SupportedDialects:  []string{"postgres", "mysql"},
        CanParseSQL:        true,
        CanGenerateSQL:     true,
        CanTransformAST:    true,
        TransformTargets:   []string{"mysql", "sqlite"},
        SupportsCaching:    true,
        // ... other capabilities
    }, nil
}
```

## Provider Registry

Use the registry to discover and manage AST providers:

```go
registry := NewASTProviderRegistry()

// Register providers
registry.RegisterProvider("postgres", postgresProvider)
registry.RegisterProvider("mysql", mysqlProvider)

// Find providers with specific capabilities
sqlProviders := registry.FindProvidersWithCapability("parse_sql")
transformProviders := registry.FindProvidersWithCapability("transform_ast")

// Get provider capabilities
capabilities, exists := registry.GetCapabilities("postgres")
```

## Error Handling

The AST interface provides comprehensive error reporting:

### Parse Errors

```go
type ParseError struct {
    Message     string               `json:"message"`
    Severity    ErrorSeverity        `json:"severity"`     // error, warning, info, hint
    Position    *universal.Position  `json:"position,omitempty"`
    Code        string               `json:"code,omitempty"`
    Category    ErrorCategory        `json:"category,omitempty"`
    Suggestions []string             `json:"suggestions,omitempty"`
}
```

### Compatibility Issues

```go
type CompatibilityIssue struct {
    Feature     string               `json:"feature"`
    Message     string               `json:"message"`
    Severity    ErrorSeverity        `json:"severity"`
    Workaround  string               `json:"workaround,omitempty"`
    Position    *universal.Position  `json:"position,omitempty"`
}
```

## Best Practices

### 1. Provider Implementation

- **Always validate input**: Check request parameters before processing
- **Use caching effectively**: Implement caching for expensive operations
- **Provide meaningful errors**: Include position information and suggestions
- **Support incremental parsing**: Allow partial parsing for better UX
- **Implement capability discovery**: Accurately report what your provider supports

### 2. Performance Optimization

- **Use SerializableAST efficiently**: Avoid deep copying when possible
- **Implement smart caching**: Cache at both AST and SQL level
- **Parallelize when possible**: Support parallel parsing for large inputs
- **Monitor performance**: Track parsing and generation times

### 3. Error Recovery

- **Graceful degradation**: Continue parsing even with syntax errors when possible
- **Rich error information**: Provide detailed error messages with context
- **Suggestions**: Include fix suggestions in error responses
- **Validation**: Validate AST before expensive operations

### 4. Testing

- **Test all capabilities**: Verify each declared capability actually works
- **Test edge cases**: Handle malformed input gracefully
- **Performance testing**: Ensure operations complete within reasonable time
- **Compatibility testing**: Test transformations between all supported dialects

## Integration with Kolumn Core

AST providers integrate seamlessly with Kolumn's core functionality:

### SQL Command Integration

```bash
# Parse and validate SQL files
kolumn sql validate --provider postgres

# Transform SQL between dialects
kolumn sql transform --from postgres --to mysql views/

# Optimize SQL for specific provider
kolumn sql optimize --provider postgres --level aggressive
```

### Cross-Provider Operations

The AST interface enables powerful cross-provider operations:

1. **Parse SQL in one provider dialect**
2. **Transform to Universal AST**
3. **Optimize for different provider**
4. **Generate SQL in target dialect**
5. **Validate against target provider capabilities**

This enables Kolumn to provide truly universal SQL management across the entire data stack.

## Conclusion

The AST RPC provider interface is a powerful extension to Kolumn's plugin architecture that enables:

- **Universal SQL parsing and generation** across all data providers
- **Dialect transformation** for cross-provider compatibility  
- **Advanced analysis and optimization** capabilities
- **Comprehensive error handling and validation**
- **High-performance operations** with caching and parallelization

By implementing the ASTProvider interface, providers can expose their SQL dialect expertise while benefiting from Kolumn's unified AST infrastructure.