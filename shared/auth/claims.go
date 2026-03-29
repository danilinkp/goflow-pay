package auth

import (
	"time"

	"github.com/google/uuid"
)

type AccessClaims struct {
	TokenID   string
	UserID    uuid.UUID
	CompanyID uuid.UUID
	Role      string
	ExpiresAt time.Time
}
