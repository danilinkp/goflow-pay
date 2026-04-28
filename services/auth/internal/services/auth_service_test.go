package services_test

import (
	"auth/internal/domain"
	"auth/internal/domain/entities"
	"auth/internal/services"
	mocks "auth/internal/services/mocks"
	"context"
	"errors"
	"io"
	"log/slog"
	"shared/pkg/auth"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupAuthService(t *testing.T, tokenTTL time.Duration) (
	*services.AuthService,
	*mocks.MockUserRepository,
	*mocks.MockCompanyRepository,
	*mocks.MockTransactor,
	*mocks.MockBlackListRepository,
	*mocks.MockTokenValidator,
	*mocks.MockTokenService,
	*mocks.MockPasswordHasher,
) {
	userRepo := mocks.NewMockUserRepository(t)
	companyRepo := mocks.NewMockCompanyRepository(t)
	transactor := mocks.NewMockTransactor(t)
	blackListRepo := mocks.NewMockBlackListRepository(t)
	tokenValidator := mocks.NewMockTokenValidator(t)
	tokenService := mocks.NewMockTokenService(t)
	hasher := mocks.NewMockPasswordHasher(t)

	discardLogger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	svc := services.NewAuthService(
		userRepo, companyRepo, transactor, blackListRepo,
		tokenService, tokenValidator, hasher, discardLogger,
	)
	return svc, userRepo, companyRepo, transactor, blackListRepo, tokenValidator, tokenService, hasher
}

const defaultTokenTTL = 24 * time.Hour

func validRegisterEmployeeReq() services.RegisterEmployeeInput {
	return services.RegisterEmployeeInput{
		Login:             "testuser",
		Email:             "test@example.com",
		Password:          "SecurePass123!",
		CompanyInviteCode: "",
	}
}

func validRegisterCompanyReq() services.RegisterCompanyInput {
	return services.RegisterCompanyInput{
		Login:       "testuser",
		Email:       "test@example.com",
		Password:    "SecurePass123!",
		CompanyName: "Test Company",
	}
}

func validLoginReq(email, password string) services.LoginInput {
	return services.LoginInput{
		Email:    email,
		Password: password,
	}
}

func TestAuthService_RegisterWithNewCompany(t *testing.T) {
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		svc, userRepo, companyRepo, transactor, _, _, tokenService, hasher := setupAuthService(t, defaultTokenTTL)
		companyReq := validRegisterCompanyReq()

		hasher.On("Hash", companyReq.Password).Return("hashed_password", nil)

		expectedLogin := companyReq.Login
		expectedEmail := companyReq.Email

		companyRepo.On("Save", mock.Anything, mock.MatchedBy(func(c *entities.Company) bool {
			return c != nil && c.Name() == companyReq.CompanyName && c.InviteCode() != ""
		})).Return(nil)

		userRepo.On("Save", mock.Anything, mock.MatchedBy(func(u *entities.User) bool {
			return u != nil &&
				u.Login() == expectedLogin &&
				u.Email() == expectedEmail &&
				u.Role() == entities.RoleCompanyAdmin &&
				u.PasswordHash() == "hashed_password"
		})).Return(nil)

		transactor.On("WithTx", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context) error)
			_ = fn(args.Get(0).(context.Context))
		})

		tokenService.On("Generate", mock.Anything, mock.Anything, string(entities.RoleCompanyAdmin)).Return("access_token", nil)

		resp, err := svc.RegisterWithNewCompany(ctx, companyReq)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, expectedEmail, resp.Email)
		assert.Equal(t, string(entities.RoleCompanyAdmin), resp.Role)
		assert.Equal(t, "access_token", resp.Token)
	})

	t.Run("Password hash error", func(t *testing.T) {
		svc, _, _, _, _, _, _, hasher := setupAuthService(t, defaultTokenTTL)
		companyReq := validRegisterCompanyReq()

		hasher.On("Hash", companyReq.Password).Return("", errors.New("hash failed"))

		resp, err := svc.RegisterWithNewCompany(ctx, companyReq)
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "hash failed")
	})

	t.Run("Company save error in transaction", func(t *testing.T) {
		svc, userRepo, companyRepo, transactor, _, _, tokenService, hasher := setupAuthService(t, defaultTokenTTL)
		companyReq := validRegisterCompanyReq()

		hasher.On("Hash", companyReq.Password).Return("hashed", nil)

		companyRepo.On("Save", mock.Anything, mock.Anything).Return(errors.New("db error"))

		transactor.On("WithTx", mock.Anything, mock.Anything).Return(errors.New("db error")).Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context) error)
			_ = fn(args.Get(0).(context.Context))
		})

		resp, err := svc.RegisterWithNewCompany(ctx, companyReq)
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "db error")

		tokenService.AssertNotCalled(t, "Generate")
		userRepo.AssertNotCalled(t, "Save")
	})

	t.Run("Token generation error", func(t *testing.T) {
		svc, userRepo, companyRepo, transactor, _, _, tokenService, hasher := setupAuthService(t, defaultTokenTTL)
		companyReq := validRegisterCompanyReq()

		hasher.On("Hash", companyReq.Password).Return("hashed", nil)
		companyRepo.On("Save", mock.Anything, mock.MatchedBy(func(c *entities.Company) bool {
			return c != nil && c.Name() == companyReq.CompanyName
		})).Return(nil)
		userRepo.On("Save", mock.Anything, mock.MatchedBy(func(u *entities.User) bool {
			return u != nil && u.Login() == companyReq.Login
		})).Return(nil)
		transactor.On("WithTx", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context) error)
			_ = fn(args.Get(0).(context.Context))
		})
		tokenService.On("Generate", mock.Anything, mock.Anything, string(entities.RoleCompanyAdmin)).Return("", errors.New("token error"))

		resp, err := svc.RegisterWithNewCompany(ctx, companyReq)
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "token error")
	})
}

