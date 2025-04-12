//go:build go1.24

package sqltest_test

import (
	"os"
	"testing"
	"time"

	"github.com/henvic/pgtools/sqltest"
)

func TestQuick(t *testing.T) {
	t.Parallel()
	t.Run("ValidMigrations", func(t *testing.T) {
		t.Parallel()

		pool := sqltest.Quick(t, os.DirFS("example/testdata/migrations"))
		var tt time.Time
		if err := pool.QueryRow(t.Context(), "SELECT NOW();").Scan(&tt); err != nil {
			t.Errorf("cannot execute query: %v", err)
		}
		if tt.IsZero() {
			t.Error("time returned by pgx is zero")
		}
	})

	// Using an empty migrations directory
	t.Run("EmptyMigrations", func(t *testing.T) {
		t.Parallel()
		pool := sqltest.Quick(t, sqltest.Empty)
		var tt time.Time
		if err := pool.QueryRow(t.Context(), "SELECT NOW();").Scan(&tt); err != nil {
			t.Errorf("cannot execute query: %v", err)
		}
		if tt.IsZero() {
			t.Error("time returned by pgx is zero")
		}
	})
}

func TestDatabaseWithAnExceedinglyLongAndVerboseNameThatStretchesTheImagination(t *testing.T) {
	t.Parallel()

	// This test is to check that the database name is truncated to 63 characters
	// when creating the database.
	pool := sqltest.Quick(t, os.DirFS("example/testdata/migrations"))
	var dbName string
	if err := pool.QueryRow(t.Context(), "SELECT current_database();").Scan(&dbName); err != nil {
		t.Errorf("cannot get current database name: %v", err)
	}
	if dbName != "testdatabasewithanexceedinglylongandverbosenamethatstretch9f9b2" {
		t.Errorf("unexpected database name: %s", dbName)
	}
}
