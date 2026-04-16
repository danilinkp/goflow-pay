package authgrpc

import (
	"auth/internal/domain"
	"auth/internal/domain/entities"
	"auth/internal/services"
	"context"
	"errors"
	authv1 "shared/pkg/gen/go/auth/v1"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type AuthProvider interface {
	RegisterWithNewCompany(ctx context.Context, in services.RegisterCompanyInput) (*services.AuthOutput, error)
	RegisterWithExistingCompany(ctx context.Context, in services.RegisterEmployeeInput) (*services.AuthOutput, error)
	Login(ctx context.Context, request services.LoginInput) (*services.AuthOutput, error)
	Logout(ctx context.Context, token string) error
	GetUsersByCompanyId(ctx context.Context, companyId uuid.UUID) ([]*entities.User, error)
	GetUserById(ctx context.Context, userId uuid.UUID) (*entities.User, error)
}

type AuthServer struct {
	authv1.UnimplementedAuthServiceServer
	svc AuthProvider
}

func NewAuthServer(svc AuthProvider) *AuthServer {
	return &AuthServer{
		svc: svc,
	}
}

func Register(gRPCServer *grpc.Server, svc AuthProvider) {
	authv1.RegisterAuthServiceServer(gRPCServer, NewAuthServer(svc))
}

func (s *AuthServer) RegisterWithNewCompany(ctx context.Context, req *authv1.RegisterWithNewCompanyRequest) (*authv1.AuthResponse, error) {
	out, err := s.svc.RegisterWithNewCompany(ctx, services.RegisterCompanyInput{
		Login:       req.GetLogin(),
		Email:       req.GetEmail(),
		Password:    req.GetPassword(),
		CompanyName: req.GetCompanyName(),
	})
	if err != nil {
		return nil, mapError(err)
	}

	return mapAuthOutputToResponse(out), nil
}

func (s *AuthServer) RegisterWithExistingCompany(ctx context.Context, req *authv1.RegisterWithExistingCompanyRequest) (*authv1.AuthResponse, error) {
	out, err := s.svc.RegisterWithExistingCompany(ctx, services.RegisterEmployeeInput{
		Login:             req.GetLogin(),
		Email:             req.GetEmail(),
		Password:          req.GetPassword(),
		CompanyInviteCode: req.GetCompanyInviteCode(),
	})
	if err != nil {
		return nil, mapError(err)
	}

	return mapAuthOutputToResponse(out), nil
}

func (s *AuthServer) Login(ctx context.Context, req *authv1.LoginRequest) (*authv1.AuthResponse, error) {
	out, err := s.svc.Login(ctx, services.LoginInput{
		Email:    req.GetEmail(),
		Password: req.GetPassword(),
	})
	if err != nil {
		return nil, mapError(err)
	}

	return mapAuthOutputToResponse(out), nil
}

func (s *AuthServer) Logout(ctx context.Context, req *authv1.LogoutRequest) (*authv1.LogoutResponse, error) {
	if err := s.svc.Logout(ctx, req.GetToken()); err != nil {
		return nil, mapError(err)
	}
	return &authv1.LogoutResponse{}, nil
}

func (s *AuthServer) GetUsersByCompanyId(ctx context.Context, req *authv1.GetUsersByCompanyIdRequest) (*authv1.GetUsersByCompanyIdResponse, error) {
	companyID, err := uuid.Parse(req.GetCompanyId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid company_id")
	}

	users, err := s.svc.GetUsersByCompanyId(ctx, companyID)
	if err != nil {
		return nil, mapError(err)
	}

	pbUsers := make([]*authv1.UserInfo, len(users))
	for i, u := range users {
		pbUsers[i] = mapUserToProto(u)
	}

	return &authv1.GetUsersByCompanyIdResponse{Users: pbUsers}, nil
}

func (s *AuthServer) GetUserById(ctx context.Context, req *authv1.GetUserByIdRequest) (*authv1.GetUserByIdResponse, error) {
	userId, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	user, err := s.svc.GetUserById(ctx, userId)
	if err != nil {
		return nil, mapError(err)
	}

	pbUser := mapUserToProto(user)
	return &authv1.GetUserByIdResponse{User: pbUser}, nil
}

func mapAuthOutputToResponse(out *services.AuthOutput) *authv1.AuthResponse {
	return &authv1.AuthResponse{
		UserId:    out.UserID.String(),
		CompanyId: out.CompanyID.String(),
		Email:     out.Email,
		Role:      out.Role,
		Token:     out.Token,
	}
}

func mapUserToProto(u *entities.User) *authv1.UserInfo {
	return &authv1.UserInfo{
		UserId:    u.UserId().String(),
		CompanyId: u.CompanyId().String(),
		Login:     u.Login(),
		Email:     u.Email(),
		Role:      u.Role().String(),
		CreatedAt: timestamppb.New(u.CreatedAt()),
	}
}

func mapError(err error) error {
	switch {
	case errors.Is(err, services.ErrInvalidCredentials):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Is(err, services.ErrInvalidInviteCode):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrUserNotFound),
		errors.Is(err, domain.ErrCompanyNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, domain.ErrUserAlreadyExists),
		errors.Is(err, domain.ErrCompanyAlreadyExists):
		return status.Error(codes.AlreadyExists, err.Error())
	default:
		return status.Error(codes.Internal, "internal error")
	}
}
