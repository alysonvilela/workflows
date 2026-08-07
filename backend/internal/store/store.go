// Package store owns the Postgres connection pool and applies migrations
// at boot. Discovery Run and Node coordination lives entirely in SQL
// (ADR-0003) — this package is where that SQL is issued from.
package store

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/discovery/backend/migrations"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pressly/goose/v3"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"

	_ "github.com/jackc/pgx/v5/stdlib" // database/sql driver for goose
)

type Store struct {
	Pool *pgxpool.Pool

	// River is the job-insertion side of the River client (ticket 16):
	// CreateDocumentIfNew uses it to enqueue a process_document job in the
	// same transaction that creates the Document row. Set by the caller
	// after New returns — main.go and the test helpers construct the River
	// client with this Store's Pool, so the field can't be filled in New
	// itself without a chicken-and-egg dependency on the client's own
	// Workers bundle.
	River *river.Client[pgx.Tx]
}

func New(ctx context.Context, databaseURL string) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("store: connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("store: ping: %w", err)
	}
	return &Store{Pool: pool}, nil
}

// Migrate applies every pending goose migration embedded in the binary.
func (s *Store) Migrate(databaseURL string) error {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return fmt.Errorf("store: open for migrate: %w", err)
	}
	defer db.Close()

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("store: goose dialect: %w", err)
	}
	if err := goose.Up(db, "."); err != nil {
		return fmt.Errorf("store: migrate: %w", err)
	}
	return nil
}

// MigrateRiver applies River's own schema migrations (the job queue
// tables) via rivermigrate — separate from Migrate above because River
// ships its migrations in its own format, not goose's, but both run
// against the same Postgres database at boot.
func (s *Store) MigrateRiver(ctx context.Context) error {
	migrator, err := rivermigrate.New(riverpgxv5.New(s.Pool), nil)
	if err != nil {
		return fmt.Errorf("store: river migrator: %w", err)
	}
	if _, err := migrator.Migrate(ctx, rivermigrate.DirectionUp, nil); err != nil {
		return fmt.Errorf("store: migrate river schema: %w", err)
	}
	return nil
}

func (s *Store) Ping(ctx context.Context) error {
	return s.Pool.Ping(ctx)
}

func (s *Store) Close() {
	s.Pool.Close()
}
