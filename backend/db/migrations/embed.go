// Package migrations embeds the canonical goose migrations so the migrate
// command ships a single binary and cannot run a different set of files than the
// one that was reviewed.
package migrations

import "embed"

// FS holds every *.sql migration at the package root.
//
//go:embed *.sql
var FS embed.FS
