package entities_test

import (
	"auth/internal/domain/entities"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCompany_Success(t *testing.T) {
	c, err := entities.NewCompany("Acme")

	require.NoError(t, err)
	assert.Equal(t, "Acme", c.Name())
	assert.NotEmpty(t, c.InviteCode())
	assert.NotEqual(t, uuid.Nil, c.CompanyId())
}

func TestNewCompany_EmptyName(t *testing.T) {
	_, err := entities.NewCompany("")

	assert.Error(t, err)
}

func TestCompany_UpdateName_Success(t *testing.T) {
	c, _ := entities.NewCompany("Old")

	err := c.UpdateName("New")

	require.NoError(t, err)
	assert.Equal(t, "New", c.Name())
}

func TestCompany_UpdateName_Empty(t *testing.T) {
	c, _ := entities.NewCompany("Old")

	err := c.UpdateName("")

	assert.Error(t, err)
	assert.Equal(t, "Old", c.Name())
}
