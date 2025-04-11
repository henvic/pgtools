//go:build go1.24

package main

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/henvic/pgtools/sqltest"
	"github.com/henvic/pgtools/sqltest/example/testdata/migrations"
)

func TestQuick(t *testing.T) {
	conn := sqltest.Quick(t, os.DirFS("testdata/migrations"))

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

func TestEmbed(t *testing.T) {
	// You can almost always run your tests in parallel.
	t.Parallel()

	// If you are running your migrations on multiple tests, you can create a migrations.go file
	// exposing the migrations (see migrations.go) to make it easier to use the migrations from any test.
	conn := sqltest.Quick(t, migrations.Files)
	// Alternatively, if you use go:embed annotation
	// conn := sqltest.Quick(t, migrations.Files)

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
