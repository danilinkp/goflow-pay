package testutils

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// TruncatePostgresTables dynamically discovers all user tables in public schema
// (excluding goose_db_version and explicitly excluded tables) and executes
// TRUNCATE TABLE ... RESTART IDENTITY CASCADE.
func TruncatePostgresTables(ctx context.Context, pool *pgxpool.Pool, excludedTables ...string) error {
	excluded := map[string]bool{
		"goose_db_version": true,
	}
	for _, t := range excludedTables {
		excluded[t] = true
	}

	query := `
		SELECT table_name 
		FROM information_schema.tables 
		WHERE table_schema = 'public' 
		  AND table_type = 'BASE TABLE';
	`
	rows, err := pool.Query(ctx, query)
	if err != nil {
		return fmt.Errorf("testutils: list tables: %w", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return fmt.Errorf("testutils: scan table: %w", err)
		}
		if !excluded[tableName] {
			tables = append(tables, fmt.Sprintf("%q", tableName))
		}
	}

	if len(tables) == 0 {
		return nil
	}

	truncateSQL := fmt.Sprintf("TRUNCATE TABLE %s RESTART IDENTITY CASCADE;", strings.Join(tables, ", "))
	if _, err := pool.Exec(ctx, truncateSQL); err != nil {
		return fmt.Errorf("testutils: truncate tables: %w", err)
	}

	return nil
}

// FlushRedis purges all keys from the current Redis database (clears active user sessions/tokens).
func FlushRedis(ctx context.Context, client *redis.Client) error {
	if client == nil {
		return nil
	}
	if err := client.FlushDB(ctx).Err(); err != nil {
		return fmt.Errorf("testutils: flush redis: %w", err)
	}
	return nil
}

// DrainOutbox removes all records from outbox table to ensure no residual messages remain.
func DrainOutbox(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, "DELETE FROM outbox;")
	if err != nil && !strings.Contains(err.Error(), "does not exist") {
		return fmt.Errorf("testutils: drain outbox: %w", err)
	}
	return nil
}

// ResetTestState combines table truncation and outbox drainage for complete test isolation.
func ResetTestState(ctx context.Context, pool *pgxpool.Pool, redisClient *redis.Client) error {
	if err := TruncatePostgresTables(ctx, pool); err != nil {
		return err
	}
	if err := DrainOutbox(ctx, pool); err != nil {
		return err
	}
	if redisClient != nil {
		if err := FlushRedis(ctx, redisClient); err != nil {
			return err
		}
	}
	return nil
}
