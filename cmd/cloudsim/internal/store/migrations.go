package store

import (
	"embed"

	"github.com/smart-core-os/sc-bos/internal/sqlite"
)

// Migrations contains the SQL migrations for the store schema.
//
//go:embed migrations/*.sql
var migrationsFS embed.FS

var schema = sqlite.MustLoadVersionedSchema(migrationsFS, "migrations")
