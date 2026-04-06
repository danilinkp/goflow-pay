package services_test

import (
	"context"
	"errors"
	"notifications/internal/domain/entities"
	"notifications/internal/dto/response"
	"notifications/internal/services"
	mocks "notifications/internal/services/mocks"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupNotificationService(t *testing.T) (
	*services.NotificationService,
	*mocks.MockAccountClient,
	*mocks.MockUserClient,
	*mocks.MockNotificationRepository,
	*mocks.MockEmailSender,
	*mocks.MockCacheRepository,
) {
	accountClient := mocks.NewMockAccountClient(t)
	userClient := mocks.NewMockUserClient(t)
	notificationRepo := mocks.NewMockNotificationRepository(t)
	emailSender := mocks.NewMockEmailSender(t)
	cache := mocks.NewMockCacheRepository(t)

	svc := services.NewNotificationService(
		accountClient, userClient, notificationRepo, emailSender, cache,
	)
	return svc, accountClient, userClient, notificationRepo, emailSender, cache
}

func validAccountID() uuid.UUID { return uuid.New() }
func validCompanyID() uuid.UUID { return uuid.New() }
func mockUser(id uuid.UUID, email string) *response.UserResponse {
	return &response.UserResponse{ID: id, Email: email}
}

func TestNotificationService_NotifyTransferCompleted(t *testing.T) {
	ctx := context.Background()
	accountID := validAccountID()
	companyID := validCompanyID()
	amount := int64(1500)
	currency := "USD"

	expectedTitle := "Перевод на 1500 USD выполнен"
	expectedMessage := "Средства успешно переведены"

	t.Run("Success with cache hit", func(t *testing.T) {
		svc, _, userClient, notificationRepo, emailSender, cache := setupNotificationService(t)
		user1, user2 := mockUser(uuid.New(), "user1@test.com"), mockUser(uuid.New(), "user2@test.com")
		users := []*response.UserResponse{user1, user2}

		cache.On("GetCompanyId", mock.Anything, accountID).Return(companyID, nil)
		userClient.On("GetUsersByCompanyId", mock.Anything, companyID).Return(users, nil)

		for _, user := range users {
			notificationRepo.On("Save", mock.Anything, mock.MatchedBy(func(n *entities.Notification) bool {
				return n != nil &&
					n.UserID() == user.ID &&
					n.Title() == expectedTitle &&
					n.Message() == expectedMessage
			})).Return(nil)

			emailSender.On("Send", mock.Anything, user.Email, expectedTitle, expectedMessage).Return(nil)
		}

		err := svc.NotifyTransferCompleted(ctx, accountID, amount, currency)
		assert.NoError(t, err)
	})

	t.Run("Success with cache miss", func(t *testing.T) {
		svc, accountClient, userClient, notificationRepo, emailSender, cache := setupNotificationService(t)
		user := mockUser(uuid.New(), "user@test.com")

		cache.On("GetCompanyId", mock.Anything, accountID).Return(uuid.Nil, errors.New("miss"))
		accountClient.On("GetCompanyIdByAccountId", mock.Anything, accountID).Return(companyID, nil)
		cache.On("SetCompanyId", mock.Anything, accountID, companyID).Return(nil)
		userClient.On("GetUsersByCompanyId", mock.Anything, companyID).Return([]*response.UserResponse{user}, nil)
		notificationRepo.On("Save", mock.Anything, mock.Anything).Return(nil)
		emailSender.On("Send", mock.Anything, user.Email, expectedTitle, expectedMessage).Return(nil)

		err := svc.NotifyTransferCompleted(ctx, accountID, amount, currency)
		assert.NoError(t, err)
		cache.AssertCalled(t, "SetCompanyId", mock.Anything, accountID, companyID)
	})

	t.Run("Error from accountClient", func(t *testing.T) {
		svc, accountClient, _, _, _, cache := setupNotificationService(t)
		cache.On("GetCompanyId", mock.Anything, accountID).Return(uuid.Nil, errors.New("miss"))
		accountClient.On("GetCompanyIdByAccountId", mock.Anything, accountID).Return(uuid.Nil, errors.New("not found"))

		err := svc.NotifyTransferCompleted(ctx, accountID, amount, currency)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("Error from userClient", func(t *testing.T) {
		svc, _, userClient, _, _, cache := setupNotificationService(t)
		cache.On("GetCompanyId", mock.Anything, accountID).Return(companyID, nil)
		userClient.On("GetUsersByCompanyId", mock.Anything, companyID).Return(nil, errors.New("fetch failed"))

		err := svc.NotifyTransferCompleted(ctx, accountID, amount, currency)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "fetch failed")
	})

	t.Run("Error saving notification", func(t *testing.T) {
		svc, _, userClient, notificationRepo, _, cache := setupNotificationService(t)
		user := mockUser(uuid.New(), "user@test.com")

		cache.On("GetCompanyId", mock.Anything, accountID).Return(companyID, nil)
		userClient.On("GetUsersByCompanyId", mock.Anything, companyID).Return([]*response.UserResponse{user}, nil)
		notificationRepo.On("Save", mock.Anything, mock.Anything).Return(errors.New("db error"))

		err := svc.NotifyTransferCompleted(ctx, accountID, amount, currency)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "db error")
	})

	t.Run("Error sending email", func(t *testing.T) {
		svc, _, userClient, notificationRepo, emailSender, cache := setupNotificationService(t)
		user := mockUser(uuid.New(), "user@test.com")

		cache.On("GetCompanyId", mock.Anything, accountID).Return(companyID, nil)
		userClient.On("GetUsersByCompanyId", mock.Anything, companyID).Return([]*response.UserResponse{user}, nil)
		notificationRepo.On("Save", mock.Anything, mock.Anything).Return(nil)
		emailSender.On("Send", mock.Anything, user.Email, expectedTitle, expectedMessage).Return(errors.New("smtp error"))

		err := svc.NotifyTransferCompleted(ctx, accountID, amount, currency)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "smtp error")
	})

	t.Run("Empty users list", func(t *testing.T) {
		svc, _, userClient, _, _, cache := setupNotificationService(t)
		cache.On("GetCompanyId", mock.Anything, accountID).Return(companyID, nil)
		userClient.On("GetUsersByCompanyId", mock.Anything, companyID).Return([]*response.UserResponse{}, nil)

		err := svc.NotifyTransferCompleted(ctx, accountID, amount, currency)
		assert.NoError(t, err)
	})
}

