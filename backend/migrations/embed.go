// Package migrations embeds the SQL migration files for use by
// golang-migrate at build time.
package migrations

import "embed"

// FS contains all versioned migration files.
//
//go:embed *.sql
var FS embed.FS
