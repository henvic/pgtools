# pgtools
[![GoDoc](https://godoc.org/github.com/hatch-studio/pgtools?status.svg)](https://godoc.org/github.com/hatch-studio/pgtools) [![Build Status](https://github.com/hatch-studio/pgtools/workflows/Tests/badge.svg)](https://github.com/hatch-studio/pgtools/actions?query=workflow%3ATests) [![Coverage Status](https://coveralls.io/repos/hatch-studio/pgtools/badge.svg)](https://coveralls.io/r/hatch-studio/pgtools)

pgtools contains features [HATCH Studio](https://hatchstudio.co/) developed and rely upon to use PostgreSQL more effectively with [Go](https://golang.org/).

Please see the [official documentation](https://godoc.org/github.com/hatch-studio/pgtools) or source code for more details.

## Features
### pgtools.Wildcard
Use the Wildcard function to generate expressions for SELECT queries.

Given a table `user`:

```sql
CREATE TABLE user {
	username text PRIMARY KEY,
	fullname text NOT NULL,
	email text NOT NULL,
	id text NOT NULL,
	Theme jsonb NOT NULL,
}
```

You might want to create a struct to map it like the following for use with [scany](https://github.com/georgysavva/scany).

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

The db struct tag follows the same pattern of other SQL libraries, besides scany.

* A field without a db tag is mapped to its equivalent form in `snake_case` instead of `CamelCase`.
* Fields with `db:"-"` are ignored and no mapping is done for them.
* A field with `db:"name"` maps that field to the name SQL column.
* A field with `db:",json"` or `db:"something,json"` maps to a [JSON datatype](https://www.postgresql.org/docs/current/datatype-json.html) column named _something_.

Therefore, you can use:

```go
sql := "SELECT " + pgtools.Wildcard(User{}) + " WHERE id = $1"
```

instead of

```go
sql := "SELECT username,full_name,email,theme WHERE id = $1"
```

This works better than using `SELECT *` for the following reasons:

* Performance: you only query data that your struct can map.
* Correctness: no mismatch.
* If you add a new field in a struct, you don't need to change your queries.
* scany fails when reading unmapped columns with `SELECT *`, but this solves it.
* If you delete a field, you don't need to change your queries.

#### Limitations
Using `pgtools.Wildcard()` on a JOIN is tricky, and not generally recommended – at least for now.

To see why, take the following example:

```go
sql := `SELECT ` + postgres.Wildcard(Entity{}) + `
	FROM entity
	LEFT JOIN
sister_entity on sister_entity.entity_id = entity.id`
```

This will be roughly translated to:

```sql
SELECT id, name, ...
```

Which is not guaranteed to be correct due to ambiguity.
What we want is to have the following instead:

```sql
SELECT table.field1, table.field2...
```

In this case, we want to write everything manually so that PostgreSQL doesn't try to fetch each field in each joined table, as this might lead to conflicts, extra data, bugs, or eventually an error.

For now, it's better to avoid using `pgtools.Wildcard()` for JOINs altogether, even when it seems to work fine.

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

The path indicates where your SQL migration files created for use with [tern](https://github.com/jackc/tern) live.

Example of tern migration file `003_posts.sql`:

```sql
CREATE TABLE posts (
	id text PRIMARY KEY,
	name text NOT NULL,
	message text NOT NULL,
	created_at timestamp with time zone NOT NULL DEFAULT now(),
	modified_at timestamp with time zone NOT NULL DEFAULT now()
);

---- create above / drop below ----
DROP TABLE IF EXISTS posts;
```

To effectively work with tests that use PostgreSQL, you'll want to run your tests with a command like:

```sh
go test -v -race -count 1 -p 1 -tags=integration ./...
```

* `-race` to pro-actively avoid race conditions
* `-count 1` to disable test caching
* `-p 1` to limit to one test execution at a time for multiple packages and avoid concurrency issues that might persist despite not using `t.Parallel()`
* `-tags=integration` or a build environment to opt-in for Postgres-related tests

If you use environment variables to connect to the database with tools like psql or tern, you're already good to go once you create a database for testing starting with the prefix `test_`.

We use GitHub Actions for running your integration tests with Postgres in a Continuous Integration (CI) environment.
You can find our workflow in [.github/workflows/integration.yml](.github/workflows/integration.yml).

## Acknowledgements
HATCH Studio uses the following Postgres-related software, and this work is in direct relation to them.

* [pgx](https://github.com/jackc/pgx) is a PostgreSQL driver and toolkit for Go.
* [tern](https://github.com/jackc/tern) is a standalone migration tool for PostgreSQL and part of the pgx toolkit.
* [scany](https://github.com/georgysavva/scany) is a library for scanning data from a database into Go structs.