func TestAuthService_RegisterWithExistingCompany(t *testing.T) {
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		svc, userRepo, companyRepo, _, _, _, tokenService, hasher := setupAuthService(t, defaultTokenTTL)
		userReq := validRegisterEmployeeReq()

		company, err := entities.NewCompany("Test Corp")
		assert.NoError(t, err)

		actualInviteCode := company.InviteCode()

		userReq.CompanyInviteCode = actualInviteCode

		companyRepo.On("GetByInviteCode", mock.Anything, actualInviteCode).Return(company, nil)

		hasher.On("Hash", userReq.Password).Return("hashed", nil)
		userRepo.On("Save", mock.Anything, mock.MatchedBy(func(u *entities.User) bool {
			return u != nil && u.Role() == entities.RoleEmployee
		})).Return(nil)
		tokenService.On("Generate", mock.Anything, mock.Anything, string(entities.RoleEmployee)).Return("token", nil)

		resp, err := svc.RegisterWithExistingCompany(ctx, userReq)

		assert.NoError(t, err)
		if err == nil {
			assert.NotNil(t, resp)
			assert.Equal(t, string(entities.RoleEmployee), resp.Role)
		}
	})

	t.Run("Invite code not found", func(t *testing.T) {
		svc, _, companyRepo, _, _, _, _, _ := setupAuthService(t, defaultTokenTTL)
		userReq := validRegisterEmployeeReq()
		userReq.CompanyInviteCode = "INVALID"

		companyRepo.On("GetByInviteCode", mock.Anything, userReq.CompanyInviteCode).Return(nil, domain.ErrCompanyNotFound)

		resp, err := svc.RegisterWithExistingCompany(ctx, userReq)
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), domain.ErrCompanyNotFound.Error())
	})

	t.Run("Invite code mismatch", func(t *testing.T) {
		svc, _, companyRepo, _, _, _, _, _ := setupAuthService(t, defaultTokenTTL)
		userReq := validRegisterEmployeeReq()
		userReq.CompanyInviteCode = "ABC12345"

		otherCompany, err := entities.NewCompany("Other Corp")
		assert.NoError(t, err)

		companyRepo.On("GetByInviteCode", mock.Anything, userReq.CompanyInviteCode).Return(otherCompany, nil)

		resp, err := svc.RegisterWithExistingCompany(ctx, userReq)
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.ErrorIs(t, err, services.ErrInvalidInviteCode)
	})

	t.Run("User save error", func(t *testing.T) {
		svc, userRepo, companyRepo, _, _, _, tokenService, hasher := setupAuthService(t, defaultTokenTTL)
		userReq := validRegisterEmployeeReq()

		company, err := entities.NewCompany("Test Corp")
		assert.NoError(t, err)
		actualInviteCode := company.InviteCode()

		userReq.CompanyInviteCode = actualInviteCode

		companyRepo.On("GetByInviteCode", mock.Anything, actualInviteCode).Return(company, nil)
		hasher.On("Hash", userReq.Password).Return("hashed", nil)
		userRepo.On("Save", mock.Anything, mock.Anything).Return(errors.New("db error"))

		resp, err := svc.RegisterWithExistingCompany(ctx, userReq)
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "db error")

		tokenService.AssertNotCalled(t, "Generate")
	})
}

