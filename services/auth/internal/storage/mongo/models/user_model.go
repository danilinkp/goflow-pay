package models

import (
	"auth/internal/domain/entities"
	"time"

	"github.com/google/uuid"
)

type UserModel struct {
	UserId       uuid.UUID `bson:"_id"`
	CompanyId    uuid.UUID `bson:"company_id"`
	Login        string    `bson:"login"`
	Email        string    `bson:"email"`
	PasswordHash string    `bson:"password_hash"`
	Role         string    `bson:"role"`
	CreatedAt    time.Time `bson:"created_at"`
	UpdatedAt    time.Time `bson:"updated_at"`
}

func (m *UserModel) ToDomain() *entities.User {
	return entities.ReconstructUser(
		m.UserId,
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
		UserId:       user.UserId(),
		CompanyId:    user.CompanyId(),
		Login:        user.Login(),
		Email:        user.Email(),
		PasswordHash: user.PasswordHash(),
		Role:         user.Role().String(),
		CreatedAt:    user.CreatedAt(),
		UpdatedAt:    user.UpdatedAt(),
	}
}
