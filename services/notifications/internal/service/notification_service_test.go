package service_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"notifications/internal/domain/entities"
	"notifications/internal/service"
	mocks "notifications/internal/service/mocks"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupNotificationService(t *testing.T) (
	*service.NotificationService,
	*mocks.MockUserClient,
	*mocks.MockNotificationRepository,
	*mocks.MockEmailSender,
) {
	userClient := mocks.NewMockUserClient(t)
	notificationRepo := mocks.NewMockNotificationRepository(t)
	emailSender := mocks.NewMockEmailSender(t)
	discardLogger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	svc := service.NewNotificationService(
		userClient, notificationRepo, emailSender, discardLogger,
	)
	return svc, userClient, notificationRepo, emailSender
}

func validUserID() uuid.UUID    { return uuid.New() }
func validAccountID() uuid.UUID { return uuid.New() }

func mockUser(id uuid.UUID, email string) *service.UserResponse {
	return &service.UserResponse{ID: id, Email: email}
}

func TestNotificationService_NotifyTransferCompleted(t *testing.T) {
	ctx := context.Background()
	userID := validUserID()
	accountID := validAccountID()
	amount := int64(1500)
	currency := "USD"

	expectedTitle := "Перевод на 1500 USD выполнен"
	expectedMessage := "Средства успешно переведены"

	t.Run("Success", func(t *testing.T) {
		svc, userClient, notificationRepo, emailSender := setupNotificationService(t)
		user := mockUser(userID, "user@test.com")

		userClient.On("GetUserById", mock.Anything, userID).Return(user, nil)

		notificationRepo.On("Save", mock.Anything, mock.MatchedBy(func(n *entities.Notification) bool {
			return n != nil &&
				n.UserId() == userID &&
				n.Title() == expectedTitle &&
				n.Message() == expectedMessage &&
				n.SourceId() == accountID
		})).Return(nil)

		emailSender.On("Send", mock.Anything, user.Email, expectedTitle, expectedMessage).Return(nil)

		// Исправлено: передаём userID и accountId
		err := svc.NotifyTransferCompleted(ctx, userID, accountID, amount, currency)
		assert.NoError(t, err)
	})

	t.Run("Error from userClient", func(t *testing.T) {
		svc, userClient, _, _ := setupNotificationService(t)
		userClient.On("GetUserById", mock.Anything, userID).Return(nil, errors.New("user not found"))

		err := svc.NotifyTransferCompleted(ctx, userID, accountID, amount, currency)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("Error saving notification", func(t *testing.T) {
		svc, userClient, notificationRepo, _ := setupNotificationService(t)
		user := mockUser(userID, "user@test.com")

		userClient.On("GetUserById", mock.Anything, userID).Return(user, nil)
		notificationRepo.On("Save", mock.Anything, mock.Anything).Return(errors.New("db error"))

		err := svc.NotifyTransferCompleted(ctx, userID, accountID, amount, currency)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "db error")
	})

	t.Run("Error sending email", func(t *testing.T) {
		svc, userClient, notificationRepo, emailSender := setupNotificationService(t)
		user := mockUser(userID, "user@test.com")

		userClient.On("GetUserById", mock.Anything, userID).Return(user, nil)
		notificationRepo.On("Save", mock.Anything, mock.Anything).Return(nil)
		emailSender.On("Send", mock.Anything, user.Email, expectedTitle, expectedMessage).Return(errors.New("smtp error"))

		err := svc.NotifyTransferCompleted(ctx, userID, accountID, amount, currency)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "smtp error")
	})
}

func TestNotificationService_NotifyTransferFailed(t *testing.T) {
	ctx := context.Background()
	userID := validUserID()
	accountID := validAccountID()
	amount := int64(2000)
	currency := "EUR"

	expectedTitle := "Перевод на 2000 EUR не выполнен"
	expectedMessage := "Произошла ошибка при переводе средств"

	t.Run("Success", func(t *testing.T) {
		svc, userClient, notificationRepo, emailSender := setupNotificationService(t)
		user := mockUser(userID, "admin@test.com")

		userClient.On("GetUserById", mock.Anything, userID).Return(user, nil)
		notificationRepo.On("Save", mock.Anything, mock.MatchedBy(func(n *entities.Notification) bool {
			return n != nil && n.Title() == expectedTitle && n.Message() == expectedMessage
		})).Return(nil)
		emailSender.On("Send", mock.Anything, user.Email, expectedTitle, expectedMessage).Return(nil)

		err := svc.NotifyTransferFailed(ctx, userID, accountID, amount, currency)
		assert.NoError(t, err)
	})
}

func TestNotificationService_BankOperations(t *testing.T) {
	ctx := context.Background()
	userID := validUserID()
	accountID := validAccountID()

	tests := []struct {
		name            string
		amount          int64
		currency        string
		notifyFn        func(*service.NotificationService) error
		expectedTitle   string
		expectedMessage string
	}{
		{
			name:   "NotifyBankDepositCompleted",
			amount: 5000, currency: "RUB",
			expectedTitle:   "Пополнение с банковского счёта на сумму 5000 RUB",
			expectedMessage: "Средства успешно пополнены",
			notifyFn: func(s *service.NotificationService) error {
				return s.NotifyBankDepositCompleted(ctx, userID, accountID, 5000, "RUB")
			},
		},
		{
			name:   "NotifyBankDepositFailed",
			amount: 5000, currency: "RUB",
			expectedTitle:   "Пополнение с банковского счёта на сумму 5000 RUB",
			expectedMessage: "Произошла ошибка при пополнении средств",
			notifyFn: func(s *service.NotificationService) error {
				return s.NotifyBankDepositFailed(ctx, userID, accountID, 5000, "RUB")
			},
		},
		{
			name:   "NotifyBankWithdrawalCompleted",
			amount: 5000, currency: "RUB",
			expectedTitle:   "Вывод на банковский счёт на сумму 5000 RUB",
			expectedMessage: "Средства успешно выведены",
			notifyFn: func(s *service.NotificationService) error {
				return s.NotifyBankWithdrawalCompleted(ctx, userID, accountID, 5000, "RUB")
			},
		},
		{
			name:   "NotifyBankWithdrawalFailed",
			amount: 5000, currency: "RUB",
			expectedTitle:   "Вывод на банковский счёт на сумму 5000 RUB",
			expectedMessage: "Произошла ошибка при выводе средств",
			notifyFn: func(s *service.NotificationService) error {
				return s.NotifyBankWithdrawalFailed(ctx, userID, accountID, 5000, "RUB")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, userClient, notificationRepo, emailSender := setupNotificationService(t)
			user := mockUser(userID, "finance@test.com")

			userClient.On("GetUserById", mock.Anything, userID).Return(user, nil)
			notificationRepo.On("Save", mock.Anything, mock.MatchedBy(func(n *entities.Notification) bool {
				return n != nil && n.Title() == tt.expectedTitle && n.Message() == tt.expectedMessage
			})).Return(nil)
			emailSender.On("Send", mock.Anything, user.Email, tt.expectedTitle, tt.expectedMessage).Return(nil)

			err := tt.notifyFn(svc)
			assert.NoError(t, err)
		})
	}
}
