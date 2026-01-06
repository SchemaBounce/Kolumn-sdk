package sqltemplates

import (
	"fmt"
	"strings"
	"sync"

	pongo2 "github.com/flosch/pongo2/v6"
)

// AdapterInfo describes the provider dialect being rendered.
type AdapterInfo struct {
	Name         string
	Capabilities map[string]bool
}

// MacroFunc renders provider-aware SQL fragments.
type MacroFunc func(adapter AdapterInfo, args ...interface{}) (string, error)

var (
	macroMu       sync.RWMutex
	macroRegistry = map[string]map[string]MacroFunc{}
)

func init() {
	RegisterMacro("*", "column", columnMacro)
	RegisterMacro("*", "identifier", identifierMacro)
	RegisterMacro("*", "relation", relationMacro)
	registerProviderMacroPacks()
}

// RegisterMacro registers a macro for a provider ("*" = all providers).
func RegisterMacro(provider, name string, fn MacroFunc) {
	macroMu.Lock()
	defer macroMu.Unlock()

	if provider == "" {
		provider = "*"
	}
	provider = strings.ToLower(provider)
	name = strings.ToLower(name)

	if macroRegistry[provider] == nil {
		macroRegistry[provider] = make(map[string]MacroFunc)
	}
	macroRegistry[provider][name] = fn
}

// Render compiles the SQL template without any additional context.
func Render(raw string, adapter AdapterInfo) (string, error) {
	return RenderWithContext(raw, adapter, nil)
}

// RenderWithContext compiles the SQL template with optional user context (resource handles, helpers, etc.).
func RenderWithContext(raw string, adapter AdapterInfo, templateCtx map[string]interface{}) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw, nil
	}

	tmpl, err := pongo2.FromString("{% autoescape off %}" + raw + "{% endautoescape %}")
	if err != nil {
		return "", err
	}

	ctx := pongo2.Context{}

	adapterName := strings.ToLower(adapter.Name)
	ctx["adapter"] = pongo2.Context{
		"name":         adapterName,
		"capabilities": adapter.Capabilities,
		"dispatch": func(args ...interface{}) interface{} {
			if len(args) == 0 {
				panic("adapter.dispatch expects macro name")
			}
			macroName := fmt.Sprint(args[0])
			fn := resolveMacroFunc(adapterName, macroName)
			if fn == nil {
				panic(fmt.Sprintf("adapter %s cannot dispatch macro %s", adapterName, macroName))
			}
			wrapped := macroWrapper(fn, adapter)
			callable, ok := wrapped.(func(...interface{}) string)
			if !ok {
				panic(fmt.Sprintf("macro wrapper did not return callable for %s", macroName))
			}
			return callable(args[1:]...)
		},
	}

	if templateCtx != nil {
		for k, v := range templateCtx {
			ctx[k] = v
		}
		if resources, ok := templateCtx["resources"].(map[string]interface{}); ok {
			ctx["object"] = func(key string) interface{} {
				if handle, ok := resources[key]; ok {
					return handle
				}
				panic(fmt.Sprintf("unknown resource handle: %s", key))
			}
		}
	}

	macroMu.RLock()
	defer macroMu.RUnlock()

	attachMacros := func(provider string) {
		for name, fn := range macroRegistry[strings.ToLower(provider)] {
			ctx[name] = macroWrapper(fn, adapter)
		}
	}

	attachMacros("*")
	attachMacros(adapterName)

	rendered, err := tmpl.Execute(ctx)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(rendered), nil
}

func macroWrapper(fn MacroFunc, adapter AdapterInfo) interface{} {
	return func(args ...interface{}) string {
		out, err := fn(adapter, args...)
		if err != nil {
			panic(err)
		}
		return out
	}
}

func columnMacro(adapter AdapterInfo, args ...interface{}) (string, error) {
	if len(args) != 1 {
		return "", fmt.Errorf("column() expects 1 argument, got %d", len(args))
	}
	return quoteIdentifier(adapter, fmt.Sprint(args[0])), nil
}

