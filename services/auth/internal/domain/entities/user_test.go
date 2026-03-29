package entities_test

import (
	"auth/internal/domain/entities"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUser_Success(t *testing.T) {
	companyId := uuid.New()

	u, err := entities.NewUser(companyId, "login", "email@test.com", "hash", entities.RoleEmployee)
	require.NoError(t, err)
	assert.Equal(t, companyId, u.CompanyID())
	assert.Equal(t, "login", u.Login())
	assert.Equal(t, "email@test.com", u.Email())
	assert.Equal(t, entities.RoleEmployee, u.Role())
}

func TestNewUser_EmptyLogin(t *testing.T) {
	_, err := entities.NewUser(uuid.New(), "", "email@test.com", "hash", entities.RoleEmployee)

	assert.Error(t, err)
}

func TestNewUser_EmptyEmail(t *testing.T) {
	_, err := entities.NewUser(uuid.New(), "login", "", "hash", entities.RoleEmployee)

	assert.Error(t, err)
}

func TestNewUser_EmptyPasswordHash(t *testing.T) {
	_, err := entities.NewUser(uuid.New(), "login", "email@test.com", "", entities.RoleEmployee)

	assert.Error(t, err)
}

func TestNewUser_InvalidRole(t *testing.T) {
	_, err := entities.NewUser(uuid.New(), "login", "email@test.com", "hash", "invalid")

	assert.Error(t, err)
}

func TestUser_UpdateEmail_Success(t *testing.T) {
	u, _ := entities.NewUser(uuid.New(), "login", "old@test.com", "hash", entities.RoleEmployee)

	err := u.UpdateEmail("new@test.com")

	require.NoError(t, err)
	assert.Equal(t, "new@test.com", u.Email())
}

func TestUser_UpdateEmail_Empty(t *testing.T) {
	u, _ := entities.NewUser(uuid.New(), "login", "old@test.com", "hash", entities.RoleEmployee)

	err := u.UpdateEmail("")

	assert.Error(t, err)
	assert.Equal(t, "old@test.com", u.Email())
}

func TestUser_UpdateRole_Success(t *testing.T) {
	u, _ := entities.NewUser(uuid.New(), "login", "email@test.com", "hash", entities.RoleEmployee)

	err := u.UpdateRole(entities.RoleCompanyAdmin)

	require.NoError(t, err)
	assert.Equal(t, entities.RoleCompanyAdmin, u.Role())
}

func TestUser_UpdateRole_Invalid(t *testing.T) {
	u, _ := entities.NewUser(uuid.New(), "login", "email@test.com", "hash", entities.RoleEmployee)

	err := u.UpdateRole(entities.Role("invalid"))

	assert.Error(t, err)
	assert.Equal(t, entities.RoleEmployee, u.Role())
}

func TestUser_UpdateLogin_Success(t *testing.T) {
	u, _ := entities.NewUser(uuid.New(), "old", "email@test.com", "hash", entities.RoleEmployee)

	err := u.UpdateLogin("new")

	require.NoError(t, err)
	assert.Equal(t, "new", u.Login())
}

func TestUser_UpdateLogin_Empty(t *testing.T) {
	u, _ := entities.NewUser(uuid.New(), "old", "email@test.com", "hash", entities.RoleEmployee)

	err := u.UpdateLogin("")

	assert.Error(t, err)
	assert.Equal(t, "old", u.Login())
}

func TestUser_UpdatePasswordHash_Success(t *testing.T) {
	u, _ := entities.NewUser(uuid.New(), "login", "email@test.com", "oldhash", entities.RoleEmployee)

	err := u.UpdatePasswordHash("newhash")

	require.NoError(t, err)
	assert.Equal(t, "newhash", u.PasswordHash())
}

func TestUser_UpdatePasswordHash_Empty(t *testing.T) {
	u, _ := entities.NewUser(uuid.New(), "login", "email@test.com", "oldhash", entities.RoleEmployee)

	err := u.UpdatePasswordHash("")

	assert.Error(t, err)
	assert.Equal(t, "oldhash", u.PasswordHash())
}
