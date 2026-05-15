package service

import (
	"context"
	"fmt"
	"log/slog"
	"shared/pkg/logger/sl"
	"shared/pkg/outbox"
	"time"
	"transactions/internal/domain/entities"

	"github.com/google/uuid"
)

type TransactionRepository interface {
	Save(ctx context.Context, transaction *entities.Transaction) error
	GetById(ctx context.Context, id uuid.UUID) (*entities.Transaction, error)
	GetByIdempotencyKey(ctx context.Context, idempotencyKey string) (*entities.Transaction, error)
	GetByAccountId(ctx context.Context, accountId uuid.UUID) ([]*entities.Transaction, error)
	GetStale(ctx context.Context, olderThan time.Duration, statuses []string) ([]*entities.Transaction, error)
	UpdateStatus(ctx context.Context, txId uuid.UUID, status string) error
}

type AccountClient interface {
	ReserveWithdraw(ctx context.Context, accountId, counterpartyId uuid.UUID, txId uuid.UUID, amount int64) error
	ReserveDeposit(ctx context.Context, accountId, counterpartyId uuid.UUID, txId uuid.UUID, amount int64) error
	ConfirmOperation(ctx context.Context, txId uuid.UUID) error
	CancelOperation(ctx context.Context, txId uuid.UUID) error
}

type OutboxRepository interface {
	Save(ctx context.Context, event *outbox.Event) error
}

type Transactor interface {
	WithTx(ctx context.Context, fn func(context.Context) error) error
}

type TransactionService struct {
	accountClient         AccountClient
	transactionRepository TransactionRepository
	outboxRepository      OutboxRepository
	transactor            Transactor
	log                   *slog.Logger
}

func NewTransactionService(client AccountClient, transactionRepository TransactionRepository, outboxRepository OutboxRepository, transactor Transactor, log *slog.Logger) *TransactionService {
	return &TransactionService{
		accountClient:         client,
		transactionRepository: transactionRepository,
		outboxRepository:      outboxRepository,
		transactor:            transactor,
		log:                   log,
	}
}

