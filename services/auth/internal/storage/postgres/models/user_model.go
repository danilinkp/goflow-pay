package models

import (
	"auth/internal/domain/entities"
	"time"

	"github.com/google/uuid"
)

type UserModel struct {
	Id           uuid.UUID `db:"id"`
	CompanyId    uuid.UUID `db:"company_id"`
	Login        string    `db:"login"`
	Email        string    `db:"email"`
	PasswordHash string    `db:"password_hash"`
	Role         string    `db:"role"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}

func (m *UserModel) ToDomain() *entities.User {
	return entities.ReconstructUser(
		m.Id,
		m.CompanyId,
		m.Login,
		m.Email,
		m.PasswordHash,
		entities.Role(m.Role),
		m.CreatedAt,
		m.UpdatedAt,
	)
}

func ToUserModel(user *entities.User) *UserModel {
	return &UserModel{
		Id:           user.ID(),
		CompanyId:    user.CompanyID(),
		Login:        user.Login(),
		Email:        user.Email(),
		PasswordHash: user.PasswordHash(),
		Role:         user.Role().String(),
		CreatedAt:    user.CreatedAt(),
		UpdatedAt:    user.UpdatedAt(),
	}
}
