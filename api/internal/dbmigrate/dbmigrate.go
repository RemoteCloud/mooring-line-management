// Package dbmigrate runs embedded golang-migrate migrations against Postgres.
package dbmigrate

import (
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5" // registers pgx5:// driver
	"github.com/golang-migrate/migrate/v4/source/iofs"

	appdb "github.com/ncl/mooring-api/db"
)

// Up applies all pending migrations. No-op if already current.
func Up(databaseURL string) error {
	m, err := newMigrate(databaseURL)
	if err != nil {
		return err
	}
	defer m.Close()
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate up: %w", err)
	}
	return nil
}

// Down rolls back all migrations. Used in tests / teardown.
func Down(databaseURL string) error {
	m, err := newMigrate(databaseURL)
	if err != nil {
		return err
	}
	defer m.Close()
	if err := m.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate down: %w", err)
	}
	return nil
}

func newMigrate(databaseURL string) (*migrate.Migrate, error) {
	src, err := iofs.New(appdb.Migrations, "migrations")
	if err != nil {
		return nil, fmt.Errorf("open embedded migrations: %w", err)
	}
	// golang-migrate's pgx/v5 driver expects the pgx5:// scheme.
	url := "pgx5" + trimScheme(databaseURL)
	m, err := migrate.NewWithSourceInstance("iofs", src, url)
	if err != nil {
		return nil, fmt.Errorf("init migrate: %w", err)
	}
	return m, nil
}

// trimScheme strips a leading postgres:// or postgresql:// so we can re-prefix with pgx5.
func trimScheme(url string) string {
	for _, p := range []string{"postgresql", "postgres"} {
		if len(url) > len(p) && url[:len(p)] == p {
			return url[len(p):]
		}
	}
	return url
}