func identifierMacro(adapter AdapterInfo, args ...interface{}) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("identifier() requires at least one part")
	}
	parts := make([]string, len(args))
	for i, part := range args {
		parts[i] = quoteIdentifier(adapter, fmt.Sprint(part))
	}
	return strings.Join(parts, "."), nil
}

func relationMacro(adapter AdapterInfo, args ...interface{}) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("relation() requires table name")
	}

	if len(args) == 1 {
		if handle, ok := asTemplateHandle(args[0]); ok {
			schema := safeString(handle["schema"])
			table := safeString(handle["table"])
			if schema == "" {
				return quoteIdentifier(adapter, table), nil
			}
			return fmt.Sprintf("%s.%s", quoteIdentifier(adapter, schema), quoteIdentifier(adapter, table)), nil
		}
	}

	var schema string
	table := fmt.Sprint(args[len(args)-1])
	if len(args) > 1 {
		schema = fmt.Sprint(args[len(args)-2])
	}
	if schema == "" {
		return quoteIdentifier(adapter, table), nil
	}
	return fmt.Sprintf("%s.%s", quoteIdentifier(adapter, schema), quoteIdentifier(adapter, table)), nil
}

func quoteIdentifier(adapter AdapterInfo, ident string) string {
	ident = strings.TrimSpace(ident)
	left, right := identifierQuotes(adapter.Name)
	if ident == "" {
		return left + right
	}

	escaped := ident
	if right == left {
		escaped = strings.ReplaceAll(ident, right, right+right)
	} else {
		escaped = strings.ReplaceAll(ident, right, right+right)
	}
	return fmt.Sprintf("%s%s%s", left, escaped, right)
}

func identifierQuotes(provider string) (string, string) {
	switch strings.ToLower(provider) {
	case "mysql", "mariadb":
		return "`", "`"
	case "mssql":
		return "[", "]"
	default:
		return `"`, `"`
	}
}

func registerProviderMacroPacks() {
	registerPostgresMacros("postgres")
	registerPostgresMacros("redshift")
	registerSnowflakeMacros()
	registerMySQLMacros()
	registerMSSQLMacros()
}

func registerPostgresMacros(provider string) {
	RegisterMacro(provider, "limit_clause", func(adapter AdapterInfo, args ...interface{}) (string, error) {
		if len(args) != 1 {
			return "", fmt.Errorf("limit_clause expects 1 argument")
		}
		return fmt.Sprintf("LIMIT %s", args[0]), nil
	})

	RegisterMacro(provider, "date_add", func(adapter AdapterInfo, args ...interface{}) (string, error) {
		if len(args) != 3 {
			return "", fmt.Errorf("date_add expects unit, amount, expression")
		}
		unit := strings.ToLower(strings.TrimSpace(fmt.Sprint(args[0])))
		amount := strings.TrimSpace(fmt.Sprint(args[1]))
		expr := strings.TrimSpace(fmt.Sprint(args[2]))
		return fmt.Sprintf("(%s + INTERVAL '%s %s')", expr, amount, unit), nil
	})

	RegisterMacro(provider, "bool_literal", func(adapter AdapterInfo, args ...interface{}) (string, error) {
		if len(args) != 1 {
			return "", fmt.Errorf("bool_literal expects 1 argument")
		}
		switch strings.ToLower(strings.TrimSpace(safeString(args[0]))) {
		case "true", "1":
			return "TRUE", nil
		case "false", "0":
			return "FALSE", nil
		default:
			return "", fmt.Errorf("unsupported boolean literal: %s", args[0])
		}
	})

	RegisterMacro(provider, "recent_usage_expr", func(adapter AdapterInfo, args ...interface{}) (string, error) {
		if len(args) != 1 {
			return "", fmt.Errorf("recent_usage_expr expects alias argument")
		}
		alias := strings.TrimSpace(fmt.Sprint(args[0]))
		if alias == "" {
			return "", fmt.Errorf("recent_usage_expr alias cannot be empty")
		}
		return fmt.Sprintf("(%s.last_used_at > NOW() - INTERVAL '30 days')", alias), nil
	})

	RegisterMacro(provider, "current_timestamp", func(adapter AdapterInfo, args ...interface{}) (string, error) {
		if len(args) != 0 {
			return "", fmt.Errorf("current_timestamp expects no arguments")
		}
		return "CURRENT_TIMESTAMP", nil
	})
}

