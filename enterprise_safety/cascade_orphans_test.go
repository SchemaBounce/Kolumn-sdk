package enterprise_safety

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// RunOrphanDetectionTest has no implementation. It must not touch the
// database, and it must not report a pass it did not earn.
func TestRunOrphanDetectionTestReportsNotImplemented(t *testing.T) {
	for _, provider := range []string{"postgres", "mysql"} {
		db, rec := newRecordingDB(t)
		f := NewCascadeDeleteTestFramework(provider)

		result := f.RunOrphanDetectionTest(context.Background(), db)

		if result.Success {
			t.Errorf("%s: RunOrphanDetectionTest reported success without testing anything", provider)
		}
		if !strings.Contains(result.Error, "not implemented") {
			t.Errorf("%s: result.Error = %q, want it to say the test is not implemented", provider, result.Error)
		}
		if got := rec.statements(); len(got) != 0 {
			t.Errorf("%s: RunOrphanDetectionTest ran SQL: %q", provider, queries(got))
		}
		if len(f.TestResults) != 1 || f.Metrics.FailedTests != 1 || f.Metrics.PassedTests != 0 {
			t.Errorf("%s: recorded %d results, %d passed, %d failed; want 1 result counted as failed",
				provider, len(f.TestResults), f.Metrics.PassedTests, f.Metrics.FailedTests)
		}
	}
}

// No statement this package can run deletes rows: a DELETE whose WHERE is a
// placeholder is invalid SQL, and one without a WHERE empties a user's table.
// Any DELETE added later must come with its own bounded-condition test, and
// with a change to this one.
func TestPackageBuildsNoDeleteStatements(t *testing.T) {
	deleteStmt := regexp.MustCompile(`(?i)\bDELETE\s+FROM\b`)
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		src, err := os.ReadFile(file) // #nosec G304 -- a file of this package, from the glob above
		if err != nil {
			t.Fatal(err)
		}
		if loc := deleteStmt.FindIndex(src); loc != nil {
			line := 1 + strings.Count(string(src[:loc[0]]), "\n")
			t.Errorf("%s:%d builds a DELETE statement", file, line)
		}
	}
}
