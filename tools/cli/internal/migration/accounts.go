package migration

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func MigrateAccounts(ctx context.Context, postgresDSN, mongoDSN, mongoDBName string, batchSize int) error {
	pg, err := pgxpool.New(ctx, postgresDSN)
	if err != nil {
		return fmt.Errorf("postgres connect: %w", err)
	}
	defer pg.Close()

	mongoClient, err := mongo.Connect(options.Client().ApplyURI(mongoDSN))
	if err != nil {
		return fmt.Errorf("mongo connect: %w", err)
	}
	defer mongoClient.Disconnect(ctx)

	db := mongoClient.Database(mongoDBName)

	if err = migrateAccountsTable(ctx, pg, db, batchSize); err != nil {
		return err
	}
	if err = migrateBankAccountsTable(ctx, pg, db, batchSize); err != nil {
		return err
	}
	if err = migrateAccountOperationsTable(ctx, pg, db, batchSize); err != nil {
		return err
	}
	if err = migrateBankOperationsTable(ctx, pg, db, batchSize); err != nil {
		return err
	}
	if err = migrateStatementsTable(ctx, pg, db, batchSize); err != nil {
		return err
	}

	return nil
}

func migrateAccountsTable(ctx context.Context, pg *pgxpool.Pool, db *mongo.Database, batchSize int) error {
	fmt.Println("  accounts...")

	tx, err := pg.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, "DECLARE cur CURSOR FOR SELECT account_id, company_id, balance, currency, status, created_at, updated_at FROM accounts")
	if err != nil {
		return fmt.Errorf("declare cursor: %w", err)
	}

	collection := db.Collection("accounts")
	total := 0

	for {
		rows, err := tx.Query(ctx, fmt.Sprintf("FETCH %d FROM cur", batchSize))
		if err != nil {
			return fmt.Errorf("fetch: %w", err)
		}

		var batch []interface{}
		for rows.Next() {
			var (
				accountId uuid.UUID
				companyId uuid.UUID
				balance   int64
				currency  string
				status    string
				createdAt time.Time
				updatedAt time.Time
			)
			if err = rows.Scan(&accountId, &companyId, &balance, &currency, &status, &createdAt, &updatedAt); err != nil {
				rows.Close()
				return fmt.Errorf("scan: %w", err)
			}
			batch = append(batch, bson.M{
				"_id":        accountId,
				"company_id": companyId,
				"balance":    balance,
				"currency":   currency,
				"status":     status,
				"created_at": createdAt,
				"updated_at": updatedAt,
			})
		}
		rows.Close()

		if len(batch) == 0 {
			break
		}

		if err = flushBatch(ctx, collection, batch); err != nil {
			return err
		}
		total += len(batch)
		fmt.Printf("    inserted %d docs\n", total)
	}

	fmt.Printf("    total: %d\n", total)
	return nil
}

func migrateBankAccountsTable(ctx context.Context, pg *pgxpool.Pool, db *mongo.Database, batchSize int) error {
	fmt.Println("  bank_accounts...")

	tx, err := pg.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, "DECLARE cur CURSOR FOR SELECT bank_account_id, company_id, name, bic, settlement_account, currency, created_at, updated_at FROM bank_accounts")
	if err != nil {
		return fmt.Errorf("declare cursor: %w", err)
	}

	collection := db.Collection("bank_accounts")
	total := 0

	for {
		rows, err := tx.Query(ctx, fmt.Sprintf("FETCH %d FROM cur", batchSize))
		if err != nil {
			return fmt.Errorf("fetch: %w", err)
		}

		var batch []interface{}
		for rows.Next() {
			var (
				bankAccountId     uuid.UUID
				companyId         uuid.UUID
				name              string
				bic               string
				settlementAccount string
				currency          string
				createdAt         time.Time
				updatedAt         time.Time
			)
			if err = rows.Scan(&bankAccountId, &companyId, &name, &bic, &settlementAccount, &currency, &createdAt, &updatedAt); err != nil {
				rows.Close()
				return fmt.Errorf("scan: %w", err)
			}
			batch = append(batch, bson.M{
				"_id":                bankAccountId,
				"company_id":         companyId,
				"name":               name,
				"bic":                bic,
				"settlement_account": settlementAccount,
				"currency":           currency,
				"created_at":         createdAt,
				"updated_at":         updatedAt,
			})
		}
		rows.Close()

		if len(batch) == 0 {
			break
		}

		if err = flushBatch(ctx, collection, batch); err != nil {
			return err
		}
		total += len(batch)
		fmt.Printf("    inserted %d docs\n", total)
	}

	fmt.Printf("    total: %d\n", total)
	return nil
}

