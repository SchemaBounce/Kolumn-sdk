package enterprise_safety

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

// recordingDriver is a database/sql driver that records every statement it
// is asked to run. Queries return one row with the single value 7.
type recordedStmt struct {
	query string
	args  []interface{}
}

type recorder struct {
	mu      sync.Mutex
	stmts   []recordedStmt
	execErr error
}

func (r *recorder) record(query string, args []driver.NamedValue) {
	r.mu.Lock()
	defer r.mu.Unlock()
	vals := make([]interface{}, len(args))
	for i, a := range args {
		vals[i] = a.Value
	}
	r.stmts = append(r.stmts, recordedStmt{query: query, args: vals})
}

func (r *recorder) statements() []recordedStmt {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]recordedStmt(nil), r.stmts...)
}

var (
	recorders          sync.Map // DSN -> *recorder
	registerRecordOnce sync.Once
)

type recordingDriver struct{}

func (recordingDriver) Open(dsn string) (driver.Conn, error) {
	r, ok := recorders.Load(dsn)
	if !ok {
		return nil, errors.New("unknown recorder " + dsn)
	}
	return &recordingConn{r: r.(*recorder)}, nil
}

type recordingConn struct{ r *recorder }

func (c *recordingConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare not supported")
}
func (c *recordingConn) Close() error              { return nil }
func (c *recordingConn) Begin() (driver.Tx, error) { return nil, errors.New("no transactions") }

func (c *recordingConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	c.r.record(query, args)
	return &oneRow{}, nil
}

func (c *recordingConn) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	c.r.record(query, args)
	if c.r.execErr != nil {
		return nil, c.r.execErr
	}
	return driver.RowsAffected(0), nil
}

type oneRow struct{ done bool }

func (r *oneRow) Columns() []string { return []string{"n"} }
func (r *oneRow) Close() error      { return nil }
func (r *oneRow) Next(dest []driver.Value) error {
	if r.done {
		return io.EOF
	}
	r.done = true
	dest[0] = int64(7)
	return nil
}