func registerSnowflakeMacros() {
	provider := "snowflake"
	RegisterMacro(provider, "limit_clause", func(adapter AdapterInfo, args ...interface{}) (string, error) {
		if len(args) != 1 {
			return "", fmt.Errorf("limit_clause expects 1 argument")
		}
		return fmt.Sprintf("LIMIT %s", args[0]), nil
	})

	RegisterMacro(provider, "date_add", func(adapter AdapterInfo, args ...interface{}) (string, error) {
		if len(args) != 3 {
			return "", fmt.Errorf("date_add expects unit, amount, expression")
		}
		unit := strings.ToUpper(strings.TrimSpace(safeString(args[0])))
		amount := strings.TrimSpace(safeString(args[1]))
		expr := strings.TrimSpace(safeString(args[2]))
		return fmt.Sprintf("DATEADD(%s, %s, %s)", unit, amount, expr), nil
	})

	RegisterMacro(provider, "bool_literal", func(adapter AdapterInfo, args ...interface{}) (string, error) {
		if len(args) != 1 {
			return "", fmt.Errorf("bool_literal expects 1 argument")
		}
		switch strings.ToLower(strings.TrimSpace(safeString(args[0]))) {
		case "true", "1":
			return "TRUE", nil
		case "false", "0":
			return "FALSE", nil
		default:
			return "", fmt.Errorf("unsupported boolean literal: %s", args[0])
		}
	})

	RegisterMacro(provider, "recent_usage_expr", func(adapter AdapterInfo, args ...interface{}) (string, error) {
		if len(args) != 1 {
			return "", fmt.Errorf("recent_usage_expr expects alias argument")
		}
		alias := strings.TrimSpace(fmt.Sprint(args[0]))
		if alias == "" {
			return "", fmt.Errorf("recent_usage_expr alias cannot be empty")
		}
		return fmt.Sprintf("(%s.last_used_at > DATEADD('day', -30, CURRENT_TIMESTAMP()))", alias), nil
	})

	RegisterMacro(provider, "current_timestamp", func(adapter AdapterInfo, args ...interface{}) (string, error) {
		if len(args) != 0 {
			return "", fmt.Errorf("current_timestamp expects no arguments")
		}
		return "CURRENT_TIMESTAMP()", nil
	})
}

func registerMySQLMacros() {
	provider := "mysql"

	RegisterMacro(provider, "limit_clause", func(adapter AdapterInfo, args ...interface{}) (string, error) {
		if len(args) != 1 {
			return "", fmt.Errorf("limit_clause expects 1 argument")
		}
		return fmt.Sprintf("LIMIT %s", args[0]), nil
	})

	RegisterMacro(provider, "date_add", func(adapter AdapterInfo, args ...interface{}) (string, error) {
		if len(args) != 3 {
			return "", fmt.Errorf("date_add expects unit, amount, expression")
		}
		unit := strings.ToUpper(strings.TrimSpace(safeString(args[0])))
		amount := strings.TrimSpace(safeString(args[1]))
		expr := strings.TrimSpace(safeString(args[2]))
		return fmt.Sprintf("DATE_ADD(%s, INTERVAL %s %s)", expr, amount, unit), nil
	})

	RegisterMacro(provider, "bool_literal", func(adapter AdapterInfo, args ...interface{}) (string, error) {
		if len(args) != 1 {
			return "", fmt.Errorf("bool_literal expects 1 argument")
		}
		switch strings.ToLower(strings.TrimSpace(safeString(args[0]))) {
		case "true", "1":
			return "TRUE", nil
		case "false", "0":
			return "FALSE", nil
		default:
			return "", fmt.Errorf("unsupported boolean literal: %s", args[0])
		}
	})

	RegisterMacro(provider, "recent_usage_expr", func(adapter AdapterInfo, args ...interface{}) (string, error) {
		if len(args) != 1 {
			return "", fmt.Errorf("recent_usage_expr expects alias argument")
		}
		alias := strings.TrimSpace(fmt.Sprint(args[0]))
		if alias == "" {
			return "", fmt.Errorf("recent_usage_expr alias cannot be empty")
		}
		return fmt.Sprintf("(%s.last_used_at > DATE_ADD(CURRENT_TIMESTAMP(), INTERVAL -30 DAY))", alias), nil
	})

	RegisterMacro(provider, "current_timestamp", func(adapter AdapterInfo, args ...interface{}) (string, error) {
		if len(args) != 0 {
			return "", fmt.Errorf("current_timestamp expects no arguments")
		}
		return "CURRENT_TIMESTAMP()", nil
	})
}

