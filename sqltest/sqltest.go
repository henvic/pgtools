// Package sqltest makes it easy to write tests using pgx and tern.
package sqltest

import (
	"context"
	"fmt"
	"io/fs"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/tern/v2/migrate"
)

var (
	// DatabasePrefix defines a prefix for the database name.
	// It is used to mitigate the risk of running migration and tests on the wrong database.
	DatabasePrefix = "test"

	// SchemaVersionTable where tern saves the version of the current migration in PostgreSQL.
	SchemaVersionTable = "schema_version"

	// Empty can be used for when you want a temporary database for your tests but don't need to run migrations.
	Empty fs.FS = emptyFS{}
)

// emptyFS is a special fs.FS implementation used to symbolize empty migrations.
type emptyFS struct{}

// Open implements the fs.FS interface but always returns an error, as no files exist.
func (e emptyFS) Open(name string) (fs.File, error) {
	return nil, fs.ErrNotExist
}

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

	// UseExisting database from connection instead of creating a temporary one.
	// If set, the database isn't dropped during teardown / test cleanup.
	UseExisting bool

	// TemporaryDatabasePrefix for namespacing the temporary database name created for the test function.
	// Useful if you're running multiple tests in parallel to avoid flaky tests due to naming clashes.
	// Ignore if using UseExisting.
	TemporaryDatabasePrefix string

	// Files to use in the migration.
	// Implement embed.FS or use os.DirFS to load the migration files.
	// e.g., os.DirFS("migrations/")
	Files fs.FS

	// Logs enables printing status of the migration step-by-step.
	Logs bool
}

// Migration simplifies avlidadting the migration process, and setting up a test database
// for executing your PostgreSQL-based tests on.
type Migration struct {
	Options Options

	t        testing.TB
	migrator *migrate.Migrator

	pool     *pgxpool.Pool
	conn     *pgx.Conn
	database string
}

// Setup the migration.
// This function returns a pgx pool that can be used to connect to the database.
// If something fails, t.Fatal is called.
//
// It register the Teardown function with testing.TB to clean up the database once the
// tests are over by default, but this can be disabled by setting the SkipTeardown option.
//
// If the UseExisting option is set, a temporary database is used for running the tests.
//
// If you're using PostgreSQL environment variables, you should pass an empty string as the
// connection string, as in:
//
//	pool := m.Setup(context.Background(), "")
//
// Reference for configuring the PostgreSQL client with environment variables:
// https://www.postgresql.org/docs/current/libpq-envars.html
//
// Reference for using connString:
// https://www.postgresql.org/docs/current/libpq-connect.html#LIBPQ-CONNSTRING
func (m *Migration) Setup(ctx context.Context, connString string) *pgxpool.Pool {
	return m.setupVersion(ctx, connString, nil)
}

// SetupVersion of the migrations is similar to the Setup version,
// but migrates to the given target version.
func (m *Migration) SetupVersion(ctx context.Context, connString string, targetVersion int32) *pgxpool.Pool {
	return m.setupVersion(ctx, connString, &targetVersion)
}

// setupVersion is only used to avoid receiving targetVersion as a pointer in the exported function.
// If targetVersion isn't passed, it migrates to the latest migration, which is only known after
// migrate.NewMigrator is called.
func (m *Migration) setupVersion(ctx context.Context, connString string, targetVersion *int32) *pgxpool.Pool {
	if m.t == nil {
		panic("migration must be initialized with sqltest.New()")
	}

	m.t.Helper()
	if m.Options.Logs {
		m.t.Log("setup PostgreSQL database")
	}

	// Similarly to how it's done in the application code, pgxpool is used to create a pool
	// of connections to the database that is safe to be used concurrently.
	poolConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		m.t.Fatal(err)
	}

	if !m.Options.UseExisting {
		var err error
		if m.conn, err = pgx.Connect(ctx, connString); err != nil {
			m.t.Fatal(err)
		}
		m.database = m.Options.TemporaryDatabasePrefix + SQLTestName(m.t)
		// Lousy check if database name is invalid.
		// Ref: https://www.postgresql.org/docs/current/sql-syntax-lexical.html#SQL-SYNTAX-IDENTIFIERS
		if strings.ContainsAny(m.database, `" `) {
			m.t.Fatalf("invalid database name")
		}

		if err := m.cleanDB(ctx, connString); err != nil {
			m.t.Fatalf("cannot create database: %v", err)
		}

		poolConfig.ConnConfig.Database = m.database
	}
	m.pool, err = pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		m.t.Fatalf("cannot connect to database: %v", err)
	}

	poolConn, err := m.pool.Acquire(ctx)
	if err != nil {
		m.t.Fatalf("cannot acquire PostgreSQL connection: %v", err)
	}
	defer poolConn.Release()

	if err := poolConn.QueryRow(ctx, "SELECT current_database();").Scan(&m.database); err != nil {
		m.t.Fatalf("cannot get database name: %v", err)
	}

	// Enforce database name to start with "test" to mitigate risk of modifying wrong database by mistake.
	if !strings.HasPrefix(m.database, DatabasePrefix) {
		m.t.Fatalf(`refusing to run integration tests: database name is %q (%q prefix is required)`, m.database, DatabasePrefix)
	}

	if !m.Options.SkipTeardown {
		m.t.Cleanup(func() {
			m.Teardown(context.Background())
		})
	}
	if m.Options.Files != Empty {
		if err := m.migrate(ctx, poolConn, targetVersion); err != nil {
			m.t.Fatal(err)
		}
	}
	return m.pool
}

