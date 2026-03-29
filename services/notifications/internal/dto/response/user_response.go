package response

import "github.com/google/uuid"

type UserResponse struct {
	ID    uuid.UUID
	Email string
}
