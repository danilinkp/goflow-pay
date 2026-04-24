package postgres

import (
	"auth/internal/domain"
	"auth/internal/domain/entities"
	"auth/internal/storage/postgres/models"
	"context"
	"errors"
	"fmt"
	postgresLib "shared/pkg/db/postgres"

	trmpgx "github.com/avito-tech/go-transaction-manager/drivers/pgxv5/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepo struct {
	pool    *pgxpool.Pool
	getter  *trmpgx.CtxGetter
	factory *postgresLib.SelectFactory[models.UserModel]
}

func NewUserRepo(pool *pgxpool.Pool, c *trmpgx.CtxGetter) *UserRepo {
	return &UserRepo{
		pool:    pool,
		getter:  c,
		factory: postgresLib.NewSelectFactory[models.UserModel](pool, c, "users", models.UserColumns()),
	}
}

func (r *UserRepo) Save(ctx context.Context, user *entities.User) error {
	op := "UserRepo.Save"

	conn := r.getter.DefaultTrOrDB(ctx, r.pool)

	userModel := models.ToUserModel(user)

	query := `INSERT INTO users(user_id, company_id, login, email, password_hash, role, created_at, updated_at)
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := conn.Exec(ctx, query,
		userModel.UserId,
		userModel.CompanyId,
		userModel.Login,
		userModel.Email,
		userModel.PasswordHash,
		userModel.Role,
		userModel.CreatedAt,
		userModel.UpdatedAt,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return fmt.Errorf("%s: %w", op, domain.ErrUserAlreadyExists)
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}
func (r *UserRepo) GetById(ctx context.Context, userId uuid.UUID) (*entities.User, error) {
	op := "UserRepo.GetById"

	user, err := r.factory.GetOne(ctx, "user_id = $1", userId)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, domain.ErrUserNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return user.ToDomain(), nil
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*entities.User, error) {
	op := "UserRepo.GetByEmail"

	user, err := r.factory.GetOne(ctx, "email = $1", email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, domain.ErrUserNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return user.ToDomain(), nil
}

func (r *UserRepo) GetByCompanyId(ctx context.Context, companyId uuid.UUID) ([]*entities.User, error) {
	op := "UserRepo.GetByCompanyId"
	users, err := r.factory.List(ctx, "company_id = $1", companyId)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	res := make([]*entities.User, len(users))
	for i, user := range users {
		res[i] = user.ToDomain()
	}

	return res, nil
}