func TestAuthService_Login(t *testing.T) {
	ctx := context.Background()
	email := "user@example.com"
	password := "SecurePass123!"

	t.Run("Success", func(t *testing.T) {
		svc, userRepo, _, _, _, _, tokenService, hasher := setupAuthService(t, defaultTokenTTL)
		email := "user@example.com"
		password := "SecurePass123!"

		companyID := uuid.New()
		user, err := entities.NewUser(companyID, "login", email, "hashed", entities.RoleEmployee)
		assert.NoError(t, err)

		userID := user.UserId()

		userRepo.On("GetByEmail", mock.Anything, email).Return(user, nil)
		hasher.On("Verify", password, user.PasswordHash()).Return(nil)

		tokenService.On("Generate", userID, companyID, string(entities.RoleEmployee)).Return("access_token", nil)

		resp, err := svc.Login(ctx, validLoginReq(email, password))
		assert.NoError(t, err)
		if assert.NotNil(t, resp) {
			assert.Equal(t, "access_token", resp.Token)
		}
	})

	t.Run("User not found", func(t *testing.T) {
		svc, userRepo, _, _, _, _, _, _ := setupAuthService(t, defaultTokenTTL)

		userRepo.On("GetByEmail", mock.Anything, email).Return(nil, domain.ErrUserNotFound)

		resp, err := svc.Login(ctx, validLoginReq(email, password))
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.ErrorIs(t, err, services.ErrInvalidCredentials)
	})

	t.Run("Wrong password", func(t *testing.T) {
		svc, userRepo, _, _, _, _, _, hasher := setupAuthService(t, defaultTokenTTL)

		user, err := entities.NewUser(uuid.New(), "login", email, "hashed", entities.RoleEmployee)
		assert.NoError(t, err)

		userRepo.On("GetByEmail", mock.Anything, email).Return(user, nil)
		hasher.On("Verify", password, user.PasswordHash()).Return(errors.New("password mismatch"))

		resp, err := svc.Login(ctx, validLoginReq(email, password))
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.ErrorIs(t, err, services.ErrInvalidCredentials)
	})

	t.Run("Token generation error", func(t *testing.T) {
		svc, userRepo, _, _, _, _, tokenService, hasher := setupAuthService(t, defaultTokenTTL)

		user, err := entities.NewUser(uuid.New(), "login", email, "hashed", entities.RoleEmployee)
		assert.NoError(t, err)

		userRepo.On("GetByEmail", mock.Anything, email).Return(user, nil)
		hasher.On("Verify", password, user.PasswordHash()).Return(nil)
		tokenService.On("Generate", mock.Anything, mock.Anything, mock.Anything).Return("", errors.New("token error"))

		resp, err := svc.Login(ctx, validLoginReq(email, password))
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "token error")
	})

	t.Run("Repository error (not ErrUserNotFound)", func(t *testing.T) {
		svc, userRepo, _, _, _, _, _, _ := setupAuthService(t, defaultTokenTTL)

		userRepo.On("GetByEmail", mock.Anything, email).Return(nil, errors.New("db connection failed"))

		resp, err := svc.Login(ctx, validLoginReq(email, password))
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "db connection failed")
	})
}

