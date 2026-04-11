package postgres_test

import (
	"auth/internal/domain/entities"
	"auth/internal/storage/postgres"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserRepo_Save_And_GetByEmail(t *testing.T) {
	t.Cleanup(func() { truncate(t) })
	ctx := context.Background()
	userRepo := postgres.NewUserRepo(testPool, testGetter)
	compRepo := postgres.NewCompanyRepo(testPool, testGetter)

	company, _ := entities.NewCompany("UserCorp")
	require.NoError(t, compRepo.Save(ctx, company))

	email := "test@example.com"
	user, err := entities.NewUser(
		company.CompanyId(),
		"daniil_dev",
		email,
		"hashed_password",
		"admin",
	)
	require.NoError(t, err)

	err = userRepo.Save(ctx, user)
	require.NoError(t, err)

	got, err := userRepo.GetByEmail(ctx, email)
	require.NoError(t, err)
	assert.Equal(t, user.UserId(), got.UserId())
	assert.Equal(t, user.Login(), got.Login())
}

func TestUserRepo_GetByCompanyId(t *testing.T) {
	t.Cleanup(func() { truncate(t) })
	ctx := context.Background()
	userRepo := postgres.NewUserRepo(testPool, testGetter)
	compRepo := postgres.NewCompanyRepo(testPool, testGetter)

	company, err := entities.NewCompany("MultiUser")
	require.NoError(t, err)
	err = compRepo.Save(ctx, company)
	require.NoError(t, err)

	u1, err := entities.NewUser(company.CompanyId(), "u1", "u1@e.com", "pass", "employee")
	require.NoError(t, err)
	u2, err := entities.NewUser(company.CompanyId(), "u2", "u2@e.com", "pass", "employee")
	require.NoError(t, err)
	err = userRepo.Save(ctx, u1)
	require.NoError(t, err)
	err = userRepo.Save(ctx, u2)
	require.NoError(t, err)

	users, err := userRepo.GetByCompanyId(ctx, company.CompanyId())
	require.NoError(t, err)
	assert.Len(t, users, 2)
}