func TestNotificationService_NotifyTransferFailed(t *testing.T) {
	ctx := context.Background()
	accountID := validAccountID()
	companyID := validCompanyID()
	amount := int64(2000)
	currency := "EUR"

	expectedTitle := "Перевод на 2000 EUR не выполнен"
	expectedMessage := "Произошла ошибка при переводе средств"

	t.Run("Success", func(t *testing.T) {
		svc, _, userClient, notificationRepo, emailSender, cache := setupNotificationService(t)
		user := mockUser(uuid.New(), "admin@test.com")

		cache.On("GetCompanyId", mock.Anything, accountID).Return(companyID, nil)
		userClient.On("GetUsersByCompanyId", mock.Anything, companyID).Return([]*response.UserResponse{user}, nil)
		notificationRepo.On("Save", mock.Anything, mock.MatchedBy(func(n *entities.Notification) bool {
			return n != nil && n.Title() == expectedTitle && n.Message() == expectedMessage
		})).Return(nil)
		emailSender.On("Send", mock.Anything, user.Email, expectedTitle, expectedMessage).Return(nil)

		err := svc.NotifyTransferFailed(ctx, accountID, amount, currency)
		assert.NoError(t, err)
	})
}

func TestNotificationService_BankOperations(t *testing.T) {
	ctx := context.Background()
	accountID := validAccountID()
	companyID := validCompanyID()

	tests := []struct {
		name            string
		amount          int64
		currency        string
		notifyFn        func(*services.NotificationService) error
		expectedTitle   string
		expectedMessage string
	}{
		{
			name:   "NotifyBankDepositCompleted",
			amount: 5000, currency: "RUB",
			expectedTitle:   "Пополнение с банковского счёта на сумму 5000 RUB",
			expectedMessage: "Средства успешно пополнены",
			notifyFn: func(s *services.NotificationService) error {
				return s.NotifyBankDepositCompleted(ctx, accountID, 5000, "RUB")
			},
		},
		{
			name:   "NotifyBankDepositFailed",
			amount: 5000, currency: "RUB",
			expectedTitle:   "Пополнение с банковского счёта на сумму 5000 RUB",
			expectedMessage: "Произошла ошибка при пополнении средств",
			notifyFn: func(s *services.NotificationService) error {
				return s.NotifyBankDepositFailed(ctx, accountID, 5000, "RUB")
			},
		},
		{
			name:   "NotifyBankWithdrawalCompleted",
			amount: 5000, currency: "RUB",
			expectedTitle:   "Вывод на банковский счёт на сумму 5000 RUB",
			expectedMessage: "Средства успешно выведены",
			notifyFn: func(s *services.NotificationService) error {
				return s.NotifyBankWithdrawalCompleted(ctx, accountID, 5000, "RUB")
			},
		},
		{
			name:   "NotifyBankWithdrawalFailed",
			amount: 5000, currency: "RUB",
			expectedTitle:   "Вывод на банковский счёт на сумму 5000 RUB",
			expectedMessage: "Произошла ошибка при выводе средств",
			notifyFn: func(s *services.NotificationService) error {
				return s.NotifyBankWithdrawalFailed(ctx, accountID, 5000, "RUB")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _, userClient, notificationRepo, emailSender, cache := setupNotificationService(t)
			user := mockUser(uuid.New(), "finance@test.com")

			cache.On("GetCompanyId", mock.Anything, accountID).Return(companyID, nil)
			userClient.On("GetUsersByCompanyId", mock.Anything, companyID).Return([]*response.UserResponse{user}, nil)
			notificationRepo.On("Save", mock.Anything, mock.MatchedBy(func(n *entities.Notification) bool {
				return n != nil && n.Title() == tt.expectedTitle && n.Message() == tt.expectedMessage
			})).Return(nil)
			emailSender.On("Send", mock.Anything, user.Email, tt.expectedTitle, tt.expectedMessage).Return(nil)

			err := tt.notifyFn(svc)
			assert.NoError(t, err)
		})
	}
}

