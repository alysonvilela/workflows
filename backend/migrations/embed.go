// Package migrations embeds the goose SQL migrations so the binary can
// apply them at boot without shelling out to the goose CLI.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
