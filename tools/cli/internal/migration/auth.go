package migration

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func MigrateAuth(ctx context.Context, postgresDSN, mongoDSN, mongoDBName string, batchSize int) error {
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

	if err = migrateCompaniesTable(ctx, pg, db, batchSize); err != nil {
		return err
	}
	if err = migrateUsersTable(ctx, pg, db, batchSize); err != nil {
		return err
	}

	return nil
}

func migrateCompaniesTable(ctx context.Context, pg *pgxpool.Pool, db *mongo.Database, batchSize int) error {
	fmt.Println("  companies...")

	tx, err := pg.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, "DECLARE cur CURSOR FOR SELECT company_id, name, invite_code, created_at, updated_at FROM companies")
	if err != nil {
		return fmt.Errorf("declare cursor: %w", err)
	}

	collection := db.Collection("companies")
	total := 0

	for {
		rows, err := tx.Query(ctx, fmt.Sprintf("FETCH %d FROM cur", batchSize))
		if err != nil {
			return fmt.Errorf("fetch: %w", err)
		}

		var batch []interface{}
		for rows.Next() {
			var (
				companyId  uuid.UUID
				name       string
				inviteCode string
				createdAt  time.Time
				updatedAt  time.Time
			)
			if err = rows.Scan(&companyId, &name, &inviteCode, &createdAt, &updatedAt); err != nil {
				rows.Close()
				return fmt.Errorf("scan: %w", err)
			}
			batch = append(batch, bson.M{
				"_id":         companyId,
				"name":        name,
				"invite_code": inviteCode,
				"created_at":  createdAt,
				"updated_at":  updatedAt,
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

func migrateUsersTable(ctx context.Context, pg *pgxpool.Pool, db *mongo.Database, batchSize int) error {
	fmt.Println("  users...")

	tx, err := pg.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, "DECLARE cur CURSOR FOR SELECT user_id, company_id, login, email, password_hash, role, created_at, updated_at FROM users")
	if err != nil {
		return fmt.Errorf("declare cursor: %w", err)
	}

	collection := db.Collection("users")
	total := 0

	for {
		rows, err := tx.Query(ctx, fmt.Sprintf("FETCH %d FROM cur", batchSize))
		if err != nil {
			return fmt.Errorf("fetch: %w", err)
		}

		var batch []interface{}
		for rows.Next() {
			var (
				userId       uuid.UUID
				companyId    uuid.UUID
				login        string
				email        string
				passwordHash string
				role         string
				createdAt    time.Time
				updatedAt    time.Time
			)
			if err = rows.Scan(&userId, &companyId, &login, &email, &passwordHash, &role, &createdAt, &updatedAt); err != nil {
				rows.Close()
				return fmt.Errorf("scan: %w", err)
			}
			batch = append(batch, bson.M{
				"_id":           userId,
				"company_id":    companyId,
				"login":         login,
				"email":         email,
				"password_hash": passwordHash,
				"role":          role,
				"created_at":    createdAt,
				"updated_at":    updatedAt,
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
