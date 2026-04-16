package services

import "github.com/google/uuid"

type LoginInput struct {
	Email    string
	Password string
}

type RegisterCompanyInput struct {
	Login       string
	Email       string
	Password    string
	CompanyName string
}

type RegisterEmployeeInput struct {
	Login             string
	Email             string
	Password          string
	CompanyInviteCode string
}

type AuthOutput struct {
	UserID    uuid.UUID
	CompanyID uuid.UUID
	Email     string
	Role      string
	Token     string
}
