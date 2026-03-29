package response

import "github.com/google/uuid"

type AuthResponse struct {
	UserID    uuid.UUID
	CompanyID uuid.UUID
	Email     string
	Role      string
	Token     string
}
