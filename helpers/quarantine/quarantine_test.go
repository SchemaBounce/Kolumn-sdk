package quarantine

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBuildName_TruncatesAndHashes(t *testing.T) {
	name := BuildName(NameOptions{
		Kind:      "table",
		Schema:    "very_long_schema_name_that_should_force_hashing",
		Name:      "some_object_with_an_equally_long_identifier",
		MaxLength: 32,
		Timestamp: time.Date(2025, 11, 12, 8, 24, 0, 0, time.UTC),
	})

	require.LessOrEqual(t, len(name), 32)
	require.Contains(t, name, "delete_table")
}

func TestAppendMetadata(t *testing.T) {
	now := time.Date(2025, 11, 12, 8, 24, 0, 0, time.UTC)
	state := AppendMetadataWithTime(nil, "kolumn_quarantine.delete_table", now)

	require.Equal(t, "kolumn_quarantine.delete_table", state["quarantine_location"])
	require.Equal(t, now.Format(time.RFC3339), state["quarantined_at"])
}

func TestResolveRelationTarget(t *testing.T) {
	config := map[string]interface{}{
		"schema": "core",
		"name":   "users",
	}

	state := map[string]interface{}{
		"schema": "ignored",
	}

	schema, name := ResolveRelationTarget(config, state, "", "", "public", "name")
	require.Equal(t, "core", schema)
	require.Equal(t, "users", name)

	// Fallback to dotted resource id
	schema, name = ResolveRelationTarget(nil, nil, "analytics.events", "", "public", "name")
	require.Equal(t, "analytics", schema)
	require.Equal(t, "events", name)

	// Ensure fallback schema is used when nothing else is available
	schema, name = ResolveRelationTarget(nil, nil, "events", "", "public")
	require.Equal(t, "public", schema)
	require.Equal(t, "events", name)
}
