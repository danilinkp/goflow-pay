package db

import (
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib" // драйвер "pgx"
	"github.com/pressly/goose/v3"
)

func RunMigrations(pool *pgxpool.Pool) error {
	db, err := goose.OpenDBWithDriver("pgx", pool.Config().ConnConfig.ConnString())
	if err != nil {
		return fmt.Errorf("failed to open db for migrations: %w", err)
	}
	defer db.Close()

	return goose.Up(db, "../../../migrations")
}
