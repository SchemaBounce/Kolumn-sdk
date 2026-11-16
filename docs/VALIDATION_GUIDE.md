# Provider Validation Guide

## 4-Method Pattern and Validation

The Kolumn Provider SDK enforces a **strict 4-method RPC interface**. Configuration validation is **NOT** a separate RPC method but should be handled internally within the `Configure()` method.

## ❌ What Was Removed

The `ValidateConfig` method was removed from the Provider interface:

```go
// ❌ NO LONGER EXISTS - This method was removed to maintain 4-method pattern
func (p *Provider) ValidateConfig(ctx context.Context, config map[string]interface{}) *ConfigValidationResult
```

## ✅ Recommended Validation Patterns

### Pattern 1: Direct Validation in Configure

```go
func (p *MyProvider) Configure(ctx context.Context, config map[string]interface{}) error {
    // Step 1: Security validation (always do this first)
    validator := &security.InputSizeValidator{}
    if err := validator.ValidateConfigSize(config); err != nil {
        return security.NewSecureError(
            "configuration too large",
            fmt.Sprintf("config validation failed: %v", err),
            "CONFIG_TOO_LARGE",
        )
    }

    // Step 2: Required field validation
    host, ok := config["host"].(string)
    if !ok || host == "" {
        return security.NewSecureError(
            "missing required configuration",
            "host field is required and must be a non-empty string",
            "MISSING_HOST",
        )
    }

    port, ok := config["port"].(int)
    if !ok {
        return security.NewSecureError(
            "invalid configuration",
            "port field must be an integer",
            "INVALID_PORT",
        )
    }

    // Step 3: Business logic validation
    if port < 1 || port > 65535 {
        return security.NewSecureError(
            "invalid configuration",
            "port must be between 1 and 65535",
            "INVALID_PORT_RANGE",
        )
    }

    // Step 4: Apply configuration if validation passes
    p.host = host
    p.port = port

    return nil
}
```

### Pattern 2: Using Validation Framework

```go
func (p *MyProvider) Configure(ctx context.Context, config map[string]interface{}) error {
    // Use the SDK validation framework
    validators := map[string]validation.ValidationFunc{
        "host": validation.RequiredString(),
        "port": validation.PortNumber(),
        "database": validation.RequiredString(),
        "username": validation.OptionalString(),
        "password": validation.OptionalString(),
    }

    if err := validation.ValidateConfig(config, validators); err != nil {
        return fmt.Errorf("configuration validation failed: %w", err)
    }

    // Apply validated configuration
    return p.applyConfig(config)
}
```

### Pattern 3: Using BaseProvider Helper

```go
type MyProvider struct {
    *core.BaseProvider
    // your fields...
}

func NewMyProvider() *MyProvider {
    base := core.NewBaseProvider("my-provider")

    // Add validation rules to base provider
    base.AddValidationRules([]core.ConfigValidationRule{
        {
            Field:       "host",
            Required:    true,
            Type:        "string",
            Description: "Database host address",
            Custom:      validation.ValidateHost,
        },
        {
            Field:       "port",
            Required:    true,
            Type:        "int",
            Description: "Database port number",
            Custom:      validation.ValidatePort,
        },
    })

    return &MyProvider{
        BaseProvider: base,
    }
}

func (p *MyProvider) Configure(ctx context.Context, config map[string]interface{}) error {
    // Use BaseProvider validation helper
    if result := p.ValidateConfiguration(ctx, config); !result.IsValid() {
        return fmt.Errorf("configuration validation failed: %v", result.Errors)
    }

    // Apply configuration
    return p.applyConfig(config)
}
```

## Security Best Practices

### Always Validate Input Size First

```go
func (p *MyProvider) Configure(ctx context.Context, config map[string]interface{}) error {
    // SECURITY: Always validate input size first to prevent DoS attacks
    validator := &security.InputSizeValidator{}
    if err := validator.ValidateConfigSize(config); err != nil {
        return security.NewSecureError(
            "configuration too large",
            fmt.Sprintf("config validation failed: %v", err),
            "CONFIG_TOO_LARGE",
        )
    }

    // Continue with other validation...
}
```

### Use Secure Error Handling

