package services

import (
	"context"
	"fmt"
	"log/slog"
	"notifications/internal/domain/entities"
	"shared/pkg/logger/sl"
	"time"

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
	log                    *slog.Logger
	singleFlightGroup      *singleflight.Group
}

func NewNotificationService(userClient UserClient, notificationRepository NotificationRepository, emailSender EmailSender, log *slog.Logger) *NotificationService {
	return &NotificationService{
		userClient:             userClient,
		notificationRepository: notificationRepository,
		emailSender:            emailSender,
		log:                    log,
		singleFlightGroup:      &singleflight.Group{},
	}
}

func (n *NotificationService) NotifyTransferCompleted(ctx context.Context, userId, accountId uuid.UUID, amount int64, currency string) error {
	op := "NotificationService.NotifyTransferCompleted"

	start := time.Now()
	log := n.log.With(
		sl.Op(op),
		sl.EventID(),
	)

	log.Info("notify transfer completed attempt")

	err := n.notify(ctx, userId, accountId,
		fmt.Sprintf("Перевод на %d %s выполнен", amount, currency),
		"Средства успешно переведены",
	)
	if err != nil {
		log.Error("failed to notify transfer completed", sl.ErrWithStack(err), sl.Duration(time.Since(start)))
		return fmt.Errorf("%s: %w", op, err)
	}

	log.Info("notify transfer completed successfully", sl.Duration(time.Since(start)))

	return nil
}

func (n *NotificationService) NotifyTransferFailed(ctx context.Context, userId, accountId uuid.UUID, amount int64, currency string) error {
	op := "NotificationService.NotifyTransferFailed"

	start := time.Now()
	log := n.log.With(
		sl.Op(op),
		sl.EventID(),
	)

	log.Info("notify transfer failed attempt")

	err := n.notify(ctx, userId, accountId,
		fmt.Sprintf("Перевод на %d %s не выполнен", amount, currency),
		"Произошла ошибка при переводе средств",
	)
	if err != nil {
		log.Error("failed to notify transfer failed", sl.ErrWithStack(err), sl.Duration(time.Since(start)))
		return fmt.Errorf("%s: %w", op, err)
	}

	log.Info("notify transfer failed successfully", sl.Duration(time.Since(start)))

	return nil
}

func (n *NotificationService) NotifyBankDepositCompleted(ctx context.Context, userId, accountId uuid.UUID, amount int64, currency string) error {
	op := "NotificationService.NotifyBankDepositCompleted"

	start := time.Now()
	log := n.log.With(
		sl.Op(op),
		sl.EventID(),
	)

	log.Info("notify bank deposit completed attempt")

	err := n.notify(ctx, userId, accountId,
		fmt.Sprintf("Пополнение с банковского счёта на сумму %d %s", amount, currency),
		"Средства успешно пополнены",
	)
	if err != nil {
		log.Error("notify bank deposit completed failed",
			sl.ErrWithStack(err),
			sl.Duration(time.Since(start)))
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (n *NotificationService) NotifyBankDepositFailed(ctx context.Context, userId, accountId uuid.UUID, amount int64, currency string) error {
	op := "NotificationService.NotifyBankDepositFailed"

	start := time.Now()
	log := n.log.With(
		sl.Op(op),
		sl.EventID(),
	)

	log.Info("notify bank deposit failed attempt")

	err := n.notify(ctx, userId, accountId,
		fmt.Sprintf("Пополнение с банковского счёта на сумму %d %s", amount, currency),
		"Произошла ошибка при пополнении средств",
	)
	if err != nil {
		log.Error("notify bank deposit failed failed",
			sl.ErrWithStack(err),
			sl.Duration(time.Since(start)))
		return fmt.Errorf("%s: %w", op, err)
	}

	log.Info("notify bank deposit failed successfully", sl.Duration(time.Since(start)))

	return nil
}

func (n *NotificationService) NotifyBankWithdrawalCompleted(ctx context.Context, userId, accountId uuid.UUID, amount int64, currency string) error {
	op := "NotificationService.NotifyBankWithdrawalCompleted"

	start := time.Now()
	log := n.log.With(
		sl.Op(op),
		sl.EventID(),
	)

	log.Info("notify bank withdrawal completed attempt")

	err := n.notify(ctx, userId, accountId,
		fmt.Sprintf("Вывод на банковский счёт на сумму %d %s", amount, currency),
		"Средства успешно выведены",
	)
	if err != nil {
		log.Error("notify bank withdrawal completed failed",
			sl.ErrWithStack(err),
			sl.Duration(time.Since(start)))
		return fmt.Errorf("%s: %w", op, err)
	}

	log.Info("notify bank withdrawal completed successfully",
		sl.Duration(time.Since(start)))

	return nil
}

func (n *NotificationService) NotifyBankWithdrawalFailed(ctx context.Context, userId, accountId uuid.UUID, amount int64, currency string) error {
	op := "NotificationService.NotifyBankWithdrawalFailed"

	start := time.Now()
	log := n.log.With(
		sl.Op(op),
		sl.EventID(),
	)

	log.Info("notify bank withdrawal failed attempt")

	err := n.notify(ctx, userId, accountId,
		fmt.Sprintf("Вывод на банковский счёт на сумму %d %s", amount, currency),
		"Произошла ошибка при выводе средств",
	)
	if err != nil {
		log.Error("notify bank withdrawal failed failed",
			sl.ErrWithStack(err),
			sl.Duration(time.Since(start)))
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
