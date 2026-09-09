package postgres_test

import (
	"auth/migrations"
	"context"
	"os"
	"testing"
	"time"

	postgresPool "shared/pkg/db/postgres"

	trmpgx "github.com/avito-tech/go-transaction-manager/drivers/pgxv5/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	testPool   *pgxpool.Pool
	testGetter *trmpgx.CtxGetter
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	container, err := postgres.Run(ctx,
		"postgres:16",
		postgres.WithDatabase("auth_db"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		panic(err)
	}
	defer func() { _ = container.Terminate(ctx) }()

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		panic(err)
	}

	testPool, err = postgresPool.NewPool(ctx, connStr, time.Second, time.Second)
	if err != nil {
		panic(err)
	}

	err = migrations.RunMigrations(testPool)
	if err != nil {
		panic(err)
	}

	testGetter = trmpgx.DefaultCtxGetter

	os.Exit(m.Run())
}

func truncate(t *testing.T) {
	t.Helper()
	_, err := testPool.Exec(context.Background(),
		"TRUNCATE users, companies CASCADE")
	require.NoError(t, err)
}
