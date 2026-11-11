// Package testdb provides isolated PostgreSQL databases for tests.
package testdb

import (
	"database/sql"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

var invalidChars = regexp.MustCompile(`[^a-z0-9_]+`)

// DSN creates an empty PostgreSQL schema for the current test in the
// database given by TEST_DSN and returns a data source name for the "pgx"
// driver that uses this schema. The schema is dropped after the test.
// The test is skipped if TEST_DSN is not set.
func DSN(t *testing.T) string {
	t.Helper()

	dsn := os.Getenv("TEST_DSN")
	if dsn == "" {
		t.Skip("TEST_DSN not set")
	}

	schema := "test_" + invalidChars.ReplaceAllString(strings.ToLower(t.Name()), "_")
	if len(schema) > 63 {
		schema = schema[:63]
	}

	admin, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()

	if _, err := admin.Exec(fmt.Sprintf("DROP SCHEMA IF EXISTS %[1]s CASCADE; CREATE SCHEMA %[1]s", schema)); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		admin, err := sql.Open("pgx", dsn)
		if err != nil {
			t.Error(err)
			return
		}
		defer admin.Close()

		if _, err := admin.Exec("DROP SCHEMA " + schema + " CASCADE"); err != nil {
			t.Error(err)
		}
	})

	config, err := pgx.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	config.RuntimeParams["search_path"] = schema

	return stdlib.RegisterConnConfig(config)
}
