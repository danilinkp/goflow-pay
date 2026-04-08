package postgres

import (
	"auth/internal/domain"
	"auth/internal/domain/entities"
	"auth/internal/storage/postgres/models"
	"context"
	"errors"
	"fmt"

	trmpgx "github.com/avito-tech/go-transaction-manager/drivers/pgxv5/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CompanyRepo struct {
	pool   *pgxpool.Pool
	getter *trmpgx.CtxGetter
}

func NewCompanyRepo(pool *pgxpool.Pool, c *trmpgx.CtxGetter) *CompanyRepo {
	return &CompanyRepo{
		pool:   pool,
		getter: c,
	}
}

func (r *CompanyRepo) Save(ctx context.Context, company *entities.Company) error {
	op := "CompanyRepo.Save"

	conn := r.getter.DefaultTrOrDB(ctx, r.pool)

	companyModel := models.ToCompanyModel(company)

	query := `INSERT INTO companies(company_id, name, invite_code, created_at, updated_at)
			  VALUES ($1, $2, $3, $4, $5)`

	_, err := conn.Exec(ctx, query,
		companyModel.CompanyId,
		companyModel.Name,
		companyModel.InviteCode,
		companyModel.CreatedAt,
		companyModel.UpdatedAt,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return fmt.Errorf("%s: %w", op, domain.ErrCompanyAlreadyExists)
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (r *CompanyRepo) GetById(ctx context.Context, companyId uuid.UUID) (*entities.Company, error) {
	op := "CompanyRepo.GetById"

	conn := r.getter.DefaultTrOrDB(ctx, r.pool)

	query := `SELECT company_id, name, invite_code, created_at, updated_at 
			  FROM companies WHERE id = $1;`

	var companyModel models.CompanyModel
	err := conn.QueryRow(ctx, query, companyId).Scan(
		&companyModel.CompanyId,
		&companyModel.Name,
		&companyModel.InviteCode,
		&companyModel.CreatedAt,
		&companyModel.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, domain.ErrCompanyNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return companyModel.ToDomain(), nil
}

func (r *CompanyRepo) GetByInviteCode(ctx context.Context, inviteCode string) (*entities.Company, error) {
	op := "CompanyRepo.GetById"
	conn := r.getter.DefaultTrOrDB(ctx, r.pool)

	query := `SELECT company_id, name, invite_code, created_at, updated_at 
			  FROM companies WHERE invite_code = $1;`

	var companyModel models.CompanyModel
	err := conn.QueryRow(ctx, query, inviteCode).Scan(
		&companyModel.CompanyId,
		&companyModel.Name,
		&companyModel.InviteCode,
		&companyModel.CreatedAt,
		&companyModel.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, domain.ErrCompanyNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return companyModel.ToDomain(), nil
}