// migrate database using tern.
func (m *Migration) migrate(ctx context.Context, poolConn *pgxpool.Conn, targetVersion *int32) (err error) {
	m.migrator, err = migrate.NewMigrator(ctx, poolConn.Conn(), SchemaVersionTable)
	if err != nil {
		return fmt.Errorf("cannot run migration: %w", err)
	}

	if m.Options.Logs {
		m.migrator.OnStart = func(sequence int32, name, direction, sql string) {
			m.t.Logf("executing %s %s", name, direction)
		}
	}
	}

	// Test the migration scripts and prepare database for integration tests.
	if err := m.migrator.LoadMigrations(m.Options.Files); err != nil {
		return fmt.Errorf("cannot load migrations: %w", err)
	}

	// Check if the database seems to be in a reliable state.
	// If the database current version is ahead of existing migrations, refuse to overwrite it.
	if !m.Options.Force {
		switch version, err := m.migrator.GetCurrentVersion(ctx); {
		case err != nil:
			return fmt.Errorf("cannot get schema version: %w", err)
		case int(version) > len(m.migrator.Migrations):
			return fmt.Errorf("database is dirty (current version is ahead of existing migrations), please fix %q table manually or try -force", SchemaVersionTable)
		}
	}

	// Undo database migrations.
	if err := m.migrator.MigrateTo(ctx, 0); err != nil {
		return fmt.Errorf("cannot undo database migrations: %v", err)
	}

	// Migrate to the latest or target version of the database.
	tv := int32(len(m.migrator.Migrations))
	if targetVersion != nil {
		tv = *targetVersion
	}
	if err := m.migrator.MigrateTo(ctx, tv); err != nil {
		return fmt.Errorf("cannot apply migrations: %v", err)
	}
	return nil
}

// MigrateTo migrates to targetVersion.
//
// You probably only need this if you need to test code against an older version of your database,
// or if you are testing a migration process.
func (m *Migration) MigrateTo(ctx context.Context, targetVersion int32) {
	m.t.Helper()
	if err := m.migrator.MigrateTo(ctx, targetVersion); err != nil {
		m.t.Fatalf("cannot migrate database to version %d: %v", targetVersion, err)
	}
}

// Teardown database after running the tests.
//
// This function is registered by Setup to be called automatically by the testing package
// during testing cleanup. Use the SkipTeardown option to disable this.
func (m *Migration) Teardown(ctx context.Context) {
	m.t.Helper()
	if m.Options.Logs {
		m.t.Log("teardown PostgreSQL database")
	}
	m.pool.Close()

	if !m.Options.UseExisting {
		defer m.conn.Close(ctx)
		if err := m.dropDB(ctx); err != nil {
			m.t.Fatalf("cannot drop database: %v", err)
		}
	}
}

// cleanDB creates a temporary database when CleanDB is used.
func (m *Migration) cleanDB(ctx context.Context, connString string) error {
	// If force is set to true, drop database if it exists.
	if m.Options.Force {
		if err := m.dropDB(ctx); err != nil {
			return err
		}
	}

	// Create new database.
	_, err := m.conn.Exec(ctx, fmt.Sprintf(`CREATE DATABASE "%s";`, m.database))
	return err
}

// dropDB drops the created temporary database.
func (m *Migration) dropDB(ctx context.Context) error {
	_, err := m.conn.Exec(ctx, fmt.Sprintf(`DROP DATABASE IF EXISTS "%s";`, m.database))
	return err
}

// SQLTestName normalizes a test name to a database name.
// It lowercases the test name and converts / to underscore.
func SQLTestName(t testing.TB) string {
	return strings.ToLower(strings.ReplaceAll(t.Name(), "/", "_"))
}
