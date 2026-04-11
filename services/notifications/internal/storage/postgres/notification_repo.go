package postgres

import (
	"context"
	"errors"
	"fmt"
	"notifications/internal/domain"
	"notifications/internal/domain/entities"
	"notifications/internal/storage/postgres/models"
	postgresLib "shared/db/postgres"

	trmpgx "github.com/avito-tech/go-transaction-manager/drivers/pgxv5/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type NotificationRepo struct {
	pool    *pgxpool.Pool
	getter  *trmpgx.CtxGetter
	factory *postgresLib.SelectFactory[models.NotificationModel]
}

func NewNotificationRepo(pool *pgxpool.Pool, c *trmpgx.CtxGetter) *NotificationRepo {
	return &NotificationRepo{
		pool:    pool,
		getter:  c,
		factory: postgresLib.NewSelectFactory[models.NotificationModel](pool, c, "notifications", models.NotificationColumns()),
	}
}

func (r *NotificationRepo) Save(ctx context.Context, notification *entities.Notification) error {
	op := "NotificationRepo.Save"
	conn := r.getter.DefaultTrOrDB(ctx, r.pool)
	notificationModel := models.ToNotificationModel(notification)

	query := `INSERT INTO notifications(notification_id, user_id, title, message, source_id, updated_at, created_at)
			  VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := conn.Exec(ctx, query,
		notificationModel.NotificationId,
		notificationModel.UserId,
		notificationModel.Title,
		notificationModel.Message,
		notificationModel.SourceId,
		notificationModel.UpdatedAt,
		notificationModel.CreatedAt,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return fmt.Errorf("%s: %w", op, domain.ErrNotificationAlreadyExists)
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (r *NotificationRepo) GetById(ctx context.Context, id uuid.UUID) (*entities.Notification, error) {
	op := "NotificationRepo.GetById"
	notificationModel, err := r.factory.GetOne(ctx, "notification_id = $1", id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, domain.ErrNotificationNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return notificationModel.ToDomain(), nil
}
