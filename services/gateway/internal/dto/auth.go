package dto

import (
	"time"

	"github.com/google/uuid"
)

type RegisterWithNewCompanyRequest struct {
	Login       string `json:"login"`
	Email       string `json:"email"`
	Password    string `json:"password"`
	CompanyName string `json:"company_name"`
}

type RegisterWithExistingCompanyRequest struct {
	Login      string `json:"login"`
	Email      string `json:"email"`
	Password   string `json:"password"`
	InviteCode string `json:"invite_code"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UserResponse struct {
	UserId    uuid.UUID `json:"user_id"`
	CompanyId uuid.UUID `json:"company_id"`
	Login     string    `json:"login"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

type AuthResponse struct {
	UserId    uuid.UUID `json:"user_id"`
	CompanyId uuid.UUID `json:"company_id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Token     string    `json:"token"`
}
