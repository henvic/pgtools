// +build integration

package sqltest_test

import (
	"bytes"
	"context"
	"flag"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/hatch-studio/pgtools/sqltest"
)

// Check what happens if the migration has an invalid path.
var checkMigrationInvalidPath = flag.Bool("check_migration_invalid_path", false, "if true, TestMigrationInvalidPath should fail.")

func TestMigrationInvalidPath(t *testing.T) {
	if *checkMigrationInvalidPath {
		ctx := context.Background()
		migration := sqltest.New(t, sqltest.Options{Force: force, Path: "testdata/invalid"})
		migration.Setup(ctx, "")
		defer migration.Teardown(context.Background())
		return
	}

	args := []string{
		"-test.v",
		"-test.run=TestMigrationInvalidPath",
		"-check_migration_invalid_path",
	}
	out, err := exec.Command(os.Args[0], args...).CombinedOutput()
	if err == nil {
		t.Error("expected command to fail")
	}
	if want := []byte("cannot load migrations: open testdata/invalid: no such file or directory"); !bytes.Contains(out, want) {
		t.Errorf("got %q, wanted %q", out, want)
	}
}

// Check what happens if there is a dirty migration.
var checkMigrationDirty = flag.Bool("check_migration_dirty", false, "if true, TestMigrationDirty should fail.")

func TestMigrationDirty(t *testing.T) {
	if *checkMigrationDirty {
		ctx := context.Background()
		migration := sqltest.New(t, sqltest.Options{Force: force, Path: "testdata/migrations"})
		migration.Setup(ctx, "")
		defer migration.Teardown(context.Background())
		return
	}

	// Prepare clean environment.
	ctx := context.Background()
	migration := sqltest.New(t, sqltest.Options{Force: force, Path: "testdata/migrations"})
	conn := migration.Setup(ctx, "")
	defer migration.Teardown(context.Background())

	// Check if the migration version matches with the number of migration files.
	entries, err := ioutil.ReadDir("testdata/migrations")
	if err != nil {
		t.Errorf("cannot read migrations dir: %v", err)
	}
	var migrations int
	for _, f := range entries {
		if strings.HasSuffix(f.Name(), ".sql") {
			migrations++
		}
	}

	// Let's update the schema_version to make it dirty, and verify we are unable to run the tests.
	if _, err := conn.Exec(context.Background(), "UPDATE schema_version SET version = $1 WHERE version = $2", migrations+1, migrations); err != nil {
		t.Errorf("cannot update migration version: %q", err)
	}

	args := []string{
		"-test.v",
		"-test.run=TestMigrationDirty",
		"-check_migration_dirty",
	}
	out, err := exec.Command(os.Args[0], args...).CombinedOutput()
	if err == nil {
		t.Error("expected command to fail")
	}
	want := []byte(`database is dirty, please fix "schema_version" table manually or try -force`)
	if !bytes.Contains(out, want) {
		t.Errorf("got %q, wanted %q", out, want)
	}

	// Manually reset migration version to be able to test again to after the first 'legit' migration:
	if _, err := conn.Exec(context.Background(), "UPDATE schema_version SET version = $1", migrations); err != nil {
		t.Errorf("cannot update migration version: %q", err)
	}
}

func TestMigrationUninitialized(t *testing.T) {
	defer func() {
		want := "migration must be initialized with sqltest.New()"
		if r := recover(); r == nil || r != want {
			t.Errorf("wanted panic %q, got %v instead", want, r)
		}
	}()
	m := &sqltest.Migration{}
	m.Setup(context.Background(), "")
}
