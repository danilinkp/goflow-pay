package grpcclient

import (
	"context"
	"fmt"
	"notifications/internal/services"
	authv1 "shared/pkg/gen/go/auth/v1"

	"github.com/google/uuid"
	"google.golang.org/grpc"
)

type UserGrpcClient struct {
	client authv1.AuthServiceClient
}

func NewUserGrpcClient(conn *grpc.ClientConn) *UserGrpcClient {
	return &UserGrpcClient{
		client: authv1.NewAuthServiceClient(conn),
	}
}

func (c *UserGrpcClient) GetUserById(ctx context.Context, userId uuid.UUID) (*services.UserResponse, error) {
	resp, err := c.client.GetUserById(ctx, &authv1.GetUserByIdRequest{
		UserId: userId.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("UserGrpcClient.GetUserById: %w", err)
	}

	return &services.UserResponse{
		ID:    userId,
		Email: resp.User.Email,
	}, nil
}
