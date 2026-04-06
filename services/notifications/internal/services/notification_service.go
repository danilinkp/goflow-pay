package services

import (
	"context"
	"fmt"
	"notifications/internal/domain/entities"
	"notifications/internal/dto/response"

	"github.com/google/uuid"
)

type AccountClient interface {
	GetCompanyIdByAccountId(ctx context.Context, accountId uuid.UUID) (uuid.UUID, error)
}

type UserClient interface {
	GetUsersByCompanyId(ctx context.Context, companyId uuid.UUID) ([]*response.UserResponse, error)
}

type NotificationRepository interface {
	Save(ctx context.Context, operation *entities.Notification) error
	GetById(ctx context.Context, id uuid.UUID) (*entities.Notification, error)
}

type EmailSender interface {
	Send(ctx context.Context, email string, title string, message string) error
}

type CacheRepository interface {
	GetCompanyId(ctx context.Context, accountId uuid.UUID) (uuid.UUID, error)
	SetCompanyId(ctx context.Context, accountId uuid.UUID, companyId uuid.UUID) error
}

type NotificationService struct {
	accountClient          AccountClient
	userClient             UserClient
	notificationRepository NotificationRepository
	emailSender            EmailSender
	cache                  CacheRepository
}

func NewNotificationService(accountClient AccountClient, userClient UserClient, notificationRepository NotificationRepository, emailSender EmailSender, cache CacheRepository) *NotificationService {
	return &NotificationService{
		accountClient:          accountClient,
		userClient:             userClient,
		notificationRepository: notificationRepository,
		emailSender:            emailSender,
		cache:                  cache,
	}
}

func (n *NotificationService) NotifyTransferCompleted(ctx context.Context, accountId uuid.UUID, amount int64, currency string) error {
	op := "NotificationService.NotifyTransferCompleted"
	err := n.notify(ctx, accountId,
		fmt.Sprintf("Перевод на %d %s выполнен", amount, currency),
		"Средства успешно переведены",
	)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (n *NotificationService) NotifyTransferFailed(ctx context.Context, accountId uuid.UUID, amount int64, currency string) error {
	op := "NotificationService.NotifyTransferFailed"
	err := n.notify(ctx, accountId,
		fmt.Sprintf("Перевод на %d %s не выполнен", amount, currency),
		"Произошла ошибка при переводе средств",
	)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (n *NotificationService) NotifyBankDepositCompleted(ctx context.Context, accountId uuid.UUID, amount int64, currency string) error {
	op := "NotificationService.NotifyBankDepositCompleted"
	err := n.notify(ctx, accountId,
		fmt.Sprintf("Пополнение с банковского счёта на сумму %d %s", amount, currency),
		"Средства успешно пополнены",
	)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (n *NotificationService) NotifyBankDepositFailed(ctx context.Context, accountId uuid.UUID, amount int64, currency string) error {
	op := "NotificationService.NotifyBankDepositFailed"
	err := n.notify(ctx, accountId,
		fmt.Sprintf("Пополнение с банковского счёта на сумму %d %s", amount, currency),
		"Произошла ошибка при пополнении средств",
	)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (n *NotificationService) NotifyBankWithdrawalCompleted(ctx context.Context, accountId uuid.UUID, amount int64, currency string) error {
	op := "NotificationService.NotifyBankWithdrawalCompleted"
	err := n.notify(ctx, accountId,
		fmt.Sprintf("Вывод на банковский счёт на сумму %d %s", amount, currency),
		"Средства успешно выведены",
	)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (n *NotificationService) NotifyBankWithdrawalFailed(ctx context.Context, accountId uuid.UUID, amount int64, currency string) error {
	op := "NotificationService.NotifyBankWithdrawalFailed"
	err := n.notify(ctx, accountId,
		fmt.Sprintf("Вывод на банковский счёт на сумму %d %s", amount, currency),
		"Произошла ошибка при выводе средств",
	)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (n *NotificationService) notify(ctx context.Context, accountId uuid.UUID, title string, message string) error {
	companyId, err := n.getCompanyId(ctx, accountId)
	if err != nil {
		return err
	}

	users, err := n.userClient.GetUsersByCompanyId(ctx, companyId)
	if err != nil {
		return err
	}

	for _, user := range users {
		notification, err := entities.NewNotification(user.ID, title, message, accountId)
		if err != nil {
			return err
		}

		if err = n.notificationRepository.Save(ctx, notification); err != nil {
			return err
		}

		if err = n.emailSender.Send(ctx, user.Email, notification.Title(), notification.Message()); err != nil {
			return err
		}
	}

	return nil
}

func (n *NotificationService) getCompanyId(ctx context.Context, accountId uuid.UUID) (uuid.UUID, error) {
	companyId, err := n.cache.GetCompanyId(ctx, accountId)
	if err == nil {
		return companyId, nil
	}

	companyId, err = n.accountClient.GetCompanyIdByAccountId(ctx, accountId)
	if err != nil {
		return uuid.Nil, err
	}

	_ = n.cache.SetCompanyId(ctx, accountId, companyId)

	return companyId, nil
}
