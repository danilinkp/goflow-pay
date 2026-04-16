package services

import (
	"auth/internal/domain"
	"auth/internal/domain/entities"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"shared/pkg/auth"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidInviteCode  = errors.New("invalid invite code")
)

type UserRepository interface {
	Save(ctx context.Context, user *entities.User) error
	GetById(ctx context.Context, userId uuid.UUID) (*entities.User, error)
	GetByEmail(ctx context.Context, email string) (*entities.User, error)
	GetByCompanyId(ctx context.Context, companyId uuid.UUID) ([]*entities.User, error)
}

type CompanyRepository interface {
	Save(ctx context.Context, company *entities.Company) error
	GetById(ctx context.Context, companyId uuid.UUID) (*entities.Company, error)
	GetByInviteCode(ctx context.Context, inviteCode string) (*entities.Company, error)
}

type Transactor interface {
	WithTx(ctx context.Context, fn func(context.Context) error) error
}

type BlackListRepository interface {
	Save(ctx context.Context, tokenID string, tokenTTL time.Duration) error
	Exists(ctx context.Context, tokenID string) (bool, error)
}

type TokenValidator interface {
	Validate(tokenString string) (*auth.AccessClaims, error)
}

type TokenService interface {
	Generate(userId uuid.UUID, companyId uuid.UUID, role string) (string, error)
}

type PasswordHasher interface {
	Hash(password string) (string, error)
	Verify(password, hash string) error
}

type AuthService struct {
	userRepository      UserRepository
	companyRepository   CompanyRepository
	transactor          Transactor
	blackListRepository BlackListRepository
	validator           TokenValidator
	tokenService        TokenService
	hasher              PasswordHasher
	logger              *slog.Logger
}

func NewAuthService(
	userRepo UserRepository,
	companyRepo CompanyRepository,
	transactor Transactor,
	blackListRepo BlackListRepository,
	tokenService TokenService,
	validator TokenValidator,
	hasher PasswordHasher,
	logger *slog.Logger,
) *AuthService {
	return &AuthService{
		userRepository:      userRepo,
		companyRepository:   companyRepo,
		transactor:          transactor,
		blackListRepository: blackListRepo,
		validator:           validator,
		tokenService:        tokenService,
		hasher:              hasher,
		logger:              logger,
	}
}

func (a *AuthService) RegisterWithNewCompany(ctx context.Context, in RegisterCompanyInput) (*AuthOutput, error) {
	op := "Auth.RegisterWithNewCompany"

	passwordHash, err := a.hasher.Hash(in.Password)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	company, err := entities.NewCompany(in.CompanyName)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	user, err := entities.NewUser(company.CompanyId(), in.Login, in.Email, passwordHash, entities.RoleCompanyAdmin)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	err = a.transactor.WithTx(ctx, func(txCtx context.Context) error {
		if err = a.companyRepository.Save(txCtx, company); err != nil {
			return err
		}
		return a.userRepository.Save(txCtx, user)
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	token, err := a.tokenService.Generate(user.UserId(), user.CompanyId(), string(user.Role()))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &AuthOutput{
		UserID:    user.UserId(),
		CompanyID: user.CompanyId(),
		Email:     user.Email(),
		Role:      string(user.Role()),
		Token:     token,
	}, nil
}

func (a *AuthService) RegisterWithExistingCompany(ctx context.Context, userRequest RegisterEmployeeInput) (*AuthOutput, error) {
	op := "Auth.RegisterWithExistingCompany"

	company, err := a.companyRepository.GetByInviteCode(ctx, userRequest.CompanyInviteCode)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if company.InviteCode() != userRequest.CompanyInviteCode {
		return nil, fmt.Errorf("%s: %w", op, ErrInvalidInviteCode)
	}

	passwordHash, err := a.hasher.Hash(userRequest.Password)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	user, err := entities.NewUser(company.CompanyId(), userRequest.Login, userRequest.Email, passwordHash, entities.RoleEmployee)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	err = a.userRepository.Save(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	token, err := a.tokenService.Generate(user.UserId(), user.CompanyId(), string(user.Role()))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &AuthOutput{
		UserID:    user.UserId(),
		CompanyID: user.CompanyId(),
		Email:     user.Email(),
		Role:      string(user.Role()),
		Token:     token,
	}, nil
}

func (a *AuthService) Login(ctx context.Context, request LoginInput) (*AuthOutput, error) {
	op := "Auth.Login"
	user, err := a.userRepository.GetByEmail(ctx, request.Email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, fmt.Errorf("%s: %w", op, ErrInvalidCredentials)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if err = a.hasher.Verify(request.Password, user.PasswordHash()); err != nil {
		return nil, fmt.Errorf("%s: %w", op, ErrInvalidCredentials)
	}

	token, err := a.tokenService.Generate(user.UserId(), user.CompanyId(), string(user.Role()))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &AuthOutput{
		UserID:    user.UserId(),
		CompanyID: user.CompanyId(),
		Email:     user.Email(),
		Role:      string(user.Role()),
		Token:     token,
	}, nil
}

func (a *AuthService) Logout(ctx context.Context, token string) error {
	op := "Auth.Logout"

	claims, err := a.validator.Validate(token)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	remainingTTL := time.Until(claims.ExpiresAt)
	if remainingTTL <= 0 {
		return nil
	}

	if err = a.blackListRepository.Save(ctx, claims.TokenID, remainingTTL); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (a *AuthService) IsTokenValid(ctx context.Context, token string) (*auth.AccessClaims, error) {
	op := "Auth.IsTokenValid"

	claims, err := a.validator.Validate(token)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	revoked, err := a.blackListRepository.Exists(ctx, claims.TokenID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if revoked {
		return nil, fmt.Errorf("%s: %w", op, errors.New("token is revoked"))
	}

	return claims, nil
}

func (a *AuthService) GetUsersByCompanyId(ctx context.Context, companyId uuid.UUID) ([]*entities.User, error) {
	op := "Auth.GetUsersByCompanyId"

	users, err := a.userRepository.GetByCompanyId(ctx, companyId)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return users, nil
}

func (a *AuthService) GetUserById(ctx context.Context, userId uuid.UUID) (*entities.User, error) {
	op := "Auth.GetUserById"

	user, err := a.userRepository.GetById(ctx, userId)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return user, nil
}