func TestAuthService_Logout(t *testing.T) {
	ctx := context.Background()
	token := "valid_access_token"
	tokenID := uuid.New().String()

	makeClaims := func(expiresAt time.Time) *auth.AccessClaims {
		return &auth.AccessClaims{
			TokenID:   tokenID,
			UserID:    uuid.New(),
			CompanyID: uuid.New(),
			Role:      "employee",
			ExpiresAt: expiresAt,
		}
	}

	t.Run("Success - token with remaining TTL", func(t *testing.T) {
		svc, _, _, _, blackListRepo, tokenValidator, _, _ := setupAuthService(t, defaultTokenTTL)

		claims := makeClaims(time.Now().Add(1 * time.Hour))
		tokenValidator.On("Validate", token).Return(claims, nil)
		blackListRepo.On("Save", mock.Anything, tokenID, mock.MatchedBy(func(d time.Duration) bool {
			return d > 0 && d <= defaultTokenTTL
		})).Return(nil)

		err := svc.Logout(ctx, token)
		assert.NoError(t, err)
	})

	t.Run("Token already expired - no blacklist save", func(t *testing.T) {
		svc, _, _, _, blackListRepo, tokenValidator, _, _ := setupAuthService(t, defaultTokenTTL)

		claims := makeClaims(time.Now().Add(-1 * time.Hour))
		tokenValidator.On("Validate", token).Return(claims, nil)

		err := svc.Logout(ctx, token)
		assert.NoError(t, err)
		blackListRepo.AssertNotCalled(t, "Save")
	})

	t.Run("Invalid token", func(t *testing.T) {
		svc, _, _, _, _, tokenValidator, _, _ := setupAuthService(t, defaultTokenTTL)

		tokenValidator.On("Validate", token).Return(nil, errors.New("invalid signature"))

		err := svc.Logout(ctx, token)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid signature")
	})

	t.Run("Blacklist save error", func(t *testing.T) {
		svc, _, _, _, blackListRepo, tokenValidator, _, _ := setupAuthService(t, defaultTokenTTL)

		claims := makeClaims(time.Now().Add(30 * time.Minute))
		tokenValidator.On("Validate", token).Return(claims, nil)
		blackListRepo.On("Save", mock.Anything, tokenID, mock.Anything).Return(errors.New("redis error"))

		err := svc.Logout(ctx, token)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "redis error")
	})
}

func TestAuthService_GetUsersByCompanyId(t *testing.T) {
	ctx := context.Background()
	companyID := uuid.New()

	t.Run("Success with multiple users", func(t *testing.T) {
		svc, userRepo, _, _, _, _, _, _ := setupAuthService(t, defaultTokenTTL)

		user1, _ := entities.NewUser(companyID, "u1", "u1@test.com", "h1", entities.RoleEmployee)
		user2, _ := entities.NewUser(companyID, "u2", "u2@test.com", "h2", entities.RoleAdmin)
		users := []*entities.User{user1, user2}

		userRepo.On("GetByCompanyId", mock.Anything, companyID).Return(users, nil)

		result, err := svc.GetUsersByCompanyId(ctx, companyID)
		assert.NoError(t, err)
		assert.Len(t, result, 2)
		for _, u := range result {
			assert.NotEmpty(t, u.UserId())
			assert.NotEmpty(t, u.Email)
		}
	})

	t.Run("Empty result", func(t *testing.T) {
		svc, userRepo, _, _, _, _, _, _ := setupAuthService(t, defaultTokenTTL)

		userRepo.On("GetByCompanyId", mock.Anything, companyID).Return([]*entities.User{}, nil)

		result, err := svc.GetUsersByCompanyId(ctx, companyID)
		assert.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("Repository error", func(t *testing.T) {
		svc, userRepo, _, _, _, _, _, _ := setupAuthService(t, defaultTokenTTL)

		userRepo.On("GetByCompanyId", mock.Anything, companyID).Return(nil, errors.New("db error"))

		result, err := svc.GetUsersByCompanyId(ctx, companyID)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "db error")
	})
}
