package postgres_test

import (
	"accounts/migrations"
	"context"
	"os"
	postgresPool "shared/pkg/db/postgres"
	"testing"
	"time"

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
		postgres.WithDatabase("account_db"),
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

	testPool, err = postgresPool.NewPool(ctx, connStr, time.Second*5, time.Second*5)
	if err != nil {
		panic(err)
	}

	err = migrations.RunMigrations(testPool)
	if err != nil {
		panic(err)
	}

	testGetter = trmpgx.DefaultCtxGetter

	code := m.Run()

	if testPool != nil {
		testPool.Close()
	}
	_ = container.Terminate(ctx)

	os.Exit(code)
}

func truncate(t *testing.T) {
	t.Helper()
	_, err := testPool.Exec(context.Background(),
		"TRUNCATE accounts, bank_accounts, bank_operations, account_operations, outbox, statements CASCADE")
	require.NoError(t, err)
}
