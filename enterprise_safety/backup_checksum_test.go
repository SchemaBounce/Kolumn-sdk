package enterprise_safety

import (
	"errors"
	"strings"
	"testing"
)

// The PostgreSQL checksum query aliases the table as t. PostgreSQL then
// rejects any other reference to the table ("invalid reference to
// FROM-clause entry for table"), so outside the FROM clause the query may
// name the table only through t. It must also not order by ctid, which
// changes when rows move (VACUUM FULL, CLUSTER, updates).
func TestPostgresChecksumQueryReferencesOnlyTheAlias(t *testing.T) {
	db, rec := newRecordingDB(t)
	f := NewBackupIntegrityFramework("postgres", t.TempDir())

	orders := ObjectReference{Type: "table", SchemaName: "public", Name: "orders"}
	checksum, rowCount, _, err := f.getTableDataInfo(db, orders)
	if err != nil {
		t.Fatalf("getTableDataInfo: %v", err)
	}
	if rowCount != 7 || checksum != "7" {
		t.Errorf("getTableDataInfo = checksum %q, rows %d; want the recorded 7 and 7", checksum, rowCount)
	}

	stmts := queries(rec.statements())
	if len(stmts) != 2 {
		t.Fatalf("statements = %q, want the count and the checksum", stmts)
	}
	query := stmts[1]
	const from = "FROM public.orders t"
	if strings.Count(query, from) != 1 {
		t.Fatalf("checksum query %q does not read %q exactly once", query, from)
	}
	rest := strings.Replace(query, from, "", 1)
	for _, ref := range []string{"orders", "public"} {
		if strings.Contains(rest, ref) {
			t.Errorf("checksum query names %q outside its FROM clause, which PostgreSQL "+
				"rejects once the table is aliased t:\n%s", ref, query)
		}
	}
	if strings.Contains(query, "ctid") {
		t.Errorf("checksum query orders by ctid, which changes when rows move:\n%s", query)
	}
	if !strings.Contains(query, "ORDER BY md5(CAST((t.*) AS text))") {
		t.Errorf("checksum query does not order by the row hash:\n%s", query)
	}
}

// A failing checksum query used to be dropped, leaving an empty checksum and
// no reason; the error must reach the caller.
func TestPostgresChecksumQueryErrorIsReturned(t *testing.T) {
	db, rec := newRecordingDB(t)
	rec.queryErr = func(query string) error {
		if strings.Contains(query, "string_agg") {
			return errors.New(`invalid reference to FROM-clause entry for table "orders"`)
		}
		return nil
	}
	f := NewBackupIntegrityFramework("postgres", t.TempDir())

	_, _, _, err := f.getTableDataInfo(db, ObjectReference{Type: "table", SchemaName: "public", Name: "orders"})
	if err == nil || !strings.Contains(err.Error(), "invalid reference") {
		t.Fatalf("getTableDataInfo error = %v, want the checksum query's error", err)
	}
}
