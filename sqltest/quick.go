//go:build go1.24

package sqltest

import (
	"io/fs"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Quick connects to a PostgreSQL database using environment variables,
// runs migrations, and returns a pgx connection pool.
//
// If you want a connection to the data pool without migrations,
// use sqltest.Empty as the files parameter.
//
// If a database already exists, it will be dropped and recreated.
// To do this as safe as possible by default the databases managed by sqltest use a "test" prefix.
func Quick(t testing.TB, files fs.FS) *pgxpool.Pool {
	t.Helper()
	migration := New(t, Options{
		Force: true,
		Files: files,
	})
	return migration.Setup(t.Context(), "")
}
