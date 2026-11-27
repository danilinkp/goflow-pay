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

type CompanyRepo struct {
	pool    *pgxpool.Pool
	getter  *trmpgx.CtxGetter
	factory *postgresLib.SelectFactory[models.CompanyModel]
}

func NewCompanyRepo(pool *pgxpool.Pool, c *trmpgx.CtxGetter) *CompanyRepo {
	return &CompanyRepo{
		pool:    pool,
		getter:  c,
		factory: postgresLib.NewSelectFactory[models.CompanyModel](pool, c, "companies", models.CompanyColumns()),
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

	companyModel, err := r.factory.GetOne(ctx, "company_id = $1", companyId)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, domain.ErrCompanyNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return companyModel.ToDomain(), nil
}

func (r *CompanyRepo) GetByInviteCode(ctx context.Context, inviteCode string) (*entities.Company, error) {
	op := "CompanyRepo.GetByInviteCode"

	companyModel, err := r.factory.GetOne(ctx, "invite_code = $1", inviteCode)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, domain.ErrCompanyNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return companyModel.ToDomain(), nil
}

func (r *CompanyRepo) GetInviteCodeById(ctx context.Context, companyId uuid.UUID) (string, error) {
	op := "CompanyRepo.GetInviteCodeById"

	conn := r.getter.DefaultTrOrDB(ctx, r.pool)

	var inviteCode string
	err := conn.QueryRow(ctx, "SELECT invite_code FROM companies WHERE company_id = $1", companyId).Scan(&inviteCode)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", fmt.Errorf("%s: %w", op, domain.ErrCompanyNotFound)
		}
		return "", fmt.Errorf("%s: %w", op, err)
	}

	return inviteCode, nil
}

func (r *CompanyRepo) GetAll(ctx context.Context, search string, limit, offset int) ([]*entities.Company, int, error) {
	op := "CompanyRepo.GetAll"
	conn := r.getter.DefaultTrOrDB(ctx, r.pool)

	query := "SELECT company_id, name, invite_code, created_at, updated_at FROM companies"
	countQuery := "SELECT COUNT(*) FROM companies"
	var args []any
	var countArgs []any

	if search != "" {
		query += " WHERE name ILIKE $1"
		countQuery += " WHERE name ILIKE $1"
		args = append(args, "%"+search+"%")
		countArgs = append(countArgs, "%"+search+"%")
	}

	var total int
	if err := conn.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("%s: %w", op, err)
	}

	query += " ORDER BY created_at DESC"
	if limit > 0 {
		args = append(args, limit)
		query += fmt.Sprintf(" LIMIT $%d", len(args))
	}
	if offset > 0 {
		args = append(args, offset)
		query += fmt.Sprintf(" OFFSET $%d", len(args))
	}

	rows, err := conn.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var companies []*entities.Company
	for rows.Next() {
		var m models.CompanyModel
		if err := rows.Scan(&m.CompanyId, &m.Name, &m.InviteCode, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("%s: %w", op, err)
		}
		companies = append(companies, m.ToDomain())
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("%s: %w", op, err)
	}
	return companies, total, nil
}