func newRecordingDB(t *testing.T) (*sql.DB, *recorder) {
	t.Helper()
	registerRecordOnce.Do(func() { sql.Register("enterprise-safety-recorder", recordingDriver{}) })
	rec := &recorder{}
	dsn := t.Name()
	recorders.Store(dsn, rec)
	t.Cleanup(func() { recorders.Delete(dsn) })
	db, err := sql.Open("enterprise-safety-recorder", dsn)
	if err != nil {
		t.Fatalf("open recording db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db, rec
}

func queries(stmts []recordedStmt) []string {
	out := make([]string, len(stmts))
	for i, s := range stmts {
		out[i] = s.query
	}
	return out
}

// Names that must never reach SQL text.
var unsafeNames = []string{
	"",
	"users; DROP TABLE accounts",
	"users--",
	"a.b",
	`x"y`,
	"x`y",
	"x'y",
	"a b",
	"1users",
	"naïve",
	strings.Repeat("a", maxSQLIdentifierLength+1),
}

func TestValidateSQLIdentifier(t *testing.T) {
	valid := []string{"users", "_t", "Users", "order_items2", "a$b", strings.Repeat("a", maxSQLIdentifierLength)}
	for _, name := range valid {
		if err := validateSQLIdentifier("object name", name); err != nil {
			t.Errorf("validateSQLIdentifier(%q) = %v, want nil", name, err)
		}
	}
	for _, name := range unsafeNames {
		if err := validateSQLIdentifier("object name", name); err == nil {
			t.Errorf("validateSQLIdentifier(%q) = nil, want an error", name)
		}
	}
}

func TestQualifiedSQLName(t *testing.T) {
	got, err := qualifiedSQLName("public", "users")
	if err != nil || got != "public.users" {
		t.Fatalf("qualifiedSQLName(public, users) = %q, %v; want public.users, nil", got, err)
	}
	if _, err := qualifiedSQLName("public; DROP SCHEMA x", "users"); err == nil {
		t.Error("an unsafe qualifier was accepted")
	}
	if _, err := qualifiedSQLName("public", "users WHERE 1=1"); err == nil {
		t.Error("an unsafe object name was accepted")
	}
}

// Valid names must produce exactly the SQL the frameworks produced before.
func TestBackupFrameworkSQLUnchangedForValidNames(t *testing.T) {
	db, rec := newRecordingDB(t)

	pg := NewBackupIntegrityFramework("postgres", t.TempDir())
	counts := pg.countRelatedObjects(db, []ObjectReference{{Type: "table", SchemaName: "public", Name: "orders"}})
	if !reflect.DeepEqual(counts, []int{7}) {
		t.Fatalf("counts = %v, want [7]", counts)
	}
	if err := pg.deleteObject(db, ObjectReference{Type: "table", SchemaName: "public", Name: "users"}); err != nil {
		t.Fatalf("deleteObject: %v", err)
	}
	if err := pg.deleteObject(db, ObjectReference{Type: "view", SchemaName: "public", Name: "v_users"}); err != nil {
		t.Fatalf("deleteObject: %v", err)
	}

	my := NewBackupIntegrityFramework("mysql", t.TempDir())
	my.countRelatedObjects(db, []ObjectReference{{Type: "table", DatabaseName: "shop", Name: "orders"}})
	if err := my.deleteObject(db, ObjectReference{Type: "table", DatabaseName: "shop", Name: "users"}); err != nil {
		t.Fatalf("deleteObject: %v", err)
	}
	if err := my.deleteObject(db, ObjectReference{Type: "view", DatabaseName: "shop", Name: "v_users"}); err != nil {
		t.Fatalf("deleteObject: %v", err)
	}

	want := []string{
		"SELECT COUNT(*) FROM public.orders",
		"DROP TABLE IF EXISTS public.users CASCADE",
		"DROP VIEW IF EXISTS public.v_users CASCADE",
		"SELECT COUNT(*) FROM shop.orders",
		"DROP TABLE IF EXISTS shop.users",
		"DROP VIEW IF EXISTS shop.v_users",
	}
	if got := queries(rec.statements()); !reflect.DeepEqual(got, want) {
		t.Errorf("queries =\n%q\nwant\n%q", got, want)
	}
}

func TestBackupFrameworkRejectsUnsafeNames(t *testing.T) {
	for _, provider := range []string{"postgres", "mysql"} {
		for _, name := range unsafeNames {
			db, rec := newRecordingDB(t)
			f := NewBackupIntegrityFramework(provider, t.TempDir())
			bad := ObjectReference{Type: "table", SchemaName: "public", DatabaseName: "shop", Name: name}
			badQualifier := ObjectReference{Type: "table", SchemaName: name, DatabaseName: name, Name: "users"}

			if _, err := f.BackupObject(context.Background(), db, bad); err == nil {
				t.Errorf("%s: BackupObject(name %q) succeeded, want an error", provider, name)
			}
			if _, err := f.BackupObject(context.Background(), db, badQualifier); err == nil {
				t.Errorf("%s: BackupObject(qualifier %q) succeeded, want an error", provider, name)
			}
			if _, _, _, err := f.getTableDataInfo(db, bad); err == nil {
				t.Errorf("%s: getTableDataInfo(%q) succeeded, want an error", provider, name)
			}
			if err := f.deleteObject(db, bad); err == nil {
				t.Errorf("%s: deleteObject(%q) succeeded, want an error", provider, name)
			}
			if counts := f.countRelatedObjects(db, []ObjectReference{bad}); counts[0] != 0 {
				t.Errorf("%s: countRelatedObjects(%q) = %v, want [0]", provider, name, counts)
			}
			if got := rec.statements(); len(got) != 0 {
				t.Errorf("%s: name %q reached the database: %q", provider, name, queries(got))
			}
		}
	}
}

func TestCascadeFrameworkCatalogQueriesUseParameters(t *testing.T) {
	db, rec := newRecordingDB(t)
	injection := "x' OR '1'='1"

	pg := NewCascadeDeleteTestFramework("postgres")
	pg.countObject(db, ObjectInfo{Type: "view", SchemaName: "public", Name: injection})
	my := NewCascadeDeleteTestFramework("mysql")
	my.countObject(db, ObjectInfo{Type: "procedure", DatabaseName: "shop", Name: injection})

	got := rec.statements()
	want := []recordedStmt{
		{
			query: "SELECT COUNT(*) FROM information_schema.views WHERE table_schema = $1 AND table_name = $2",
			args:  []interface{}{"public", injection},
		},
		{
			query: "SELECT COUNT(*) FROM information_schema.routines WHERE routine_schema = ? AND routine_name = ? AND routine_type = 'PROCEDURE'",
			args:  []interface{}{"shop", injection},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("statements =\n%#v\nwant\n%#v", got, want)
	}
}

func TestCascadeFrameworkSQLUnchangedForValidNames(t *testing.T) {
	db, rec := newRecordingDB(t)
	ctx := context.Background()

	pg := NewCascadeDeleteTestFramework("postgres")
	if n := pg.countObject(db, ObjectInfo{Type: "table", SchemaName: "public", Name: "orders"}); n != 7 {
		t.Fatalf("countObject = %d, want 7", n)
	}
	for _, obj := range []ObjectInfo{
		{Type: "table", SchemaName: "public", Name: "users"},
		{Type: "view", SchemaName: "public", Name: "v_users"},
		{Type: "schema", Name: "scratch"},
	} {
		if err := pg.executeCascadeDelete(ctx, db, obj); err != nil {
			t.Fatalf("executeCascadeDelete(%+v): %v", obj, err)
		}
	}

	my := NewCascadeDeleteTestFramework("mysql")
	my.countObject(db, ObjectInfo{Type: "table", DatabaseName: "shop", Name: "orders"})
	for _, obj := range []ObjectInfo{
		{Type: "table", DatabaseName: "shop", Name: "users"},
		{Type: "view", DatabaseName: "shop", Name: "v_users"},
		{Type: "database", Name: "scratch"},
	} {
		if err := my.executeCascadeDelete(ctx, db, obj); err != nil {
			t.Fatalf("executeCascadeDelete(%+v): %v", obj, err)
		}
	}

	want := []string{
		"SELECT COUNT(*) FROM public.orders",
		"DROP TABLE IF EXISTS public.users CASCADE",
		"DROP VIEW IF EXISTS public.v_users CASCADE",
		"DROP SCHEMA IF EXISTS scratch CASCADE",
		"SELECT COUNT(*) FROM shop.orders",
		"DROP TABLE IF EXISTS shop.users",
		"DROP VIEW IF EXISTS shop.v_users",
		"DROP DATABASE IF EXISTS scratch",
	}
	if got := queries(rec.statements()); !reflect.DeepEqual(got, want) {
		t.Errorf("queries =\n%q\nwant\n%q", got, want)
	}
}

func TestCascadeFrameworkRejectsUnsafeNames(t *testing.T) {
	ctx := context.Background()
	for _, name := range unsafeNames {
		db, rec := newRecordingDB(t)
		pg := NewCascadeDeleteTestFramework("postgres")
		my := NewCascadeDeleteTestFramework("mysql")

		if n := pg.countObject(db, ObjectInfo{Type: "table", SchemaName: "public", Name: name}); n != 0 {
			t.Errorf("countObject(%q) = %d, want 0", name, n)
		}
		for _, c := range []struct {
			f   *CascadeDeleteTestFramework
			obj ObjectInfo
		}{
			{pg, ObjectInfo{Type: "table", SchemaName: "public", Name: name}},
			{pg, ObjectInfo{Type: "view", SchemaName: name, Name: "v"}},
			{pg, ObjectInfo{Type: "schema", Name: name}},
			{my, ObjectInfo{Type: "table", DatabaseName: "shop", Name: name}},
			{my, ObjectInfo{Type: "database", Name: name}},
		} {
			if err := c.f.executeCascadeDelete(ctx, db, c.obj); err == nil {
				t.Errorf("%s: executeCascadeDelete(%+v) succeeded, want an error", c.f.ProviderType, c.obj)
			}
		}
		if err := pg.cleanupIntentionalOrphans(ctx, db, []OrphanedResource{{SchemaName: "public", Name: name}}); err == nil {
			t.Errorf("cleanupIntentionalOrphans(%q) succeeded, want an error", name)
		}
		if got := rec.statements(); len(got) != 0 {
			t.Errorf("name %q reached the database: %q", name, queries(got))
		}
	}
}

func TestCleanupIntentionalOrphansReturnsExecErrors(t *testing.T) {
	db, rec := newRecordingDB(t)
	rec.execErr = errors.New("boom")
	f := NewCascadeDeleteTestFramework("postgres")

	err := f.cleanupIntentionalOrphans(context.Background(), db, []OrphanedResource{
		{SchemaName: "public", Name: "a"},
		{SchemaName: "public", Name: "b"},
	})
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("cleanupIntentionalOrphans error = %v, want it to carry the exec error", err)
	}
	// Every orphan is still attempted after the first failure.
	if got := len(rec.statements()); got != 2 {
		t.Errorf("statements run = %d, want 2", got)
	}
	if err := f.cleanupIntentionalOrphans(context.Background(), db, nil); err != nil {
		t.Errorf("cleanupIntentionalOrphans(nil) = %v, want nil", err)
	}
}

func TestSaveBackupToDiskRefusesPathTraversal(t *testing.T) {
	root := t.TempDir()
	f := NewBackupIntegrityFramework("postgres", filepath.Join(root, "backups"))

	for _, b := range []*BackupObject{
		{ProviderType: "postgres", ObjectType: "table", ObjectName: "../../escape"},
		{ProviderType: "postgres", ObjectType: "table", ObjectName: "x/../../escape"},
		{ProviderType: "../postgres", ObjectType: "table", ObjectName: "users"},
		{ProviderType: "postgres", ObjectType: `..\..`, ObjectName: "users"},
	} {
		if err := f.saveBackupToDisk(b); err == nil {
			t.Errorf("saveBackupToDisk(%+v) succeeded, want an error", b)
		}
	}
	// Nothing was written, not even the backup directory.
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("files were created under %s: %v", root, entries)
	}
}

func TestSaveBackupToDiskWritesPrivateFiles(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "backups")
	f := NewBackupIntegrityFramework("postgres", dir)
	backup := &BackupObject{
		ProviderType:    "postgres",
		ObjectType:      "table",
		ObjectName:      "users",
		BackupTimestamp: time.Now(),
	}
	if err := f.saveBackupToDisk(backup); err != nil {
		t.Fatalf("saveBackupToDisk: %v", err)
	}

	dirInfo, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if perm := dirInfo.Mode().Perm(); perm&0o077 != 0 {
		t.Errorf("backup directory mode = %v, want no group or other access", perm)
	}
	fileInfo, err := os.Stat(filepath.Join(dir, "postgres_table_users.json"))
	if err != nil {
		t.Fatal(err)
	}
	if perm := fileInfo.Mode().Perm(); perm != 0o600 {
		t.Errorf("backup file mode = %v, want 0600", perm)
	}

	loaded, err := f.loadAllBackups()
	if err != nil || len(loaded) != 1 || loaded[0].ObjectName != "users" {
		t.Errorf("loadAllBackups = %v, %v; want the one saved backup", loaded, err)
	}
}