```go
// ✅ CORRECT - Use SecureError for consistent error handling
return security.NewSecureError(
    "invalid configuration",        // User-friendly message
    "detailed internal message",    // Debug information
    "ERROR_CODE",                  // Machine-readable error code
)

// ❌ INCORRECT - Don't expose internal details
return fmt.Errorf("failed to connect to database at %s:%d with credentials %s:%s", host, port, user, pass)
```

## Common Validation Patterns

### Database Connection Validation

```go
func (p *DatabaseProvider) Configure(ctx context.Context, config map[string]interface{}) error {
    // Extract and validate connection parameters
    dsn, err := p.buildDSN(config)
    if err != nil {
        return fmt.Errorf("invalid connection configuration: %w", err)
    }

    // Test connection during configuration
    db, err := sql.Open(p.driverName, dsn)
    if err != nil {
        return fmt.Errorf("failed to open database connection: %w", err)
    }
    defer db.Close()

    if err := db.PingContext(ctx); err != nil {
        return fmt.Errorf("failed to ping database: %w", err)
    }

    // Store validated configuration
    p.dsn = dsn
    return nil
}
```

### API Client Validation

```go
func (p *APIProvider) Configure(ctx context.Context, config map[string]interface{}) error {
    // Validate API configuration
    endpoint, ok := config["endpoint"].(string)
    if !ok {
        return fmt.Errorf("endpoint is required")
    }

    apiKey, ok := config["api_key"].(string)
    if !ok {
        return fmt.Errorf("api_key is required")
    }

    // Validate endpoint URL
    if _, err := url.Parse(endpoint); err != nil {
        return fmt.Errorf("invalid endpoint URL: %w", err)
    }

    // Test API connectivity
    client := &http.Client{Timeout: 10 * time.Second}
    req, _ := http.NewRequestWithContext(ctx, "GET", endpoint+"/health", nil)
    req.Header.Set("Authorization", "Bearer "+apiKey)

    resp, err := client.Do(req)
    if err != nil {
        return fmt.Errorf("failed to connect to API: %w", err)
    }
    resp.Body.Close()

    if resp.StatusCode >= 400 {
        return fmt.Errorf("API returned error status: %d", resp.StatusCode)
    }

    // Store validated configuration
    p.endpoint = endpoint
    p.apiKey = apiKey
    return nil
}
```

## Migration from ValidateConfig

If you have existing code with ValidateConfig, migrate it like this:

### Before (DEPRECATED)

```go
func (p *MyProvider) ValidateConfig(ctx context.Context, config map[string]interface{}) *ConfigValidationResult {
    // This method no longer exists
    errors := []string{}
    warnings := []string{}

    if host, ok := config["host"].(string); !ok || host == "" {
        errors = append(errors, "host is required")
    }

    return &ConfigValidationResult{
        IsValid:  len(errors) == 0,
        Errors:   errors,
        Warnings: warnings,
    }
}

func (p *MyProvider) Configure(ctx context.Context, config map[string]interface{}) error {
    // Just apply config without validation
    return p.applyConfig(config)
}
```

### After (RECOMMENDED)

```go
func (p *MyProvider) Configure(ctx context.Context, config map[string]interface{}) error {
    // Move validation logic into Configure
    if host, ok := config["host"].(string); !ok || host == "" {
        return fmt.Errorf("host is required")
    }

    // Apply configuration after validation
    return p.applyConfig(config)
}
```

## Best Practices Summary

1. **✅ DO**: Validate within `Configure()` method
2. **✅ DO**: Use security validation helpers for input size limits
3. **✅ DO**: Test connectivity/authentication during configuration
4. **✅ DO**: Use `SecureError` for consistent error handling
5. **✅ DO**: Fail fast with clear error messages

6. **❌ DON'T**: Add ValidateConfig method to Provider interface
7. **❌ DON'T**: Skip input size validation
8. **❌ DON'T**: Expose sensitive data in error messages
9. **❌ DON'T**: Defer validation to runtime operations
10. **❌ DON'T**: Break the 4-method RPC pattern

## Reference

- [Core Provider Interface](../core/provider.go)
- [Validation Framework](../helpers/validation/)
- [Security Helpers](../helpers/security/)
- [Simple Provider Example](../examples/simple/provider.go)