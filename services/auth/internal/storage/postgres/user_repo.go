package postgres

import (
	"auth/internal/domain"
	"auth/internal/domain/entities"
	"auth/internal/storage/postgres/models"
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{
		pool: pool,
	}
}

func (r *UserRepo) Save(ctx context.Context, user *entities.User) error {
	op := "UserRepo.Save"

	userModel := models.ToUserModel(user)

	query := `INSERT INTO users(id, company_id, login, email, password_hash, role, created_at, updated_at)
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := r.pool.Exec(ctx, query,
		userModel.Id,
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

	query := `SELECT id, company_id, login, email, password_hash, role, created_at, updated_at 
			  FROM users WHERE id = $1;`

	var user models.UserModel
	err := r.pool.QueryRow(ctx, query, userId).Scan(
		&user.Id,
		&user.CompanyId,
		&user.Email,
		&user.Role,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

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

	query := `SELECT id, company_id, login, email, password_hash, role, created_at, updated_at 
			  FROM users WHERE email = $1;`

	var user models.UserModel
	err := r.pool.QueryRow(ctx, query, email).Scan(
		&user.Id,
		&user.CompanyId,
		&user.Login,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

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

	query := `SELECT id, company_id, login, email, password_hash, role, created_at, updated_at from users WHERE company_id = $1;`

	rows, err := r.pool.Query(ctx, query, companyId)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	users, err := pgx.CollectRows(rows, pgx.RowToStructByName[models.UserModel])
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	res := make([]*entities.User, len(users))
	for i, user := range users {
		res[i] = user.ToDomain()
	}

	return res, nil
}
