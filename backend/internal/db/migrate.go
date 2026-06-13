// Package db provides the database connection pool, migration runner, and
// sqlc-generated query code.
package db

import (
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	// Registers the postgres migration driver.
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"github.com/isAdamBailey/massa/backend/migrations"
)

// Migrate applies all pending up migrations to the database at dsn.
func Migrate(dsn string) error {
	source, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return fmt.Errorf("load migrations: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", source, dsn)
	if err != nil {
		return fmt.Errorf("init migrator: %w", err)
	}
	defer func() { _, _ = m.Close() }()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("run migrations: %w", err)
	}

	return nil
}
