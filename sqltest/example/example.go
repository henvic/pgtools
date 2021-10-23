package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hatch-studio/pgtools/sqltest/example/internal/postgres"
	"github.com/jackc/pgx/v4/pgxpool"
)

// Database layer for your application.
type Database struct {
	// Postgres connection using the limited postgres.PGX interface,
	// instead of *pgxpool.Pool.
	// Used as a light way of disencouraging unwarranted access to low-level APIs.
	Postgres postgres.PGX
}

// Now gets the current time in the database.
func (db *Database) Now(ctx context.Context) (time.Time, error) {
	var now time.Time
	err := db.Postgres.QueryRow(ctx, "SELECT NOW();").Scan(&now)
	return now, err
}

func main() {
	pool, err := pgxpool.Connect(context.Background(), "")
	if err != nil {
		log.Fatal(err)
	}

	db := &Database{
		Postgres: pool,
	}

	now, err := db.Now(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("time from PostgreSQL: %s\n", now.Format(time.RubyDate))
}