func (t *TransactionService) Transfer(ctx context.Context, request *TransferInput) (*entities.Transaction, error) {
	op := "TransactionService.Transfer"

	start := time.Now()
	log := t.log.With(
		sl.Op(op),
		sl.EventID(),
	)

	log.Info("starting transaction")

	existing, err := t.transactionRepository.GetByIdempotencyKey(ctx, request.IdempotencyKey)
	if err == nil && existing != nil {
		log.Info("transaction already exists", sl.Duration(time.Since(start)))
		return existing, nil
	}

	tx, err := entities.NewTransaction(
		request.InitiatorID,
		request.FromAccountId,
		request.ToAccountId,
		request.Amount,
		request.Currency,
		entities.PendingStatus,
		request.IdempotencyKey,
	)
	if err != nil {
		log.Error("failed to create new transaction", sl.ErrWithStack(err), sl.Duration(time.Since(start)))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if err = t.transactionRepository.Save(ctx, tx); err != nil {
		log.Error("failed to save transaction", sl.ErrWithStack(err), sl.Duration(time.Since(start)))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	err = t.executeSaga(ctx, tx)
	if err != nil {
		log.Error("failed to execute saga", sl.ErrWithStack(err), sl.Duration(time.Since(start)))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("transaction succeeded", sl.Duration(time.Since(start)))

	return tx, nil
}

func (t *TransactionService) executeSaga(ctx context.Context, transaction *entities.Transaction) error {
	op := "TransactionService.executeSaga"

	err := t.accountClient.ReserveWithdraw(ctx, transaction.FromAccountID(), transaction.ToAccountID(), transaction.TransactionID(), transaction.Amount())
	if err != nil {
		return t.failTransaction(ctx, transaction, fmt.Errorf("%s: withdraw reservation: %w", op, err))
	}

	err = t.accountClient.ReserveDeposit(ctx, transaction.ToAccountID(), transaction.FromAccountID(), transaction.TransactionID(), transaction.Amount())
	if err != nil {
		_ = t.accountClient.CancelOperation(ctx, transaction.TransactionID())
		return t.failTransaction(ctx, transaction, fmt.Errorf("%s: deposit reservation: %w", op, err))
	}

	_ = t.processingTransaction(ctx, transaction)

	err = t.accountClient.ConfirmOperation(ctx, transaction.TransactionID())
	if err != nil {
		return fmt.Errorf("%s: confirm failed (will retry): %w", op, err)
	}

	return t.successTransaction(ctx, transaction)
}

func (t *TransactionService) Recover(ctx context.Context, txID uuid.UUID) error {
	op := "TransactionService.Recover"

	start := time.Now()
	log := t.log.With(
		sl.Op(op),
		sl.EventID(),
	)

	log.Info("starting recover")

	tx, err := t.transactionRepository.GetById(ctx, txID)
	if err != nil {
		log.Error("failed to get transaction", sl.ErrWithStack(err), sl.Duration(time.Since(start)))
		return err
	}

	if tx.Status() == entities.PendingStatus {
		return t.executeSaga(ctx, tx)
	} else if tx.Status() == entities.ProcessingStatus {
		err = t.accountClient.ConfirmOperation(ctx, tx.TransactionID())
		if err != nil {
			log.Error("failed to confirm operation", sl.ErrWithStack(err), sl.Duration(time.Since(start)))
			return fmt.Errorf("%s: confirm failed (will retry): %w", op, err)
		}
		return t.successTransaction(ctx, tx)
	}

	log.Info("recovering succeeded", sl.Duration(time.Since(start)))

	return nil
}

func (t *TransactionService) GetAllTransactions(ctx context.Context, accountId uuid.UUID) ([]*entities.Transaction, error) {
	op := "TransactionService.GetAllTransactions"

	start := time.Now()
	log := t.log.With(
		sl.Op(op),
		sl.EventID(),
	)

	log.Info("starting list transactions")

	transactions, err := t.transactionRepository.GetByAccountId(ctx, accountId)
	if err != nil {
		log.Error("failed to get transactions", sl.ErrWithStack(err), sl.Duration(time.Since(start)))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("listing transactions succeeded", sl.Duration(time.Since(start)))

	return transactions, nil
}

func (t *TransactionService) GetTransaction(ctx context.Context, txId uuid.UUID) (*entities.Transaction, error) {
	op := "Transaction.GetTransaction"

	start := time.Now()
	log := t.log.With(
		sl.Op(op),
		sl.EventID(),
	)

	log.Info("starting get transaction")

	transaction, err := t.transactionRepository.GetById(ctx, txId)
	if err != nil {
		log.Error("failed to get transaction", sl.ErrWithStack(err), sl.Duration(time.Since(start)))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("getting transaction succeeded", sl.Duration(time.Since(start)))

	return transaction, nil
}

func (t *TransactionService) failTransaction(ctx context.Context, tx *entities.Transaction, err error) error {
	_ = tx.UpdateStatus(entities.FailedStatus)

	_ = t.transactor.WithTx(ctx, func(txCtx context.Context) error {
		if err := t.transactionRepository.UpdateStatus(txCtx, tx.TransactionID(), string(entities.FailedStatus)); err != nil {
			return err
		}

		return t.saveOutboxEvent(txCtx, tx)
	})

	return err
}

func (t *TransactionService) processingTransaction(ctx context.Context, tx *entities.Transaction) error {
	_ = tx.UpdateStatus(entities.ProcessingStatus)
	return t.transactionRepository.UpdateStatus(ctx, tx.TransactionID(), string(entities.ProcessingStatus))
}

func (t *TransactionService) successTransaction(ctx context.Context, tx *entities.Transaction) error {
	_ = tx.UpdateStatus(entities.SuccessStatus)
	return t.transactor.WithTx(ctx, func(txCtx context.Context) error {
		if err := t.transactionRepository.UpdateStatus(txCtx, tx.TransactionID(), string(entities.SuccessStatus)); err != nil {
			return err
		}

		return t.saveOutboxEvent(txCtx, tx)
	})
}

func (t *TransactionService) saveOutboxEvent(ctx context.Context, o outbox.Outboxable) error {
	evt, err := o.ToOutboxEvent()
	if err != nil {
		return fmt.Errorf("build outbox event: %w", err)
	}
	if evt == nil {
		return nil
	}
	return t.outboxRepository.Save(ctx, evt)
}
