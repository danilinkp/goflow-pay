package services

import (
	"context"
	"fmt"
	"notifications/internal/domain/entities"

	"github.com/google/uuid"
	"golang.org/x/sync/singleflight"
)

type UserClient interface {
	GetUserById(ctx context.Context, userId uuid.UUID) (*UserResponse, error)
}

type NotificationRepository interface {
	Save(ctx context.Context, notification *entities.Notification) error
	GetById(ctx context.Context, id uuid.UUID) (*entities.Notification, error)
}

type EmailSender interface {
	Send(ctx context.Context, email string, title string, message string) error
}

type NotificationService struct {
	userClient             UserClient
	notificationRepository NotificationRepository
	emailSender            EmailSender
	singleFlightGroup      *singleflight.Group
}

func NewNotificationService(userClient UserClient, notificationRepository NotificationRepository, emailSender EmailSender) *NotificationService {
	return &NotificationService{
		userClient:             userClient,
		notificationRepository: notificationRepository,
		emailSender:            emailSender,
		singleFlightGroup:      &singleflight.Group{},
	}
}

func (n *NotificationService) NotifyTransferCompleted(ctx context.Context, userId, accountId uuid.UUID, amount int64, currency string) error {
	op := "NotificationService.NotifyTransferCompleted"
	err := n.notify(ctx, userId, accountId,
		fmt.Sprintf("Перевод на %d %s выполнен", amount, currency),
		"Средства успешно переведены",
	)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (n *NotificationService) NotifyTransferFailed(ctx context.Context, userId, accountId uuid.UUID, amount int64, currency string) error {
	op := "NotificationService.NotifyTransferFailed"
	err := n.notify(ctx, userId, accountId,
		fmt.Sprintf("Перевод на %d %s не выполнен", amount, currency),
		"Произошла ошибка при переводе средств",
	)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (n *NotificationService) NotifyBankDepositCompleted(ctx context.Context, userId, accountId uuid.UUID, amount int64, currency string) error {
	op := "NotificationService.NotifyBankDepositCompleted"
	err := n.notify(ctx, userId, accountId,
		fmt.Sprintf("Пополнение с банковского счёта на сумму %d %s", amount, currency),
		"Средства успешно пополнены",
	)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (n *NotificationService) NotifyBankDepositFailed(ctx context.Context, userId, accountId uuid.UUID, amount int64, currency string) error {
	op := "NotificationService.NotifyBankDepositFailed"
	err := n.notify(ctx, userId, accountId,
		fmt.Sprintf("Пополнение с банковского счёта на сумму %d %s", amount, currency),
		"Произошла ошибка при пополнении средств",
	)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (n *NotificationService) NotifyBankWithdrawalCompleted(ctx context.Context, userId, accountId uuid.UUID, amount int64, currency string) error {
	op := "NotificationService.NotifyBankWithdrawalCompleted"
	err := n.notify(ctx, userId, accountId,
		fmt.Sprintf("Вывод на банковский счёт на сумму %d %s", amount, currency),
		"Средства успешно выведены",
	)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (n *NotificationService) NotifyBankWithdrawalFailed(ctx context.Context, userId, accountId uuid.UUID, amount int64, currency string) error {
	op := "NotificationService.NotifyBankWithdrawalFailed"
	err := n.notify(ctx, userId, accountId,
		fmt.Sprintf("Вывод на банковский счёт на сумму %d %s", amount, currency),
		"Произошла ошибка при выводе средств",
	)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (n *NotificationService) notify(ctx context.Context, userId, sourceId uuid.UUID, title string, message string) error {
	user, err := n.userClient.GetUserById(ctx, userId)
	if err != nil {
		return err
	}

	notification, err := entities.NewNotification(userId, title, message, sourceId)
	if err != nil {
		return err
	}

	if err = n.notificationRepository.Save(ctx, notification); err != nil {
		return err
	}

	if err = n.emailSender.Send(ctx, user.Email, notification.Title(), notification.Message()); err != nil {
		return err
	}

	return nil
}
