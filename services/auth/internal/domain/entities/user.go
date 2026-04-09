package entities

import (
	"fmt"
	"regexp"
	"time"

	"github.com/google/uuid"
)

var emailRe = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

func validateEmail(email string) bool {
	return emailRe.MatchString(email)
}

type Role string

const (
	RoleAdmin        Role = "admin"
	RoleCompanyAdmin Role = "company_admin"
	RoleEmployee     Role = "employee"
	RoleGuest        Role = "guest"
)

func (r Role) String() string {
	return string(r)
}

func (r Role) IsValid() bool {
	return r == RoleAdmin || r == RoleCompanyAdmin || r == RoleEmployee || r == RoleGuest
}

type User struct {
	userId       uuid.UUID
	companyId    uuid.UUID
	login        string
	email        string
	passwordHash string
	role         Role
	createdAt    time.Time
	updatedAt    time.Time
}

func NewUser(companyId uuid.UUID, login, email, passwordHash string, role Role) (*User, error) {
	if login == "" {
		return nil, fmt.Errorf("%s: login is required", "create user")
	}
	if !validateEmail(email) {
		return nil, fmt.Errorf("%s: invalid email format", "create user")
	}
	if passwordHash == "" {
		return nil, fmt.Errorf("%s: password hash is required", "create user")
	}
	if !role.IsValid() {
		return nil, fmt.Errorf("%s: invalid role: %v", "create user", role)
	}

	now := time.Now().UTC()
	return &User{
		userId:       uuid.New(),
		companyId:    companyId,
		login:        login,
		email:        email,
		passwordHash: passwordHash,
		role:         role,
		createdAt:    now,
		updatedAt:    now,
	}, nil
}

func ReconstructUser(id, companyId uuid.UUID, login, email, hash string, role Role, createdAt, updateAt time.Time) *User {
	return &User{
		userId:       id,
		companyId:    companyId,
		login:        login,
		email:        email,
		passwordHash: hash,
		role:         role,
		createdAt:    createdAt,
		updatedAt:    updateAt,
	}
}

func (u *User) UserId() uuid.UUID    { return u.userId }
func (u *User) CompanyId() uuid.UUID { return u.companyId }
func (u *User) Login() string        { return u.login }
func (u *User) Email() string        { return u.email }
func (u *User) Role() Role           { return u.role }
func (u *User) PasswordHash() string { return u.passwordHash }
func (u *User) CreatedAt() time.Time { return u.createdAt }
func (u *User) UpdatedAt() time.Time { return u.updatedAt }

func (u *User) UpdateEmail(newEmail string) error {
	if !validateEmail(newEmail) {
		return fmt.Errorf("%s: email must be valid", "update email")
	}
	u.email = newEmail
	u.updatedAt = time.Now().UTC()
	return nil
}

func (u *User) UpdateLogin(newLogin string) error {
	if newLogin == "" {
		return fmt.Errorf("%s: login cannot be empty", "update login")
	}
	u.login = newLogin
	u.updatedAt = time.Now().UTC()
	return nil
}

func (u *User) UpdatePasswordHash(newPasswordHash string) error {
	if newPasswordHash == "" {
		return fmt.Errorf("%s: password cannot be empty", "update password")
	}
	u.passwordHash = newPasswordHash
	u.updatedAt = time.Now().UTC()
	return nil
}

func (u *User) UpdateRole(newRole Role) error {
	if !newRole.IsValid() {
		return fmt.Errorf("%s: invalid role: %v", "update role", newRole)
	}

	u.role = newRole
	u.updatedAt = time.Now().UTC()
	return nil
}
