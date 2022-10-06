package main

import (
	"context"
	"flag"
	"log"
	"os"
	"testing"
	"time"

	"github.com/henvic/pgtools/sqltest"
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
	ctx := context.Background()
	migration := sqltest.New(t, sqltest.Options{
		Force: *force,
		Files: os.DirFS("testdata/migrations"),
	})
	conn := migration.Setup(ctx, "") // Using environment variables instead of connString to configure tests.

	db := &Database{
		Postgres: conn,
	}
	got, err := db.Now(context.Background())
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	diff := time.Since(got)
	if diff < -time.Minute || diff > time.Minute {
		t.Errorf("application and database clocks are not synced: %v", diff)
	}
}

func TestMigrationSubtests(t *testing.T) {
	ctx := context.Background()
	migration := sqltest.New(t, sqltest.Options{
		Force: *force,
		Files: os.DirFS("testdata/migrations"),
	})
	conn := migration.Setup(ctx, "") // Using environment variables instead of connString to configure tests.

	db := &Database{
		Postgres: conn,
	}

	// Run each subtest sequentially.
	// If you're modifying data on the database, you don't want to use t.Parallel() here.
	var tests = []struct {
		name string
		f    func(t *testing.T, db *Database)
	}{
		{"subfeature", testSubFeature},
		{"now", testNow},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.f(t, db)
		})
	}
}

// testSubFeature implements your integration tests.
// As the purpose of this is to document the migration process itself, it's a "no-op".
// Avoid t.Parallel() for your own sake.
func testSubFeature(t *testing.T, db *Database) {
	// Test a functionality here.
}

// testNow is another silly test to demonstrate what a test might look like.
func testNow(t *testing.T, db *Database) {
	got, err := db.Now(context.Background())
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	diff := time.Since(got)
	if diff < -time.Minute || diff > time.Minute {
		t.Errorf("application and database clocks are not synced: %v", diff)
	}
}
