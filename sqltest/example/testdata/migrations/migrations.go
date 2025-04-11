package migrations

import "embed"

// Files contains the SQL migration files in this directory.
// See TestEmbed in example/example_test.go for usage.
//
//go:embed *.sql
var Files embed.FS
