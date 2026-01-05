package sqltemplates

import (
	"strings"
	"testing"
)

func TestRenderWithDefaultMacros(t *testing.T) {
	tpl := `SELECT {{ column("email") }} FROM {{ relation("public", "users") }}`
	sql, err := Render(tpl, AdapterInfo{Name: "postgres"})
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	want := "SELECT \"email\" FROM \"public\".\"users\""
	if sql != want {
		t.Fatalf("unexpected render result: got %q want %q", sql, want)
	}
}

func TestRenderWithProviderMacro(t *testing.T) {
	const provider = "custom-dialect"

	RegisterMacro(provider, "recent_usage_expr", func(adapter AdapterInfo, args ...interface{}) (string, error) {
		return "DATEADD('day', -30, CURRENT_TIMESTAMP())", nil
	})

	tpl := `{{ recent_usage_expr() }}`
	sql, err := Render(tpl, AdapterInfo{Name: provider})
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	if sql != "DATEADD('day', -30, CURRENT_TIMESTAMP())" {
		t.Fatalf("unexpected macro output: %q", sql)
	}
}

func TestProviderMacroPacks(t *testing.T) {
	tpl := `SELECT {{ bool_literal("true") }} {{ limit_clause(5) }} as stmt, {{ date_add("day", 30, "created_at") }} as future FROM t`

	if _, err := Render(tpl, AdapterInfo{Name: "postgres"}); err != nil {
		t.Fatalf("postgres pack failed: %v", err)
	}

	if _, err := Render(tpl, AdapterInfo{Name: "snowflake"}); err != nil {
		t.Fatalf("snowflake pack failed: %v", err)
	}

	if _, err := Render(tpl, AdapterInfo{Name: "mssql"}); err != nil {
		t.Fatalf("mssql pack failed: %v", err)
	}
}

func TestRenderWithResourceHandles(t *testing.T) {
	tpl := `
{% set users = object("postgres_table.users") %}
SELECT {{ users.id }} FROM {{ relation(users) }}
`
	ctx := map[string]interface{}{
		"resources": map[string]interface{}{
			"postgres_table.users": map[string]interface{}{
				"__handle_type":  "postgres_table",
				"schema":         "public",
				"table":          "users",
				"qualified_name": `"public"."users"`,
				"id":             `"public"."users"."id"`,
			},
		},
	}

	sql, err := RenderWithContext(tpl, AdapterInfo{Name: "postgres"}, ctx)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	want := `SELECT "public"."users"."id" FROM "public"."users"`
	if sql != want {
		t.Fatalf("unexpected render output: got %q want %q", sql, want)
	}
}

func TestRelationQuotingByProvider(t *testing.T) {
	handle := map[string]interface{}{
		"__handle_type": "postgres_table",
		"schema":        "demo",
		"table":         "users",
	}
	ctx := map[string]interface{}{
		"resources": map[string]interface{}{
			"table.users": handle,
		},
		"table": map[string]interface{}{
			"users": handle,
		},
	}
	t.Run("postgres", func(t *testing.T) {
		sql, err := RenderWithContext(`{{ relation(table.users) }}`, AdapterInfo{Name: "postgres"}, ctx)
		if err != nil {
			t.Fatalf("render failed: %v", err)
		}
		if sql != `"demo"."users"` {
			t.Fatalf("unexpected postgres relation: %s", sql)
		}
	})
	t.Run("mysql", func(t *testing.T) {
		sql, err := RenderWithContext(`{{ relation(table.users) }}`, AdapterInfo{Name: "mysql"}, ctx)
		if err != nil {
			t.Fatalf("render failed: %v", err)
		}
		if sql != "`demo`.`users`" {
			t.Fatalf("unexpected mysql relation: %s", sql)
		}
	})
	t.Run("mssql", func(t *testing.T) {
		sql, err := RenderWithContext(`{{ relation(table.users) }}`, AdapterInfo{Name: "mssql"}, ctx)
		if err != nil {
			t.Fatalf("render failed: %v", err)
		}
		if sql != "[demo].[users]" {
			t.Fatalf("unexpected mssql relation: %s", sql)
		}
	})
}

func TestRecentUsageExprMacros(t *testing.T) {
	for _, provider := range []string{"postgres", "snowflake", "mysql"} {
		t.Run(provider, func(t *testing.T) {
			sql, err := Render(`{{ recent_usage_expr("k") }}`, AdapterInfo{Name: provider})
			if err != nil {
				t.Fatalf("macro render failed: %v", err)
			}
			if !strings.Contains(sql, "last_used_at") {
				t.Fatalf("unexpected macro output: %s", sql)
			}
		})
	}
}

func TestAdapterDispatchMacros(t *testing.T) {
	tpl := `{% set cutoff = adapter.dispatch("date_add")("day", -7, adapter.dispatch("current_timestamp")()) %}SELECT {{ cutoff }}`

	sql, err := Render(tpl, AdapterInfo{Name: "postgres"})
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	if !strings.Contains(sql, "CURRENT_TIMESTAMP") {
		t.Fatalf("expected CURRENT_TIMESTAMP in rendered SQL, got %q", sql)
	}
	if !strings.Contains(sql, "INTERVAL") {
		t.Fatalf("expected INTERVAL arithmetic in rendered SQL, got %q", sql)
	}
}