func migrateAccountOperationsTable(ctx context.Context, pg *pgxpool.Pool, db *mongo.Database, batchSize int) error {
	fmt.Println("  account_operations...")

	tx, err := pg.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `DECLARE cur CURSOR FOR
		SELECT operation_id, account_id, counterparty_id, transaction_id,
		       operation_type, operation_status, amount, balance_after, created_at, updated_at
		FROM account_operations`)
	if err != nil {
		return fmt.Errorf("declare cursor: %w", err)
	}

	collection := db.Collection("account_operations")
	total := 0

	for {
		rows, err := tx.Query(ctx, fmt.Sprintf("FETCH %d FROM cur", batchSize))
		if err != nil {
			return fmt.Errorf("fetch: %w", err)
		}

		var batch []interface{}
		for rows.Next() {
			var (
				operationId     uuid.UUID
				accountId       uuid.UUID
				counterpartyId  uuid.UUID
				transactionId   uuid.UUID
				operationType   string
				operationStatus string
				amount          int64
				balanceAfter    int64
				createdAt       time.Time
				updatedAt       time.Time
			)
			if err = rows.Scan(&operationId, &accountId, &counterpartyId, &transactionId,
				&operationType, &operationStatus, &amount, &balanceAfter, &createdAt, &updatedAt); err != nil {
				rows.Close()
				return fmt.Errorf("scan: %w", err)
			}
			batch = append(batch, bson.M{
				"_id":              operationId,
				"account_id":       accountId,
				"counterparty_id":  counterpartyId,
				"transaction_id":   transactionId,
				"operation_type":   operationType,
				"operation_status": operationStatus,
				"amount":           amount,
				"balance_after":    balanceAfter,
				"created_at":       createdAt,
				"updated_at":       updatedAt,
			})
		}
		rows.Close()

		if len(batch) == 0 {
			break
		}

		if err = flushBatch(ctx, collection, batch); err != nil {
			return err
		}
		total += len(batch)
		fmt.Printf("    inserted %d docs\n", total)
	}

	fmt.Printf("    total: %d\n", total)
	return nil
}

