package sqltest_test

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/henvic/pgtools/sqltest"
	"github.com/jackc/pgx/v5"
)

func TestMain(m *testing.M) {
	if os.Getenv("INTEGRATION_TESTDB") != "true" {
		log.Printf("Skipping tests that require database connection")
		return
	}
	os.Exit(m.Run())
}

var force = flag.Bool("force", false, "Force cleaning the database before starting")

func TestNow(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	migration := sqltest.New(t, sqltest.Options{
		Force: *force,
		Files: os.DirFS("example/testdata/migrations"),

		// If we don't use a prefix, the test will be flaky when testing multi packages
		// because there is another TestNow function in example/example_test.go.
		TemporaryDatabasePrefix: "test_internal_",
	})
	conn := migration.Setup(ctx, "") // Using environment variables instead of connString to configure tests.
	var tt time.Time
	if err := conn.QueryRow(ctx, "SELECT NOW();").Scan(&tt); err != nil {
		t.Errorf("cannot execute query: %v", err)
	}
	if tt.IsZero() {
		t.Error("time returned by pgx is zero")
	}
}

func TestSetupVersion(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	migration := sqltest.New(t, sqltest.Options{
		Force: *force,
		Files: os.DirFS("example/testdata/migrations"),
	})
	// Similar to the previous test, but we want to migrate to a specific version (2).
	conn := migration.SetupVersion(ctx, "", 2) // Target version 2
	var version int32
	if err := conn.QueryRow(ctx, "SELECT version FROM schema_version").Scan(&version); err != nil {
		t.Errorf("cannot query schema version: %v", err)
	}
	if version != 2 {
		t.Errorf("got version %d, wanted %d", version, 2)
	}
}

func TestPrefixedDatabase(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	migration := sqltest.New(t, sqltest.Options{
		Force:                   *force,
		Files:                   os.DirFS("example/testdata/migrations"),
		TemporaryDatabasePrefix: "test_must_have_prefix_",
	})
	conn := migration.Setup(ctx, "") // Using environment variables instead of connString to configure tests.
	var got string
	if err := conn.QueryRow(ctx, "SELECT current_database();").Scan(&got); err != nil {
		t.Errorf("cannot get database name: %v", err)
	}
	if want := "test_must_have_prefix_testprefixeddatabase"; want != got {
		t.Errorf("got %q, wanted %q", got, want)
	}
}

var checkMigrationInvalidPath = flag.Bool("check_migration_invalid_path", false, "if true, TestMigrationInvalidPath should fail.")

func TestMigrationEmpty(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	migration := sqltest.New(t, sqltest.Options{
		Force: true,
		Files: sqltest.Empty,
	})
	pool := migration.Setup(ctx, "")
	var tt time.Time
	if err := pool.QueryRow(ctx, "SELECT NOW();").Scan(&tt); err != nil {
		t.Errorf("cannot execute query: %v", err)
	}
	if tt.IsZero() {
		t.Error("time returned by pgx is zero")
	}
}

func TestMigrationInvalidPath(t *testing.T) {
	if *checkMigrationInvalidPath {
		ctx := context.Background()
		migration := sqltest.New(t, sqltest.Options{
			Force:       *force,
			Files:       os.DirFS("testdata/invalid"),
			UseExisting: true,
		})
		migration.Setup(ctx, "")
		return
	}

	args := []string{
		"-test.v",
		"-test.run=TestMigrationInvalidPath",
		"-check_migration_invalid_path",
	}
	if *force {
		args = append(args, "-force")
	}
	out, err := exec.Command(os.Args[0], args...).CombinedOutput()
	if err == nil {
		t.Error("expected command to fail")
	}
	if want := []byte("no such file or directory"); !bytes.Contains(out, want) {
		t.Errorf("got %q, wanted %q", out, want)
	}
}

// Check what happens if there is a dirty migration.
var checkMigrationDirty = flag.Bool("check_migration_dirty", false, "if true, TestMigrationDirty should fail.")

