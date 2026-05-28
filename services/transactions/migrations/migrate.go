package migrations

import (
	"embed"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib" // драйвер pgx
	"github.com/pressly/goose/v3"
)

//go:embed *.sql
var embedMigrations embed.FS

func RunMigrations(pool *pgxpool.Pool) error {
	goose.SetBaseFS(embedMigrations)

	db, err := goose.OpenDBWithDriver("pgx", pool.Config().ConnConfig.ConnString())
	if err != nil {
		return fmt.Errorf("failed to open db for migrations: %w", err)
	}
	defer func() { _ = db.Close() }()

	return goose.Up(db, ".")
}
