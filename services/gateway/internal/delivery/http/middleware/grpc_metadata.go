package middleware

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
)

func GRPCContext(c *gin.Context) context.Context {
	userID, _ := c.Get(ContextUserID)
	md := metadata.Pairs("x-user-id", userID.(uuid.UUID).String())
	return metadata.NewOutgoingContext(c.Request.Context(), md)
}