func TestNotificationService_CacheLogic(t *testing.T) {
	ctx := context.Background()
	accountID := validAccountID()
	companyID := validCompanyID()

	t.Run("Cache hit - accountClient not called", func(t *testing.T) {
		svc, accountClient, userClient, _, _, cache := setupNotificationService(t)
		cache.On("GetCompanyId", mock.Anything, accountID).Return(companyID, nil)
		userClient.On("GetUsersByCompanyId", mock.Anything, companyID).Return([]*response.UserResponse{}, nil)

		_ = svc.NotifyTransferCompleted(ctx, accountID, 100, "USD")
		accountClient.AssertNotCalled(t, "GetCompanyIdByAccountId")
		cache.AssertNotCalled(t, "SetCompanyId")
	})

	t.Run("Cache miss - fetch and cache", func(t *testing.T) {
		svc, accountClient, userClient, _, _, cache := setupNotificationService(t)
		cache.On("GetCompanyId", mock.Anything, accountID).Return(uuid.Nil, errors.New("miss"))
		accountClient.On("GetCompanyIdByAccountId", mock.Anything, accountID).Return(companyID, nil)
		cache.On("SetCompanyId", mock.Anything, accountID, companyID).Return(nil)
		userClient.On("GetUsersByCompanyId", mock.Anything, companyID).Return([]*response.UserResponse{}, nil)

		_ = svc.NotifyTransferCompleted(ctx, accountID, 100, "USD")
		accountClient.AssertCalled(t, "GetCompanyIdByAccountId", mock.Anything, accountID)
		cache.AssertCalled(t, "SetCompanyId", mock.Anything, accountID, companyID)
	})

	t.Run("SetCompanyId error is ignored", func(t *testing.T) {
		svc, accountClient, userClient, _, _, cache := setupNotificationService(t)
		cache.On("GetCompanyId", mock.Anything, accountID).Return(uuid.Nil, errors.New("miss"))
		accountClient.On("GetCompanyIdByAccountId", mock.Anything, accountID).Return(companyID, nil)
		cache.On("SetCompanyId", mock.Anything, accountID, companyID).Return(errors.New("write failed"))
		userClient.On("GetUsersByCompanyId", mock.Anything, companyID).Return([]*response.UserResponse{}, nil)

		err := svc.NotifyTransferCompleted(ctx, accountID, 100, "USD")
		assert.NoError(t, err)
	})
}