func TestMigrationDirty(t *testing.T) {
	if *checkMigrationDirty {
		ctx := context.Background()
		migration := sqltest.New(t, sqltest.Options{
			Files:       os.DirFS("example/testdata/migrations"),
			UseExisting: true,
		})
		migration.Setup(ctx, "")
		return
	}

	// Prepare clean environment.
	ctx := context.Background()
	migration := sqltest.New(t, sqltest.Options{
		Force: *force,
		Files: os.DirFS("example/testdata/migrations"),
	})
	conn := migration.Setup(ctx, "")

	// Check if the migration version matches with the number of migration files.
	entries, err := os.ReadDir("example/testdata/migrations")
	if err != nil {
		t.Errorf("cannot read migrations dir: %v", err)
	}
	var migrations int
	for _, f := range entries {
		if strings.HasSuffix(f.Name(), ".sql") {
			migrations++
		}
	}
	// Let's update the schema_version to make it ahead of the current version,
	// and verify we are unable to run the tests.
	if _, err := conn.Exec(context.Background(), "UPDATE schema_version SET version = $1 WHERE version = $2", migrations+1, migrations); err != nil {
		t.Errorf("cannot update migration version: %q", err)
	}

	args := []string{
		"-test.v",
		"-test.run=TestMigrationDirty",
		"-check_migration_dirty",
	}
	if *force {
		args = append(args, "-force")
	}
	cmd := exec.Command(os.Args[0], args...)
	cmd.Env = append(os.Environ(), "PGDATABASE="+sqltest.SQLTestName(t))
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Error("expected command to fail")
	}
	want := []byte(`database is dirty (current version is ahead of existing migrations), please fix "schema_version" table manually or try -force`)
	if !bytes.Contains(out, want) {
		t.Errorf("got %q, wanted %q", out, want)
	}

	// Manually reset migration version to be able to test again to after the first 'legit' migration:
	if _, err := conn.Exec(context.Background(), "UPDATE schema_version SET version = $1", migrations); err != nil {
		t.Errorf("cannot update migration version: %q", err)
	}
}

var checkExistingTemporaryDB = flag.Bool("check_existing_temporary_db", false, "if true, ExistingTemporaryDB should fail.")

func TestExistingTemporaryDB(t *testing.T) {
	t.Parallel()
	if *checkExistingTemporaryDB {
		ctx := context.Background()
		migration := sqltest.New(t, sqltest.Options{
			Files: os.DirFS("example/testdata/migrations"),
		})
		migration.Setup(ctx, "")
		return
	}

	// Prepare clean environment.
	ctx := context.Background()
	conn, err := pgx.Connect(context.Background(), "")
	if err != nil {
		t.Fatalf("connection error: %v", err)
	}

	testDB := sqltest.SQLTestName(t)
	_, err = conn.Exec(ctx, fmt.Sprintf(`CREATE DATABASE "%s";`, testDB))
	if err != nil {
		t.Fatalf("cannot create database: %v", err)
	}
	defer func() {
		conn.Exec(ctx, fmt.Sprintf(`DROP DATABASE IF EXISTS "%s";`, testDB))
	}()

	args := []string{
		"-test.v",
		"-test.run=TestExistingTemporaryDB",
		"-check_existing_temporary_db",
	}
	if *force {
		args = append(args, "-force")
	}
	out, err := exec.Command(os.Args[0], args...).CombinedOutput()
	if err == nil {
		t.Error("expected command to fail")
	}
	want := []byte(`cannot create database: ERROR: database "testexistingtemporarydb" already exists`)
	if !bytes.Contains(out, want) {
		t.Errorf("got %q, wanted %q", out, want)
	}
}

func TestMigrationUninitialized(t *testing.T) {
	t.Parallel()
	defer func() {
		want := "migration must be initialized with sqltest.New()"
		if r := recover(); r == nil || r != want {
			t.Errorf("wanted panic %q, got %v instead", want, r)
		}
	}()
	m := &sqltest.Migration{}
	m.Setup(context.Background(), "")
}

func TestSQLTestName(t *testing.T) {
	t.Parallel()
	var want = []string{
		"testsqltestname_foo",
		"testsqltestname_foo_bar",
	}
	var got []string
	t.Run("foo", func(t *testing.T) {
		got = append(got, sqltest.SQLTestName(t))
		t.Run("bar", func(t *testing.T) {
			got = append(got, sqltest.SQLTestName(t))
		})
	})
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}
func TestMigrationLogs(t *testing.T) {
	t.Parallel()

	var (
		logs []string
		ran  bool
	)

	t.Run("internal", func(t *testing.T) {
		ran = true
		mocked := &mockLogger{
			T:    t,
			logs: &logs,
		}

		migration := sqltest.New(mocked, sqltest.Options{
			Force: *force,
			Files: os.DirFS("example/testdata/migrations"),
			Logs:  true,
		})
		migration.Setup(context.Background(), "")
	})
	if !ran {
		t.Skip("internal test didn't run")
	}
	expectedLogs := []string{
		"setup PostgreSQL database",
		"executing 001_media.sql up",
		"executing 002_settings.sql up",
		"executing 003_posts.sql up",
		"teardown PostgreSQL database",
	}
	if len(logs) != len(expectedLogs) {
		t.Errorf("expected %d log lines, but got %d", len(expectedLogs), len(logs))
	}
	for i, expected := range expectedLogs {
		if !strings.Contains(logs[i], expected) {
			t.Errorf("log line %d: expected to contain %q, but got %q", i+1, expected, logs[i])
		}
	}
}

type mockLogger struct {
	*testing.T
	logs *[]string
}

func (m *mockLogger) Log(args ...interface{}) {
	*m.logs = append(*m.logs, fmt.Sprint(args...))
}

func (m *mockLogger) Logf(format string, args ...interface{}) {
	*m.logs = append(*m.logs, fmt.Sprintf(format, args...))
}
