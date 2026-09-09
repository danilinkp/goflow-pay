package service

import (
	"context"
	"fmt"
	"log/slog"
	"notifications/internal/domain/entities"
	"shared/pkg/logger/sl"
	"strings"
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

	title := fmt.Sprintf("Перевод на %d %s выполнен", amount, currency)
	body := wrapInHTML("<h3>Успешная операция</h3><p>Средства успешно переведены.</p>")

	err := n.notify(ctx, userId, accountId, title, body)

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

	title := fmt.Sprintf("Перевод на %d %s не выполнен", amount, currency)
	body := wrapInHTML("<h3 style='color: #dc3545;'>Ошибка перевода</h3><p>Произошла ошибка при переводе средств.</p>")

	err := n.notify(ctx, userId, accountId, title, body)
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

	title := fmt.Sprintf("Пополнение с банковского счёта на сумму %d %s", amount, currency)
	body := wrapInHTML("<h3>Успешная операция</h3><p>Средства успешно пополнены.</p>")

	err := n.notify(ctx, userId, accountId, title, body)
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

	title := fmt.Sprintf("Пополнение с банковского счёта на сумму %d %s", amount, currency)
	body := wrapInHTML("<h3 style='color: #dc3545;'>Ошибка перевода</h3><p>Произошла ошибка при пополнении средств.</p>")

	err := n.notify(ctx, userId, accountId, title, body)
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

	title := fmt.Sprintf("Вывод на банковский счёт на сумму %d %s", amount, currency)
	body := wrapInHTML("<h3>Успешная операция</h3><p>Средства успешно выведены.</p>")

	err := n.notify(ctx, userId, accountId, title, body)
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

	title := fmt.Sprintf("Вывод на банковский счёт на сумму %d %s", amount, currency)
	body := wrapInHTML("<h3 style='color: #dc3545;'>Ошибка перевода</h3><p>Произошла ошибка при выводе средств.</p>")

	err := n.notify(ctx, userId, accountId, title, body)
	if err != nil {
		log.Error("notify bank withdrawal failed failed",
			sl.ErrWithStack(err),
			sl.Duration(time.Since(start)))
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (n *NotificationService) NotifyStatementGenerated(
	ctx context.Context,
	userId, accountId uuid.UUID,
	periodFrom, periodTo time.Time,
	openingBalance, closingBalance, totalDebit, totalCredit int64,
	currency string,
	entries []StatementEntry,
) error {
	op := "NotificationService.NotifyStatementGenerated"
	start := time.Now()

	log := n.log.With(
		sl.Op(op),
		sl.EventID(),
		slog.String("user_id", userId.String()),
		slog.String("account_id", accountId.String()),
	)

	log.Info("notify statement generated attempt")

	title := fmt.Sprintf("Выписка по счёту за период %s — %s",
		periodFrom.Format("02.01.2006"),
		periodTo.Format("02.01.2006"),
	)

	message := fmt.Sprintf(`
        <html>
        <body style="font-family: sans-serif; color: #333; line-height: 1.6; padding: 20px;">
            <h2 style="color: #007bff; border-bottom: 2px solid #007bff; padding-bottom: 10px;">
                Выписка по счёту
            </h2>
            <p style="font-size: 14px; color: #666;">
                ID счёта: <code style="background: #f4f4f4; padding: 2px 5px;">%s</code>
            </p>
            
            <div style="background-color: #f8f9fa; padding: 15px; border-radius: 8px; margin-bottom: 20px;">
                <table style="width: 100%%; font-size: 14px;">
                    <tr>
                        <td><strong>Период:</strong> %s — %s</td>
                        <td style="text-align: right;"><strong>Валюта:</strong> %s</td>
                    </tr>
                </table>
                <hr style="border: 0; border-top: 1px solid #dee2e6; margin: 10px 0;">
                <table style="width: 100%%; font-size: 14px; border-spacing: 0 5px;">
                    <tr>
                        <td>Открывающий баланс:</td>
                        <td style="text-align: right;"><strong>%d</strong></td>
                    </tr>
                    <tr>
                        <td>Закрывающий баланс:</td>
                        <td style="text-align: right; font-size: 16px; color: #000;"><strong>%d</strong></td>
                    </tr>
                    <tr style="color: #28a745;">
                        <td>Итого поступлений:</td>
                        <td style="text-align: right;">+ %d</td>
                    </tr>
                    <tr style="color: #dc3545;">
                        <td>Итого списаний:</td>
                        <td style="text-align: right;">- %d</td>
                    </tr>
                </table>
            </div>

            <h3 style="color: #333;">Операции за период:</h3>
            %s
            
            <p style="margin-top: 30px; font-size: 12px; color: #999; text-align: center;">
                Это автоматическое уведомление от GoFlow Pay.
            </p>
        </body>
        </html>`,
		accountId,
		periodFrom.Format("02.01.2006"),
		periodTo.Format("02.01.2006"),
		currency,
		openingBalance,
		closingBalance,
		totalDebit,
		totalCredit,
		formatEntries(entries),
	)

	err := n.notify(ctx, userId, accountId, title, message)
	if err != nil {
		log.Error("notify statement generated failed",
			sl.ErrWithStack(err),
			sl.Duration(time.Since(start)),
		)
		return fmt.Errorf("%s: %w", op, err)
	}

	log.Info("statement generated notification sent",
		sl.Duration(time.Since(start)),
	)
	return nil
}

func formatEntries(entries []StatementEntry) string {
	if len(entries) == 0 {
		return "<p style='font-family: sans-serif; color: #666;'>Нет операций за выбранный период.</p>"
	}

	var sb strings.Builder

	sb.WriteString(`
        <table style="width: 100%; border-collapse: collapse; font-family: sans-serif; font-size: 14px;">
            <thead>
                <tr style="background-color: #f8f9fa; border-bottom: 2px solid #dee2e6; text-align: left;">
                    <th style="padding: 12px;">Дата</th>
                    <th style="padding: 12px;">Тип операции</th>
                    <th style="padding: 12px; text-align: right;">Сумма</th>
                    <th style="padding: 12px; text-align: right;">Баланс</th>
                    <th style="padding: 12px;">Контрагент</th>
                </tr>
            </thead>
            <tbody>
    `)

	for i, e := range entries {
		bgColor := "#ffffff"
		if i%2 != 0 {
			bgColor = "#fcfcfc"
		}

		sign := "+"
		color := "#28a745"
		if e.EntryType == EntryTypeTransferOut || e.EntryType == EntryTypeBankWithdrawal {
			sign = "-"
			color = "#dc3545"
		}

		fmt.Fprintf(&sb, `
            <tr style="background-color: %s; border-bottom: 1px solid #eee;">
                <td style="padding: 12px; white-space: nowrap;">%s</td>
                <td style="padding: 12px;">%s</td>
                <td style="padding: 12px; text-align: right; color: %s; font-weight: bold;">%s%d</td>
                <td style="padding: 12px; text-align: right;">%d</td>
                <td style="padding: 12px;">%s</td>
            </tr>`,
			bgColor,
			e.Date.Format("02.01.2006 15:04"),
			entryTypeLabel(e.EntryType),
			color,
			sign, e.Amount,
			e.BalanceAfter,
			e.Counterparty,
		)
	}

	sb.WriteString(`
            </tbody>
        </table>
    `)

	return sb.String()
}

func entryTypeLabel(t StatementEntryType) string {
	switch t {
	case EntryTypeTransferIn:
		return "Входящий перевод"
	case EntryTypeTransferOut:
		return "Исходящий перевод"
	case EntryTypeBankDeposit:
		return "Пополнение с банка"
	case EntryTypeBankWithdrawal:
		return "Вывод на банк"
	default:
		return string(t)
	}
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

func wrapInHTML(content string) string {
	return fmt.Sprintf(`
        <html>
        <body style="font-family: sans-serif; line-height: 1.5; color: #333; padding: 20px;">
            <div style="border-left: 4px solid #007bff; padding-left: 15px; margin: 10px 0;">
                %s
            </div>
            <p style="font-size: 12px; color: #999; margin-top: 20px;">
                Это автоматическое уведомление GoFlow Pay.
            </p>
        </body>
        </html>`, content)
}
