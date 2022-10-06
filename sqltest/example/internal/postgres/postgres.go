package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PGX limited interface with high-level API for pgx methods safe to be used in high-level business logic packages.
// It is satisfied by implementations *pgx.Conn and *pgxpool.Pool (and you should probably use the second one usually).
//
// Caveat: It doesn't expose a method to acquire a *pgx.Conn or handle notifications,
// so it's not compatible with LISTEN/NOTIFY.
//
// Reference: https://pkg.go.dev/github.com/jackc/pgx/v5
type PGX interface {
	// Begin starts a transaction. Unlike database/sql, the context only affects the begin command. i.e. there is no
	// auto-rollback on context cancellation.
	Begin(ctx context.Context) (pgx.Tx, error)

	// BeginTx starts a transaction with txOptions determining the transaction mode. Unlike database/sql, the context only
	// affects the begin command. i.e. there is no auto-rollback on context cancellation.
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)

	// CopyFrom uses the PostgreSQL copy protocol to perform bulk data insertion.
	// It returns the number of rows copied and an error.
	//
	// CopyFrom requires all values use the binary format. Almost all types
	// implemented by pgx use the binary format by default. Types implementing
	// Encoder can only be used if they encode to the binary format.
	CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error)

	// Exec executes sql. sql can be either a prepared statement name or an SQL string. arguments should be referenced
	// positionally from the sql string as $1, $2, etc.
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)

	// Query sends a query to the server and returns a Rows to read the results. Only errors encountered sending the query
	// and initializing Rows will be returned. Err() on the returned Rows must be checked after the Rows is closed to
	// determine if the query executed successfully.
	//
	// The returned Rows must be closed before the connection can be used again. It is safe to attempt to read from the
	// returned Rows even if an error is returned. The error will be the available in rows.Err() after rows are closed. It
	// is allowed to ignore the error returned from Query and handle it in Rows.
	//
	// It is possible for a query to return one or more rows before encountering an error. In most cases the rows should be
	// collected before processing rather than processed while receiving each row. This avoids the possibility of the
	// application processing rows from a query that the server rejected. The CollectRows function is useful here.
	//
	// An implementor of QueryRewriter may be passed as the first element of args. It can rewrite the sql and change or
	// replace args. For example, NamedArgs is QueryRewriter that implements named arguments.
	//
	// For extra control over how the query is executed, the types QueryExecMode, QueryResultFormats, and
	// QueryResultFormatsByOID may be used as the first args to control exactly how the query is executed. This is rarely
	// needed. See the documentation for those types for details.
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)

	// QueryRow is a convenience wrapper over Query. Any error that occurs while
	// querying is deferred until calling Scan on the returned Row. That Row will
	// error with ErrNoRows if no rows are returned.
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row

	// SendBatch sends all queued queries to the server at once. All queries are run in an implicit transaction unless
	// explicit transaction control statements are executed. The returned BatchResults must be closed before the connection
	// is used again.
	SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults
}

// Validate if the PGX interface was derived from *pgx.Conn and *pgxpool.Pool correctly.
var (
	_ PGX = (*pgx.Conn)(nil)
	_ PGX = (*pgxpool.Pool)(nil)
)
