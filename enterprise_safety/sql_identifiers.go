package enterprise_safety

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// maxSQLIdentifierLength is MySQL's identifier limit (64 characters).
// PostgreSQL's is 63 bytes; longer names are truncated by the server, so
// this check does not try to be stricter than the server.
const maxSQLIdentifierLength = 64

// sqlIdentifierPattern is the allowlist for schema, database and object
// names that the backup and cascade frameworks write into SQL text. It is
// the shape of an unquoted identifier that PostgreSQL and MySQL both accept:
// a letter or underscore, then letters, digits, underscores or dollar signs.
//
// The names are validated, not quoted, so a valid name produces exactly the
// SQL these frameworks produced before (quoting would make PostgreSQL stop
// folding mixed-case names to lower case). Anything that would need quoting,
// including every name that carries a quote, dot, space, comment or
// semicolon, is rejected before a query is built.
var sqlIdentifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_$]*$`)

// validateSQLIdentifier reports whether name is safe to write into SQL text
// as an unquoted identifier. kind names the field in the error message.
func validateSQLIdentifier(kind, name string) error {
	if name == "" {
		return fmt.Errorf("invalid %s: empty identifier", kind)
	}
	if len(name) > maxSQLIdentifierLength {
		return fmt.Errorf("invalid %s %q: longer than %d characters", kind, name, maxSQLIdentifierLength)
	}
	if !sqlIdentifierPattern.MatchString(name) {
		return fmt.Errorf("invalid %s %q: must match %s", kind, name, sqlIdentifierPattern.String())
	}
	return nil
}

// qualifiedSQLName validates qualifier (a PostgreSQL schema or MySQL
// database) and name, and returns them joined as "qualifier.name".
func qualifiedSQLName(qualifier, name string) (string, error) {
	if err := validateSQLIdentifier("schema or database name", qualifier); err != nil {
		return "", err
	}
	if err := validateSQLIdentifier("object name", name); err != nil {
		return "", err
	}
	return qualifier + "." + name, nil
}

// objectQualifier returns the field that qualifies an object name for the
// given provider: the schema for PostgreSQL, the database for MySQL.
func objectQualifier(providerType, schemaName, databaseName string) string {
	if providerType == "mysql" {
		return databaseName
	}
	return schemaName
}

// validateBackupFileName checks that name is a single, local path element,
// so joining it to the backup directory cannot leave that directory.
func validateBackupFileName(name string) error {
	if name == "" || name == "." || name == ".." ||
		strings.ContainsAny(name, `/\`) || !filepath.IsLocal(name) {
		return fmt.Errorf("invalid backup file name %q: must be a single file name inside the backup directory", name)
	}
	return nil
}
