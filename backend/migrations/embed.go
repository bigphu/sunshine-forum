// Package migrations nhúng các file SQL của goose vào binary.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
