//go:build integration

package services_test

import (
	"accounts/internal/domain/entities"
	"accounts/internal/services"
	"accounts/internal/storage/postgres"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	postgresPool "shared/pkg/db/postgres"
	outboxPgRepo "shared/pkg/outbox/repository/postgres"
	"shared/pkg/testutils"
	trm "shared/pkg/transactor"
	"testing"
	"time"

	trmpgx "github.com/avito-tech/go-transaction-manager/drivers/pgxv5/v2"
	"github.com/avito-tech/go-transaction-manager/trm/v2/manager"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func insertTestAccount(t *testing.T, ctx context.Context, pool *pgxpool.Pool, acc *entities.Account) {
	t.Helper()
	query := `INSERT INTO accounts(account_id, company_id, balance, currency, status, updated_at, created_at)
			  VALUES ($1, $2, $3, $4, $5, $6, $7);`
	_, err := pool.Exec(ctx, query,
		acc.AccountId(),
		acc.CompanyId(),
		acc.Balance(),
		acc.Currency().String(),
		acc.Status().String(),
		acc.UpdatedAt(),
		acc.CreatedAt(),
	)
	require.NoError(t, err, "failed to insert test account into db")
}

func getTestPostgresDSN() string {
	pgHost := os.Getenv("PG_HOST")
	if pgHost == "" {
		if os.Getenv("TEST_ENV") == "docker" {
			pgHost = "postgres-test"
		} else {
			pgHost = "127.0.0.1"
		}
	}
	pgPort := os.Getenv("PG_PORT")
	if pgPort == "" {
		if os.Getenv("TEST_ENV") == "docker" {
			pgPort = "5432"
		} else {
			pgPort = "5439"
		}
	}
	pgUser := os.Getenv("PG_USER")
	if pgUser == "" {
		pgUser = "postgres"
	}
	pgPass := os.Getenv("PG_PASSWORD")
	if pgPass == "" {
		pgPass = "postgres"
	}
	dbName := os.Getenv("ACCOUNTS_DB_NAME")
	if dbName == "" {
		dbName = "account_test"
	}

	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", pgUser, pgPass, pgHost, pgPort, dbName)
}

func setupIntegrationAccountService(t *testing.T, now ...func() time.Time) (*services.AccountService, *pgxpool.Pool) {
	t.Helper()
	if os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		t.Skip("Этап интеграционного тестирования пропущен: сбой на предыдущем этапе пирамиды тестирования")
		return nil, nil
	}
	ctx := context.Background()

	dsn := getTestPostgresDSN()
	pool, err := postgresPool.NewPool(ctx, dsn, 5*time.Second, 10*time.Second)
	require.NoError(t, err, "failed to connect to test postgres")

	// Pre-test cleanup: ensure isolated clean slate
	err = testutils.ResetTestState(ctx, pool, nil)
	require.NoError(t, err, "failed to reset test db state before test")

	t.Cleanup(func() {
		_ = testutils.ResetTestState(context.Background(), pool, nil)
		pool.Close()
	})

	getter := trmpgx.DefaultCtxGetter
	accountRepo := postgres.NewAccountRepo(pool, getter)
	accountOperationRepo := postgres.NewAccountOperationRepo(pool, getter)
	bankAccountRepo := postgres.NewBankAccountRepo(pool, getter)
	bankOperationRepo := postgres.NewBankOperationRepo(pool, getter)
	statementRepo := postgres.NewStatementRepo(pool, getter)
	outboxRepo := outboxPgRepo.NewOutboxRepo(pool, getter)

	trManager := manager.Must(trmpgx.NewDefaultFactory(pool))
	trmAdapter := trm.NewAvitoAdapter(trManager)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	svc := services.NewAccountService(
		accountRepo,
		accountOperationRepo,
		bankAccountRepo,
		bankOperationRepo,
		statementRepo,
		nil,
		outboxRepo,
		trmAdapter,
		logger,
		now...,
	)

	return svc, pool
}
