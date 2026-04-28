package services

import (
	"auth/internal/domain"
	"auth/internal/domain/entities"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"shared/pkg/auth"
	"shared/pkg/logger/sl"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidCredentials       = errors.New("invalid credentials")
	ErrInvalidInviteCode        = errors.New("invalid invite code")
	ErrSystemAlreadyInitialized = errors.New("system already initialized")
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
	GetInviteCodeById(ctx context.Context, companyId uuid.UUID) (string, error)
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
	op := "AuthService.RegisterWithNewCompany"
	start := time.Now()

	log := a.logger.With(
		sl.Op(op),
		sl.EventID(),
	)

	log.Info("registering new company",
		slog.String("email", in.Email),
		slog.String("company_name", in.CompanyName),
	)

	passwordHash, err := a.hasher.Hash(in.Password)
	if err != nil {
		log.Error("failed to hash password",
			sl.ErrWithStack(err),
			sl.Duration(time.Since(start)),
		)
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	company, err := entities.NewCompany(in.CompanyName)
	if err != nil {
		log.Error("failed to create company entity",
			sl.ErrWithStack(err),
			sl.Duration(time.Since(start)),
		)
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	user, err := entities.NewUser(company.CompanyId(), in.Login, in.Email, passwordHash, entities.RoleCompanyAdmin)
	if err != nil {
		log.Error("failed to create user entity",
			sl.ErrWithStack(err),
			sl.Duration(time.Since(start)),
		)
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	err = a.transactor.WithTx(ctx, func(txCtx context.Context) error {
		if err = a.companyRepository.Save(txCtx, company); err != nil {
			return err
		}
		return a.userRepository.Save(txCtx, user)
	})
	if err != nil {
		log.Error("failed to save company and user",
			sl.ErrWithStack(err),
			sl.Duration(time.Since(start)),
		)
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	token, err := a.tokenService.Generate(user.UserId(), user.CompanyId(), string(user.Role()))
	if err != nil {
		log.Error("failed to generate token",
			sl.ErrWithStack(err),
			sl.Duration(time.Since(start)),
		)
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("company registered successfully",
		slog.String("user_id", user.UserId().String()),
		slog.String("company_id", user.CompanyId().String()),
		slog.String("role", string(user.Role())),
		sl.Duration(time.Since(start)),
	)

	return &AuthOutput{
		UserID:    user.UserId(),
		CompanyID: user.CompanyId(),
		Email:     user.Email(),
		Role:      string(user.Role()),
		Token:     token,
	}, nil
}

func (a *AuthService) RegisterWithExistingCompany(ctx context.Context, userRequest RegisterEmployeeInput) (*AuthOutput, error) {
	op := "AuthService.RegisterWithExistingCompany"

	start := time.Now()

	log := a.logger.With(
		sl.Op(op),
		sl.EventID(),
	)

	log.Info("registering new user",
		slog.String("email", userRequest.Email),
		slog.String("login", userRequest.Login),
	)

	company, err := a.companyRepository.GetByInviteCode(ctx, userRequest.CompanyInviteCode)
	if err != nil {
		log.Error("failed to get company by invite code",
			sl.ErrWithStack(err),
			sl.Duration(time.Since(start)))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if company.InviteCode() != userRequest.CompanyInviteCode {
		log.Error("company invite code does not match",
			sl.ErrWithStack(ErrInvalidInviteCode),
			sl.Duration(time.Since(start)))
		return nil, fmt.Errorf("%s: %w", op, ErrInvalidInviteCode)
	}

	passwordHash, err := a.hasher.Hash(userRequest.Password)
	if err != nil {
		log.Error("failed to hash password",
			sl.ErrWithStack(err),
			sl.Duration(time.Since(start)))
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	user, err := entities.NewUser(company.CompanyId(), userRequest.Login, userRequest.Email, passwordHash, entities.RoleEmployee)
	if err != nil {
		log.Error("failed to create user entity",
			sl.ErrWithStack(err),
			sl.Duration(time.Since(start)))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	err = a.userRepository.Save(ctx, user)
	if err != nil {
		log.Error("failed to save user entity",
			sl.ErrWithStack(err),
			sl.Duration(time.Since(start)))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	token, err := a.tokenService.Generate(user.UserId(), user.CompanyId(), string(user.Role()))
	if err != nil {
		log.Error("failed to generate token",
			sl.ErrWithStack(err),
			sl.Duration(time.Since(start)))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("user registered successfully",
		slog.String("user_id", user.UserId().String()),
		slog.String("company_id", user.CompanyId().String()),
		slog.String("role", string(user.Role())),
		sl.Duration(time.Since(start)))

	return &AuthOutput{
		UserID:    user.UserId(),
		CompanyID: user.CompanyId(),
		Email:     user.Email(),
		Role:      string(user.Role()),
		Token:     token,
	}, nil
}

func (a *AuthService) Login(ctx context.Context, request LoginInput) (*AuthOutput, error) {
	op := "AuthService.Login"
	start := time.Now()

	log := a.logger.With(
		sl.Op(op),
		sl.EventID(),
		slog.String("email", request.Email),
	)

	log.Info("user login attempt")

	user, err := a.userRepository.GetByEmail(ctx, request.Email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			log.Warn("login failed: user not found",
				sl.Duration(time.Since(start)),
			)
			return nil, fmt.Errorf("%s: %w", op, ErrInvalidCredentials)
		}
		log.Error("failed to get user",
			sl.ErrWithStack(err),
			sl.Duration(time.Since(start)),
		)
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if err = a.hasher.Verify(request.Password, user.PasswordHash()); err != nil {
		log.Warn("login failed: invalid password",
			slog.String("user_id", user.UserId().String()),
			sl.Duration(time.Since(start)),
		)
		return nil, fmt.Errorf("%s: %w", op, ErrInvalidCredentials)
	}

	token, err := a.tokenService.Generate(user.UserId(), user.CompanyId(), string(user.Role()))
	if err != nil {
		log.Error("failed to generate token",
			sl.ErrWithStack(err),
			sl.Duration(time.Since(start)),
		)
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("user logged in successfully",
		slog.String("user_id", user.UserId().String()),
		slog.String("role", string(user.Role())),
		sl.Duration(time.Since(start)),
	)

	return &AuthOutput{
		UserID:    user.UserId(),
		CompanyID: user.CompanyId(),
		Email:     user.Email(),
		Role:      string(user.Role()),
		Token:     token,
	}, nil
}

func (a *AuthService) Logout(ctx context.Context, token string) error {
	op := "AuthService.Logout"
	start := time.Now()

	log := a.logger.With(
		sl.Op(op),
		sl.EventID(),
	)

	log.Info("user logout attempt")

	claims, err := a.validator.Validate(token)
	if err != nil {
		log.Warn("logout failed: invalid token",
			sl.ErrWithStack(err),
			sl.Duration(time.Since(start)),
		)
		return fmt.Errorf("%s: %w", op, err)
	}

	remainingTTL := time.Until(claims.ExpiresAt)
	if remainingTTL <= 0 {
		log.Info("token already expired, skipping blacklist",
			slog.String("user_id", claims.UserID.String()),
		)
		return nil
	}

	if err = a.blackListRepository.Save(ctx, claims.TokenID, remainingTTL); err != nil {
		log.Error("failed to blacklist token",
			sl.ErrWithStack(err),
			sl.Duration(time.Since(start)),
		)
		return fmt.Errorf("%s: %w", op, err)
	}

	log.Info("user logged out successfully",
		slog.String("user_id", claims.UserID.String()),
		sl.Duration(time.Since(start)),
	)

	return nil
}

func (a *AuthService) IsTokenValid(ctx context.Context, token string) (*auth.AccessClaims, error) {
	op := "Auth.IsTokenValid"

	start := time.Now()

	log := a.logger.With(
		sl.Op(op),
		sl.EventID(),
	)

	log.Info("user check attempt")

	claims, err := a.validator.Validate(token)
	if err != nil {
		log.Error("failed to validate token",
			sl.ErrWithStack(err),
			sl.Duration(time.Since(start)))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	revoked, err := a.blackListRepository.Exists(ctx, claims.TokenID)
	if err != nil {
		log.Error("failed to check if token is revoked",
			sl.ErrWithStack(err),
			sl.Duration(time.Since(start)))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if revoked {
		log.Warn("token is revoked, skipping blacklist",
			slog.String("user_id", claims.UserID.String()),
			sl.Duration(time.Since(start)))
		return nil, fmt.Errorf("%s: %w", op, errors.New("token is revoked"))
	}

	log.Info("token is valid",
		slog.String("user_id", claims.UserID.String()),
		sl.Duration(time.Since(start)))

	return claims, nil
}

func (a *AuthService) GetUsersByCompanyId(ctx context.Context, companyId uuid.UUID) ([]*entities.User, error) {
	op := "AuthService.GetUsersByCompanyId"

	start := time.Now()

	log := a.logger.With(
		sl.Op(op),
		sl.EventID(),
	)

	log.Info("user list attempt")

	users, err := a.userRepository.GetByCompanyId(ctx, companyId)
	if err != nil {
		log.Error("failed to get user by company id",
			sl.ErrWithStack(err),
			sl.Duration(time.Since(start)))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("user list successfully",
		sl.Duration(time.Since(start)))

	return users, nil
}

func (a *AuthService) GetUserById(ctx context.Context, userId uuid.UUID) (*entities.User, error) {
	op := "AuthService.GetUserById"

	start := time.Now()
	log := a.logger.With(
		sl.Op(op),
		sl.EventID(),
	)

	log.Info("user get attempt")

	user, err := a.userRepository.GetById(ctx, userId)
	if err != nil {
		log.Error("failed to get user by id",
			sl.ErrWithStack(err),
			sl.Duration(time.Since(start)))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("user get successfully",
		slog.String("user_id", user.UserId().String()),
		sl.Duration(time.Since(start)))

	return user, nil
}

func (a *AuthService) GetInviteCode(ctx context.Context, companyId uuid.UUID) (string, error) {
	op := "AuthService.GetInviteCode"

	start := time.Now()
	log := a.logger.With(
		sl.Op(op),
		sl.EventID(),
	)

	log.Info("invite code get attempt")

	inviteCode, err := a.companyRepository.GetInviteCodeById(ctx, companyId)
	if err != nil {
		log.Error("failed to get invite code by company id",
			sl.ErrWithStack(err),
			sl.Duration(time.Since(start)))
		return "", fmt.Errorf("%s: %w", op, err)
	}

	log.Info("invite code get successfully",
		slog.String("company_id", companyId.String()),
		sl.Duration(time.Since(start)))

	return inviteCode, nil
}

func (a *AuthService) InitSystem(ctx context.Context, req InitAdminInput) (*AuthOutput, error) {
	op := "AuthService.InitSystem"

	start := time.Now()
	log := a.logger.With(
		sl.Op(op),
		sl.EventID(),
	)
	log.Info("user init system attempt")

	_, err := a.companyRepository.GetById(ctx, auth.SystemCompanyID)
	if err != nil && !errors.Is(err, domain.ErrCompanyNotFound) {
		if errors.Is(err, domain.ErrCompanyAlreadyExists) {
			log.Warn("admin is already initialized, skipping",
				sl.Duration(time.Since(start)),
			)
			return nil, fmt.Errorf("%s: %w", op, ErrSystemAlreadyInitialized)
		}
		log.Error("failed to get user by company id",
			sl.ErrWithStack(err),
			sl.Duration(time.Since(start)),
		)
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	company, err := entities.NewCompanyWithID(auth.SystemCompanyID, "GoFlow Pay System")
	if err != nil {
		log.Error("failed to create company",
			sl.ErrWithStack(err),
			sl.Duration(time.Since(start)))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	hash, err := a.hasher.Hash(req.Password)
	if err != nil {
		log.Error("failed to hash password",
			sl.ErrWithStack(err),
			sl.Duration(time.Since(start)),
		)
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	user, err := entities.NewUser(auth.SystemCompanyID, req.Login, req.Email, hash, entities.RoleAdmin)
	if err != nil {
		log.Error("failed to create user",
			sl.ErrWithStack(err),
			sl.Duration(time.Since(start)),
		)
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	err = a.transactor.WithTx(ctx, func(ctx context.Context) error {
		if err = a.companyRepository.Save(ctx, company); err != nil {
			return err
		}
		return a.userRepository.Save(ctx, user)
	})
	if err != nil {
		log.Error("failed to create user",
			sl.ErrWithStack(err),
			sl.Duration(time.Since(start)))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	token, err := a.tokenService.Generate(user.UserId(), user.CompanyId(), string(user.Role()))
	if err != nil {
		log.Error("failed to generate token",
			sl.ErrWithStack(err),
			sl.Duration(time.Since(start)),
		)
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("system successfully initialized",
		sl.Duration(time.Since(start)))

	return &AuthOutput{
		UserID:    user.UserId(),
		CompanyID: user.CompanyId(),
		Email:     user.Email(),
		Role:      string(user.Role()),
		Token:     token,
	}, nil
}

func (a *AuthService) AddAdmin(ctx context.Context, req AddAdminInput) (*AuthOutput, error) {
	op := "AuthService.AddAdmin"

	start := time.Now()
	log := a.logger.With(
		sl.Op(op),
		sl.EventID(),
	)
	log.Info("add admin attempt")

	hash, err := a.hasher.Hash(req.Password)
	if err != nil {
		log.Error("failed to hash password",
			sl.ErrWithStack(err),
			sl.Duration(time.Since(start)),
		)
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	user, err := entities.NewUser(auth.SystemCompanyID, req.Login, req.Email, hash, entities.RoleAdmin)
	if err != nil {
		log.Error("failed to create user",
			sl.ErrWithStack(err),
			sl.Duration(time.Since(start)),
		)
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	err = a.userRepository.Save(ctx, user)
	if err != nil {
		log.Error("failed to save user",
			sl.ErrWithStack(err),
			sl.Duration(time.Since(start)),
		)
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	token, err := a.tokenService.Generate(user.UserId(), user.CompanyId(), string(user.Role()))
	if err != nil {
		log.Error("failed to generate token",
			sl.ErrWithStack(err),
			sl.Duration(time.Since(start)))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("admin successfully created",
		slog.String("user_id", user.UserId().String()),
		sl.Duration(time.Since(start)),
	)

	return &AuthOutput{
		UserID:    user.UserId(),
		CompanyID: user.CompanyId(),
		Email:     user.Email(),
		Role:      string(user.Role()),
		Token:     token,
	}, nil

}
