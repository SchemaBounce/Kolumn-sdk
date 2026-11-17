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

// TestBuildName_MicrosecondPrecision_DistinguishesSameSecond is the regression
// for the prod failure where two quarantine snapshots of the same table fired
// inside the same wall-clock second (e.g. a `dedup` migration immediately
// followed by a `fix.update_rows` migration on the same target). Before the
// timestamp format was bumped to microsecond precision, both calls produced
// the byte-identical name `delete_table_20260428T010639Z_<schema>_<table>`
// and the second `CREATE TABLE ... AS TABLE ...` failed with
// `pq: relation "<...>" already exists`.
//
// Two BuildName calls with timestamps 1µs apart must produce DIFFERENT slugs.
func TestBuildName_MicrosecondPrecision_DistinguishesSameSecond(t *testing.T) {
	base := time.Date(2026, 4, 28, 1, 6, 39, 123_000_000, time.UTC)
	first := BuildName(NameOptions{
		Kind:      "table",
		Schema:    "schemabounce_core",
		Name:      "roles",
		Timestamp: base,
	})
	second := BuildName(NameOptions{
		Kind:      "table",
		Schema:    "schemabounce_core",
		Name:      "roles",
		Timestamp: base.Add(1 * time.Microsecond),
	})
	require.NotEqual(t, first, second,
		"BuildName must produce distinct names for timestamps 1µs apart, otherwise back-to-back migrations on the same table collide on the quarantine table name")

	// Sanity-check the format itself includes microsecond digits — drift back
	// to second precision would silently re-introduce the collision. The slug
	// is lowercased and non-alnum chars are collapsed to underscores, so the
	// raw timestamp `20260428T010639.123456Z` becomes `20260428t010639_123456z`.
	require.Regexp(t, `\d{8}t\d{6}_\d{6}z`, first,
		"timestamp segment must include 6 microsecond digits — second precision (no microsecond suffix) re-introduces same-second collisions")
}

// TestBuildName_MicrosecondPrecision_StableForSameInstant ensures the function
// is deterministic for a given input — two calls with identical inputs produce
// byte-identical output. Without this, the rollback path (which rebuilds the
// expected name from stored metadata to locate the snapshot) would break.
func TestBuildName_MicrosecondPrecision_StableForSameInstant(t *testing.T) {
	ts := time.Date(2026, 4, 28, 1, 6, 39, 123_456_000, time.UTC)
	opts := NameOptions{
		Kind:      "table",
		Schema:    "schemabounce_core",
		Name:      "workspace_quickstart_state",
		Timestamp: ts,
	}
	first := BuildName(opts)
	second := BuildName(opts)
	require.Equal(t, first, second,
		"BuildName must be deterministic for a given (Kind, Schema, Name, Timestamp); rollback metadata depends on regenerating the same slug")
}

// TestBuildName_OldFormatBackCompat — older quarantine names persisted in
// state files use the second-precision format `20060102T150405Z`. The bump
// to microsecond precision must not break the ability to PARSE/READ those
// older names; only NEW names use the new format. The current BuildName
// function only emits new names, but this test pins down the format string
// so any future revert is caught.
func TestBuildName_FormatPinsMicrosecondPrecision(t *testing.T) {
	// Same Schema/Name, two timestamps where the millisecond differs but
	// the second is identical — second-precision would tie, microsecond
	// precision must distinguish.
	ts1 := time.Date(2026, 4, 28, 1, 6, 39, 100_000_000, time.UTC)
	ts2 := time.Date(2026, 4, 28, 1, 6, 39, 200_000_000, time.UTC)
	a := BuildName(NameOptions{Kind: "table", Schema: "s", Name: "t", Timestamp: ts1})
	b := BuildName(NameOptions{Kind: "table", Schema: "s", Name: "t", Timestamp: ts2})
	require.NotEqual(t, a, b)
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
