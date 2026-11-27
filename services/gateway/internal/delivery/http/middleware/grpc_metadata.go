package middleware

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
)

func GRPCContext(c *gin.Context) context.Context {
	userID, ok := c.Get(ContextUserID)
	if !ok || userID == nil {
		return c.Request.Context()
	}
	uid, ok := userID.(uuid.UUID)
	if !ok {
		return c.Request.Context()
	}
	md := metadata.Pairs("x-user-id", uid.String())
	return metadata.NewOutgoingContext(c.Request.Context(), md)
}
