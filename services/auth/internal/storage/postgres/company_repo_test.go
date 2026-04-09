package postgres_test

import (
	"auth/internal/domain/entities"
	"auth/internal/storage/postgres"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompanyRepo_Save_And_GetById(t *testing.T) {
	t.Cleanup(func() { truncate(t) })
	ctx := context.Background()
	repo := postgres.NewCompanyRepo(testPool, testGetter)

	company, err := entities.NewCompany("Tech Company")
	require.NoError(t, err)

	err = repo.Save(ctx, company)
	require.NoError(t, err)

	got, err := repo.GetById(ctx, company.CompanyId())
	require.NoError(t, err)
	assert.Equal(t, company.Name(), got.Name())
	assert.Equal(t, company.InviteCode(), got.InviteCode())
}

func TestCompanyRepo_GetByInviteCode(t *testing.T) {
	t.Cleanup(func() { truncate(t) })
	ctx := context.Background()
	repo := postgres.NewCompanyRepo(testPool, testGetter)

	company, _ := entities.NewCompany("Alpha")
	code := company.InviteCode()
	_ = repo.Save(ctx, company)

	got, err := repo.GetByInviteCode(ctx, code)
	require.NoError(t, err)
	assert.Equal(t, company.CompanyId(), got.CompanyId())
}
