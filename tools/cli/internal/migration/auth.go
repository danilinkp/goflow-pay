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

func MigrateAuth(ctx context.Context, postgresDSN, mongoDSN, mongoDBName string) error {
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

	if err = migrateCompaniesTable(ctx, pg, db); err != nil {
		return err
	}
	if err = migrateUsersTable(ctx, pg, db); err != nil {
		return err
	}

	return nil
}

func migrateCompaniesTable(ctx context.Context, pg *pgxpool.Pool, db *mongo.Database) error {
	fmt.Println("  companies...")

	rows, err := pg.Query(ctx, `
		SELECT company_id, name, invite_code, created_at, updated_at
		FROM companies
	`)
	if err != nil {
		return fmt.Errorf("query companies: %w", err)
	}
	defer rows.Close()

	var docs []interface{}
	for rows.Next() {
		var (
			companyId  uuid.UUID
			name       string
			inviteCode string
			createdAt  time.Time
			updatedAt  time.Time
		)
		if err = rows.Scan(&companyId, &name, &inviteCode, &createdAt, &updatedAt); err != nil {
			return fmt.Errorf("scan company: %w", err)
		}
		docs = append(docs, bson.M{
			"_id":         companyId,
			"name":        name,
			"invite_code": inviteCode,
			"created_at":  createdAt,
			"updated_at":  updatedAt,
		})
	}

	return bulkInsert(ctx, db.Collection("companies"), docs)
}

func migrateUsersTable(ctx context.Context, pg *pgxpool.Pool, db *mongo.Database) error {
	fmt.Println("  users...")

	rows, err := pg.Query(ctx, `
		SELECT user_id, company_id, login, email, password_hash, role, created_at, updated_at
		FROM users
	`)
	if err != nil {
		return fmt.Errorf("query users: %w", err)
	}
	defer rows.Close()

	var docs []interface{}
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
			return fmt.Errorf("scan user: %w", err)
		}
		docs = append(docs, bson.M{
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

	return bulkInsert(ctx, db.Collection("users"), docs)
}
