// Package db embeds SQL migrations so the single binary carries its own schema —
// important for the onboard deployment dropped onto a vessel.
package db

import "embed"

//go:embed migrations/*.sql
var Migrations embed.FS
