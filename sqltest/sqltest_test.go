// +build integration

// This test file has two purposes:
// 1. Serve as an implementation example.
// 2. Test the package.
//
// NOTE: Please notice this file is using build tags, and to be executed you need to use
// 	go build -tags integration .
//
// If you've this on multiple packages and want to avoid caching, use:
// 	go test -v -race -count 1 -p 1 -tags=integration ./...
package sqltest_test

import (
	"context"
	"flag"
	"testing"

	"github.com/hatch-studio/pgtools/sqltest"
	"github.com/jackc/pgx/v4/pgxpool"
)

var force bool

func init() {
	flag.BoolVar(&force, "force", false, "Force cleaning the database")
}

// Database should live on your application code.
//
// Alternatively, you can copy the interface_test.go file contents, and use the following code
// to delimit a high-level API fox pgx methods:
// 	type Database struct {
// 		Postgres PGXInterface
// 	}
//
// By using this interface you can restrict access to pgx configuration and low-level APIs in
// your business logic packages.
type Database struct {
	// Postgres connection.
	Postgres *pgxpool.Pool
}

func TestMigration(t *testing.T) {
	ctx := context.Background()
	migration := sqltest.New(t, sqltest.Options{Force: force, Path: "testdata/migrations"})
	conn := migration.Setup(ctx, "") // Using environment variables instead of connString to configure tests.

	db := &Database{
		Postgres: conn,
	}

	// Run each test sequentially.
	// Please don't use t.Parallel() here to reduce the risk of wasting time due to side-effects,
	// unless we later try it, and it turns out to be a great idea.
	var tests = []struct {
		name string
		f    func(t *testing.T, db *Database)
	}{
		{"feature", testFeature},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.f(t, db)
		})
	}
}

// testFeature implements your integration tests.
// As the purpose of this is to document the migration process itself, it's a "no-op".
// Avoid t.Parallel() for your own sake.
func testFeature(t *testing.T, db *Database) {
	// Test a functionality here.
}