func asTemplateHandle(value interface{}) (map[string]interface{}, bool) {
	if value == nil {
		return nil, false
	}
	if handle, ok := value.(map[string]interface{}); ok {
		if _, exists := handle["__handle_type"]; exists {
			return handle, true
		}
	}
	return nil, false
}

func safeString(value interface{}) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

func registerMSSQLMacros() {
	provider := "mssql"
	RegisterMacro(provider, "limit_clause", func(adapter AdapterInfo, args ...interface{}) (string, error) {
		if len(args) != 1 {
			return "", fmt.Errorf("limit_clause expects 1 argument")
		}
		return fmt.Sprintf("OFFSET 0 ROWS FETCH NEXT %s ROWS ONLY", args[0]), nil
	})

	RegisterMacro(provider, "date_add", func(adapter AdapterInfo, args ...interface{}) (string, error) {
		if len(args) != 3 {
			return "", fmt.Errorf("date_add expects unit, amount, expression")
		}
		unit := strings.ToUpper(strings.TrimSpace(safeString(args[0])))
		amount := strings.TrimSpace(safeString(args[1]))
		expr := strings.TrimSpace(safeString(args[2]))
		return fmt.Sprintf("DATEADD(%s, %s, %s)", unit, amount, expr), nil
	})

	RegisterMacro(provider, "bool_literal", func(adapter AdapterInfo, args ...interface{}) (string, error) {
		if len(args) != 1 {
			return "", fmt.Errorf("bool_literal expects 1 argument")
		}
		switch strings.ToLower(strings.TrimSpace(safeString(args[0]))) {
		case "true", "1":
			return "1", nil
		case "false", "0":
			return "0", nil
		default:
			return "", fmt.Errorf("unsupported boolean literal: %s", args[0])
		}
	})

	RegisterMacro(provider, "recent_usage_expr", func(adapter AdapterInfo, args ...interface{}) (string, error) {
		if len(args) != 1 {
			return "", fmt.Errorf("recent_usage_expr expects alias argument")
		}
		alias := strings.TrimSpace(fmt.Sprint(args[0]))
		if alias == "" {
			return "", fmt.Errorf("recent_usage_expr alias cannot be empty")
		}
		return fmt.Sprintf("(%s.last_used_at > DATEADD(day, -30, GETDATE()))", alias), nil
	})

	RegisterMacro(provider, "current_timestamp", func(adapter AdapterInfo, args ...interface{}) (string, error) {
		if len(args) != 0 {
			return "", fmt.Errorf("current_timestamp expects no arguments")
		}
		return "CURRENT_TIMESTAMP", nil
	})
}

func resolveMacroFunc(providerName, macroName string) MacroFunc {
	macroMu.RLock()
	defer macroMu.RUnlock()

	name := strings.ToLower(macroName)
	if name == "" {
		return nil
	}

	provider := strings.ToLower(providerName)
	if provider != "" {
		if providerRegistry := macroRegistry[provider]; providerRegistry != nil {
			if fn := providerRegistry[name]; fn != nil {
				return fn
			}
		}
	}

	if global := macroRegistry["*"]; global != nil {
		if fn := global[name]; fn != nil {
			return fn
		}
	}

	return nil
}
