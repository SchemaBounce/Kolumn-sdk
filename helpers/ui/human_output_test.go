package ui

import "testing"

func TestFormatResourceAction(t *testing.T) {
	plain := StyleOptions{}
	tests := []struct {
		name                       string
		resourceType, resourceName string
		action                     string
		want                       string
	}{
		{"type, name and action", "postgres_table", "users", "Creating...", "postgres_table.users: Creating..."},
		{"trims spaces", " postgres_table ", " users ", " Creating... ", "postgres_table.users: Creating..."},
		{"no name", "postgres_table", "", "Refreshing", "postgres_table: Refreshing"},
		{"no action", "postgres_table", "users", "", "postgres_table.users"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatResourceAction(tt.resourceType, tt.resourceName, tt.action, plain); got != tt.want {
				t.Errorf("FormatResourceAction() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatResourceActionColorsResourceByAction(t *testing.T) {
	colored := StyleOptions{UseColors: true}
	got := FormatResourceAction("postgres_table", "users", "Destroying", colored)
	want := BrightRed + "postgres_table.users" + Reset + ": Destroying"
	if got != want {
		t.Errorf("FormatResourceAction() = %q, want %q", got, want)
	}
}

func TestFormatResourceActionWithDuration(t *testing.T) {
	plain := StyleOptions{}
	if got, want := FormatResourceActionWithDuration("postgres_table", "users", "Creation complete", "1.2s", plain),
		"postgres_table.users: Creation complete [1.2s]"; got != want {
		t.Errorf("with duration = %q, want %q", got, want)
	}
	if got, want := FormatResourceActionWithDuration("postgres_table", "users", "Creation complete", "", plain),
		"postgres_table.users: Creation complete"; got != want {
		t.Errorf("empty duration = %q, want %q", got, want)
	}
}

// The deprecated names must keep returning exactly what the new names return,
// with and without colors, so existing callers see no change.
func TestDeprecatedResourceActionAliases(t *testing.T) {
	for _, opts := range []StyleOptions{{}, {UseColors: true}} {
		if got, want := FormatTerraformStyle("postgres_table", "users", "Updating", opts),
			FormatResourceAction("postgres_table", "users", "Updating", opts); got != want {
			t.Errorf("FormatTerraformStyle(UseColors=%v) = %q, want %q", opts.UseColors, got, want)
		}
		if got, want := FormatTerraformStyleWithDuration("postgres_table", "users", "Updated", "3s", opts),
			FormatResourceActionWithDuration("postgres_table", "users", "Updated", "3s", opts); got != want {
			t.Errorf("FormatTerraformStyleWithDuration(UseColors=%v) = %q, want %q", opts.UseColors, got, want)
		}
	}
}