func migrateBankOperationsTable(ctx context.Context, pg *pgxpool.Pool, db *mongo.Database, batchSize int) error {
	fmt.Println("  bank_operations...")

	tx, err := pg.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `DECLARE cur CURSOR FOR
		SELECT bank_operation_id, account_id, bank_account_id, initiator_id,
		       bank_name, operation_type, operation_status, amount, balance_after,
		       idempotency_key, external_id, created_at, updated_at
		FROM bank_operations`)
	if err != nil {
		return fmt.Errorf("declare cursor: %w", err)
	}

	collection := db.Collection("bank_operations")
	total := 0

	for {
		rows, err := tx.Query(ctx, fmt.Sprintf("FETCH %d FROM cur", batchSize))
		if err != nil {
			return fmt.Errorf("fetch: %w", err)
		}

		var batch []interface{}
		for rows.Next() {
			var (
				bankOperationId uuid.UUID
				accountId       uuid.UUID
				bankAccountId   uuid.UUID
				initiatorId     uuid.UUID
				bankName        string
				operationType   string
				operationStatus string
				amount          int64
				balanceAfter    int64
				idempotencyKey  string
				externalId      string
				createdAt       time.Time
				updatedAt       time.Time
			)
			if err = rows.Scan(&bankOperationId, &accountId, &bankAccountId, &initiatorId,
				&bankName, &operationType, &operationStatus, &amount, &balanceAfter,
				&idempotencyKey, &externalId, &createdAt, &updatedAt); err != nil {
				rows.Close()
				return fmt.Errorf("scan: %w", err)
			}
			batch = append(batch, bson.M{
				"_id":              bankOperationId,
				"account_id":       accountId,
				"bank_account_id":  bankAccountId,
				"initiator_id":     initiatorId,
				"bank_name":        bankName,
				"operation_type":   operationType,
				"operation_status": operationStatus,
				"amount":           amount,
				"balance_after":    balanceAfter,
				"idempotency_key":  idempotencyKey,
				"external_id":      externalId,
				"created_at":       createdAt,
				"updated_at":       updatedAt,
			})
		}
		rows.Close()

		if len(batch) == 0 {
			break
		}

		if err = flushBatch(ctx, collection, batch); err != nil {
			return err
		}
		total += len(batch)
		fmt.Printf("    inserted %d docs\n", total)
	}

	fmt.Printf("    total: %d\n", total)
	return nil
}

func migrateStatementsTable(ctx context.Context, pg *pgxpool.Pool, db *mongo.Database, batchSize int) error {
	fmt.Println("  statements...")

	tx, err := pg.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `DECLARE cur CURSOR FOR
		SELECT statement_id, account_id, company_id, initiator_id,
		       period_from, period_to, opening_balance, closing_balance,
		       total_debit, total_credit, currency, entries, created_at, updated_at
		FROM statements`)
	if err != nil {
		return fmt.Errorf("declare cursor: %w", err)
	}

	collection := db.Collection("statements")
	total := 0

	for {
		rows, err := tx.Query(ctx, fmt.Sprintf("FETCH %d FROM cur", batchSize))
		if err != nil {
			return fmt.Errorf("fetch: %w", err)
		}

		var batch []interface{}
		for rows.Next() {
			var (
				statementId    uuid.UUID
				accountId      uuid.UUID
				companyId      uuid.UUID
				initiatorId    uuid.UUID
				periodFrom     time.Time
				periodTo       time.Time
				openingBalance int64
				closingBalance int64
				totalDebit     int64
				totalCredit    int64
				currency       string
				entriesJSON    []byte
				createdAt      time.Time
				updatedAt      time.Time
			)
			if err = rows.Scan(&statementId, &accountId, &companyId, &initiatorId,
				&periodFrom, &periodTo, &openingBalance, &closingBalance,
				&totalDebit, &totalCredit, &currency, &entriesJSON, &createdAt, &updatedAt); err != nil {
				rows.Close()
				return fmt.Errorf("scan: %w", err)
			}

			var entries []interface{}
			if err = json.Unmarshal(entriesJSON, &entries); err != nil {
				rows.Close()
				return fmt.Errorf("unmarshal entries: %w", err)
			}

			batch = append(batch, bson.M{
				"_id":             statementId,
				"account_id":      accountId,
				"company_id":      companyId,
				"initiator_id":    initiatorId,
				"period_from":     periodFrom,
				"period_to":       periodTo,
				"opening_balance": openingBalance,
				"closing_balance": closingBalance,
				"total_debit":     totalDebit,
				"total_credit":    totalCredit,
				"currency":        currency,
				"entries":         entries,
				"created_at":      createdAt,
				"updated_at":      updatedAt,
			})
		}
		rows.Close()

		if len(batch) == 0 {
			break
		}

		if err = flushBatch(ctx, collection, batch); err != nil {
			return err
		}
		total += len(batch)
		fmt.Printf("    inserted %d docs\n", total)
	}

	fmt.Printf("    total: %d\n", total)
	return nil
}
