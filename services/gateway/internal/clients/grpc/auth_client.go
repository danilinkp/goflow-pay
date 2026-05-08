package grpc

import (
	"context"
	"fmt"
	authv1 "shared/pkg/gen/go/auth/v1"

	"google.golang.org/grpc"
)

type AuthGRPCClient struct {
	client authv1.AuthServiceClient
}

func NewAuthGRPCClient(conn *grpc.ClientConn) *AuthGRPCClient {
	return &AuthGRPCClient{
		client: authv1.NewAuthServiceClient(conn),
	}
}

func (c *AuthGRPCClient) RegisterWithNewCompany(ctx context.Context, req *authv1.RegisterWithNewCompanyRequest) (*authv1.AuthResponse, error) {
	op := "auth.client.RegisterWithNewCompany"

	resp, err := c.client.RegisterWithNewCompany(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return resp, nil
}

func (c *AuthGRPCClient) RegisterWithExistingCompany(ctx context.Context, req *authv1.RegisterWithExistingCompanyRequest) (*authv1.AuthResponse, error) {
	op := "auth.client.RegisterWithExistingCompany"
	resp, err := c.client.RegisterWithExistingCompany(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return resp, nil
}

func (c *AuthGRPCClient) Login(ctx context.Context, req *authv1.LoginRequest) (*authv1.AuthResponse, error) {
	op := "auth.client.Login"
	resp, err := c.client.Login(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return resp, nil
}

func (c *AuthGRPCClient) Logout(ctx context.Context, req *authv1.LogoutRequest) (*authv1.LogoutResponse, error) {
	op := "auth.client.Logout"
	resp, err := c.client.Logout(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return resp, nil
}

func (c *AuthGRPCClient) GetUsersByCompanyId(ctx context.Context, req *authv1.GetUsersByCompanyIdRequest) (*authv1.GetUsersByCompanyIdResponse, error) {
	op := "auth.client.GetUsersByCompanyId"
	resp, err := c.client.GetUsersByCompanyId(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return resp, nil
}

func (c *AuthGRPCClient) GetInviteCodeByCompanyId(ctx context.Context, req *authv1.GetInviteCodeByCompanyIdRequest) (*authv1.GetInviteCodeByCompanyIdResponse, error) {
	op := "auth.client.GetInviteCodeByCompanyId"
	resp, err := c.client.GetInviteCodeByCompanyId(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return resp, nil
}

func (c *AuthGRPCClient) IsTokenValid(ctx context.Context, req *authv1.IsTokenValidRequest) (*authv1.IsTokenValidResponse, error) {
	op := "auth.client.IsTokenValid"
	resp, err := c.client.IsTokenValid(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return resp, nil
}

func (c *AuthGRPCClient) InitSystem(ctx context.Context, req *authv1.InitSystemRequest) (*authv1.AuthResponse, error) {
	op := "auth.client.InitSystem"
	resp, err := c.client.InitSystem(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return resp, nil
}

func (c *AuthGRPCClient) AddAdmin(ctx context.Context, req *authv1.AddAdminRequest) (*authv1.AuthResponse, error) {
	op := "auth.client.AddAdmin"
	resp, err := c.client.AddAdmin(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return resp, nil
}
