// Package sqltest makes it easy to write tests using pgx and tern.
package sqltest

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/jackc/tern/migrate"
)

var (
	// DatabasePrefix defines a prefix for the database name.
	// It is used to mitigate the risk of running migration and tests on the wrong database.
	DatabasePrefix = "test_"

	// SchemaVersionTable where tern saves the version of the current migration in PostgreSQL.
	SchemaVersionTable = "schema_version"
)

// New migration to use with a test.
func New(t testing.TB, o Options) *Migration {
	return &Migration{
		Options: o,
		t:       t,
	}
}

// Options for the migration.
type Options struct {
	// Force clean the database if it's dirty.
	Force bool

	// SkipTeardown stops the Teardown function being registered with testing cleanup.
	// You can use this to debug migration after running a specific test.
	//
	// Be aware that you'll encounter errors if you try to run a migration multiple times without
	// a proper cleaning up. To forcefully clean up the database, you can use the force option.
	SkipTeardown bool

	// Path to the migration files.
	Path string
}

// Migration simplifies avlidadting the migration process, and setting up a test database
// for executing your PostgreSQL-based tests on.
type Migration struct {
	Options Options

	t        testing.TB
	migrator *migrate.Migrator
}

// Setup the migration.
// This function returns a pgx pool that can be used to connect to the database.
// If something fails, t.Fatal is called.
//
// It register the Teardown function with testing.TB to clean up the database once the
// tests are over by default, but this can be disabled by setting the SkipTeardown option.
//
// If you're using PostgreSQL environment variables, you should pass an empty string as the
// connection string, as in:
// 	pool := m.Setup(context.Background(), "")
//
// Reference for configuring the PostgreSQL client with environment variables:
// https://www.postgresql.org/docs/current/libpq-envars.html
//
// Reference for using connString:
// https://www.postgresql.org/docs/current/libpq-connect.html#LIBPQ-CONNSTRING
func (m *Migration) Setup(ctx context.Context, connString string) *pgxpool.Pool {
	if m.t == nil {
		panic("migration must be initialized with sqltest.New()")
	}

	m.t.Helper()
	// Similarly to how it's done in the application code, pgxpool is used to create a pool
	// of connections to the database that is safe to be used concurrently.
	//
	// However, it's still wise to avoid running concurrent tests.
	// In other words, please don't use t.Parallel() in your tests.
	// Also, use -p 1 whenever running tests on multiple packages with ./...
	pool, err := pgxpool.Connect(ctx, connString)
	if err != nil {
		m.t.Fatalf("cannot connect to database: %v", err)
	}

	conn, err := pool.Acquire(ctx)
	if err != nil {
		m.t.Fatalf("cannot acquire Postgres connection: %v", err)
	}
	defer conn.Release()

	var database string
	if err := conn.QueryRow(ctx, "SELECT current_database();").Scan(&database); err != nil {
		m.t.Fatalf("cannot get database name: %v", err)
	}

	// Enforce database name to start with "test_" to reduce the risk of us messing something up (example: due to environment variables).
	if !strings.HasPrefix(database, DatabasePrefix) {
		m.t.Fatalf(`refusing to run integration tests: database name is %q (%q prefix is required)`, database, DatabasePrefix)
	}

	m.migrator, err = migrate.NewMigrator(ctx, conn.Conn(), SchemaVersionTable)
	if err != nil {
		m.t.Fatalf("cannot run migration: %v", err)
	}

	m.migrator.OnStart = func(sequence int32, name, direction, sql string) {
		m.t.Logf("executing %s %s\n", name, direction)
	}

	// Test the migration scripts and prepare database for integration tests.
	m.t.Log("setup PostgreSQL database")
	if err := m.migrator.LoadMigrations(m.Options.Path); err != nil {
		m.t.Fatalf("cannot load migrations: %v", err)
	}

	// Check if the database seems to be in a reliable state.
	if !m.Options.Force {
		switch version, err := m.migrator.GetCurrentVersion(ctx); {
		case err != nil:
			m.t.Fatalf("cannot get schema version: %v", err)
		case version != 0:
			m.t.Fatalf("database is dirty, please fix %q table manually or try -force", SchemaVersionTable)
		}
	}

	// Undo database migrations.
	if err := m.migrator.MigrateTo(ctx, 0); err != nil {
		m.t.Fatalf("cannot undo database migrations: %v", err)
	}

	if !m.Options.SkipTeardown {
		m.t.Cleanup(func() {
			m.Teardown(context.Background())
		})
	}

	// Migrate to latest version of the database
	if err := m.migrator.Migrate(ctx); err != nil {
		m.t.Fatalf("cannot apply migrations: %v", err)
	}
	return pool
}

// Teardown database after running the tests.
// This function is registered by Setup to be called automatically by the testing package
// during testing cleanup.
//
// In case this is not called, you can use the Force option to reset the database.
func (m *Migration) Teardown(ctx context.Context) {
	m.t.Helper()
	m.t.Log("teardown PostgreSQL database")
	if err := m.migrator.MigrateTo(ctx, 0); err != nil {
		m.t.Fatalf("cannot tear down database migrations: %v", err)
	}
}