func TestBackupHashesUnchanged(t *testing.T) {
	// Backup IDs and metadata checksums are persisted in backup files; they
	// must not change when the hash code is touched.
	id := generateBackupID(ObjectReference{Type: "table", DatabaseName: "db", SchemaName: "public", Name: "users"})
	if id != "704d8f11db0c" {
		t.Errorf("generateBackupID = %q, want the md5-derived 704d8f11db0c", id)
	}
	f := NewBackupIntegrityFramework("postgres", t.TempDir())
	sum := f.calculateMetadataChecksum(&BackupObject{Definition: "CREATE TABLE public.users (id int);", ObjectType: "table", ObjectName: "users"})
	if sum != "9ac1dc1ea7d56b099e59fdcab8cf3768" {
		t.Errorf("calculateMetadataChecksum = %q, want the md5 9ac1dc1ea7d56b099e59fdcab8cf3768", sum)
	}
}

func TestBuildOrphanDetectionQuery(t *testing.T) {
	pg := NewCascadeDeleteTestFramework("postgres")
	my := NewCascadeDeleteTestFramework("mysql")

	want := `
				SELECT COUNT(*)
				FROM public.comments c
				LEFT JOIN public.posts p ON c.post_id = p.id
				WHERE p.id IS NULL`
	if got := pg.buildOrphanDetectionQuery(ObjectInfo{SchemaName: "public", Name: "comments"}, "posts"); got != want {
		t.Errorf("postgres query =\n%s\nwant\n%s", got, want)
	}
	if got := my.buildOrphanDetectionQuery(ObjectInfo{DatabaseName: "public", Name: "comments"}, "posts"); got != want {
		t.Errorf("mysql query =\n%s\nwant\n%s", got, want)
	}

	for _, name := range unsafeNames {
		if got := pg.buildOrphanDetectionQuery(ObjectInfo{SchemaName: name, Name: "comments"}, "posts"); got != "" {
			t.Errorf("postgres schema %q built %q, want no query", name, got)
		}
		if got := my.buildOrphanDetectionQuery(ObjectInfo{DatabaseName: name, Name: "comments"}, "posts"); got != "" {
			t.Errorf("mysql database %q built %q, want no query", name, got)
		}
	}
}
