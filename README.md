# pgtools
[![GoDoc](https://godoc.org/github.com/hatch-studio/pgtools?status.svg)](https://godoc.org/github.com/hatch-studio/pgtools) [![Build Status](https://github.com/hatch-studio/pgtools/workflows/Integration/badge.svg)](https://github.com/hatch-studio/pgtools/actions?query=workflow%3AIntegration) [![Coverage Status](https://coveralls.io/repos/hatch-studio/pgtools/badge.svg)](https://coveralls.io/r/hatch-studio/pgtools)

pgtools contains features [HATCH Studio](https://hatchstudio.co/) developed and rely upon to use PostgreSQL more effectively with [Go](https://golang.org/).

Please see the [official documentation](https://godoc.org/github.com/hatch-studio/pgtools) or source code for more details.

## Features
### pgtools.Wildcard
Use the Wildcard function to generate expressions for SELECT queries.

Given a table user

```sql
CREATE TABLE user {
	username text PRIMARY KEY,
	fullname text NOT NULL,
	email text NOT NULL,
	id text NOT NULL,
	Theme jsonb NOT NULL,
}
```

You might want to fill the following `User` struct with pgx (or not):

```go
type User struct {
	Username string
	FullName string
	Email    string
	Alias    string    `db:"id"`
	Theme    Theme     `db:"theme,json"`
	LastSeen time.Time `db:"-"`
}

type Theme struct {
	PrimaryColor       string
	SecondaryColor     string
	TextColor          string
	TextUppercase      bool
	FontFamilyHeadings string
	FontFamilyBody     string
	FontFamilyDefault  string
}
```

You can use

```go
sql := "SELECT " + pgtools.Wildcard(User{}) + " WHERE id = $1"
```

instead of

```go
sql := "SELECT username,full_name,email,theme WHERE id = $1"
```

### sqltest.Migration
You can use `sqltest.Migration` to write tests using Postgres implementation more effectively.

```go
func TestPostgres(t *testing.T) {
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
```

To effectively work with tests that use PostgreSQL, you'll want to run your tests with a command like:

```sh
go test -v -race -count 1 -p 1 -tags=integration ./...
```

* `-race` to pro-actively avoid race conditions
* `-count 1` to disable test caching
* `-p 1` to limit to one test execution at a time for multiple packages and avoid concurrency issues that might persist despite not using `t.Parallel()`
* `-tags=integration` or a build environment to opt-in for Postgres-related tests

If you use environment variables to connect to the database with tools like psql or tern, all you need to do is create a database for testing starting with the `test_` prefix.

We use GitHub Actions for running your integration tests with Postgres in a Continuous Integration (CI) environment.
You can find our workflow in [.github/workflows/integration.yml](.github/workflows/integration.yml).

## Acknowledgements
HATCH Studio uses the following Postgres-related software, and this work is in direct relation to them.

* [pgx](https://github.com/jackc/pgx) is a PostgreSQL driver and toolkit for Go.
* [tern](https://github.com/jackc/tern) is a standalone migration tool for PostgreSQL and part of the pgx toolkit.
* [scany](https://github.com/georgysavva/scany) is a library for scanning data from a database into Go structs.
