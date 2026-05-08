package service_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"shared/pkg/outbox"
	"testing"
	"transactions/internal/domain/entities"
	"transactions/internal/service"
	mocks "transactions/internal/service/mocks"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setup(t *testing.T) (*service.TransactionService, *mocks.MockAccountClient, *mocks.MockTransactionRepository, *mocks.MockOutboxRepository, *mocks.MockTransactor) {
	client := mocks.NewMockAccountClient(t)
	repo := mocks.NewMockTransactionRepository(t)
	outboxRepo := mocks.NewMockOutboxRepository(t)
	transactor := mocks.NewMockTransactor(t)

	discardLogger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	svc := service.NewTransactionService(client, repo, outboxRepo, transactor, discardLogger)
	return svc, client, repo, outboxRepo, transactor
}

func validTransferRequest() *service.TransferInput {
	return &service.TransferInput{
		FromAccountId:  uuid.New(),
		ToAccountId:    uuid.New(),
		InitiatorID:    uuid.New(),
		Amount:         1000,
		Currency:       "USD",
		IdempotencyKey: uuid.New().String(),
	}
}

func TestTransactionService_Transfer(t *testing.T) {
	ctx := context.Background()
	req := validTransferRequest()

	t.Run("Success New Transaction", func(t *testing.T) {
		svc, client, repo, outboxRepo, transactor := setup(t)

		repo.On("GetByIdempotencyKey", mock.Anything, req.IdempotencyKey).Return(nil, nil)

		repo.On("Save", mock.Anything, mock.MatchedBy(func(tx *entities.Transaction) bool {
			return tx != nil && tx.IdempotencyKey() == req.IdempotencyKey
		})).Return(nil)

		client.On("ReserveWithdraw", mock.Anything, req.FromAccountId, mock.Anything, req.Amount).Return(nil)
		client.On("ReserveDeposit", mock.Anything, req.ToAccountId, mock.Anything, req.Amount).Return(nil)
		repo.On("UpdateStatus", mock.Anything, mock.Anything, string(entities.ProcessingStatus)).Return(nil)
		client.On("ConfirmOperation", mock.Anything, mock.Anything).Return(nil)

		transactor.On("WithTx", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context) error)
			_ = fn(args.Get(0).(context.Context))
		})

		repo.On("UpdateStatus", mock.Anything, mock.Anything, string(entities.SuccessStatus)).Return(nil)

		outboxRepo.On("Save", mock.Anything, mock.MatchedBy(func(evt *outbox.Event) bool {
			return evt != nil && evt.ID != uuid.Nil
		})).Return(nil)

		tx, err := svc.Transfer(ctx, req)
		assert.NoError(t, err)
		assert.Equal(t, entities.SuccessStatus, tx.Status())
	})

	t.Run("Idempotency - return existing transaction", func(t *testing.T) {
		svc, client, repo, _, _ := setup(t)
		initiatorId := uuid.New()
		existingTx, _ := entities.NewTransaction(initiatorId, req.FromAccountId, req.ToAccountId, req.Amount, req.Currency, entities.SuccessStatus, req.IdempotencyKey)

		repo.On("GetByIdempotencyKey", mock.Anything, req.IdempotencyKey).Return(existingTx, nil)

		tx, err := svc.Transfer(ctx, req)
		assert.NoError(t, err)
		assert.Equal(t, existingTx, tx)

		client.AssertNotCalled(t, "ReserveWithdraw")
		repo.AssertNotCalled(t, "Save")
	})
}

func TestTransactionService_SagaFailures(t *testing.T) {
	ctx := context.Background()

	t.Run("Fail on Withdraw - No Cancel needed", func(t *testing.T) {
		svc, client, repo, outboxRepo, transactor := setup(t)
		req := validTransferRequest()
		req.IdempotencyKey = "key-withdraw-fail"

		repo.On("GetByIdempotencyKey", mock.Anything, req.IdempotencyKey).Return(nil, nil)
		repo.On("Save", mock.Anything, mock.MatchedBy(func(tx *entities.Transaction) bool {
			return tx != nil && tx.IdempotencyKey() == req.IdempotencyKey
		})).Return(nil)

		client.On("ReserveWithdraw", mock.Anything, mock.Anything, mock.Anything, req.Amount).Return(errors.New("no money"))

		transactor.On("WithTx", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context) error)
			_ = fn(args.Get(0).(context.Context))
		})
		repo.On("UpdateStatus", mock.Anything, mock.Anything, string(entities.FailedStatus)).Return(nil)
		outboxRepo.On("Save", mock.Anything, mock.MatchedBy(func(evt *outbox.Event) bool {
			return evt != nil
		})).Return(nil)

		_, err := svc.Transfer(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "withdraw reservation")
	})

	t.Run("Fail on Deposit - Must call Cancel", func(t *testing.T) {
		svc, client, repo, outboxRepo, transactor := setup(t)
		req := validTransferRequest()
		req.IdempotencyKey = "key-deposit-fail"

		repo.On("GetByIdempotencyKey", mock.Anything, req.IdempotencyKey).Return(nil, nil)
		repo.On("Save", mock.Anything, mock.MatchedBy(func(tx *entities.Transaction) bool {
			return tx != nil && tx.IdempotencyKey() == req.IdempotencyKey
		})).Return(nil)

		client.On("ReserveWithdraw", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		client.On("ReserveDeposit", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("blocked"))
		client.On("CancelOperation", mock.Anything, mock.Anything).Return(nil)

		transactor.On("WithTx", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context) error)
			_ = fn(args.Get(0).(context.Context))
		})
		repo.On("UpdateStatus", mock.Anything, mock.Anything, string(entities.FailedStatus)).Return(nil)
		outboxRepo.On("Save", mock.Anything, mock.MatchedBy(func(evt *outbox.Event) bool {
			return evt != nil
		})).Return(nil)

		_, err := svc.Transfer(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "deposit reservation")
		client.AssertCalled(t, "CancelOperation", mock.Anything, mock.Anything)
	})
}

func TestTransactionService_Recover(t *testing.T) {
	ctx := context.Background()
	txID := uuid.New()

	t.Run("Recover Pending Transaction", func(t *testing.T) {
		svc, client, repo, outboxRepo, transactor := setup(t)
		initiatorId := uuid.New()
		tx, _ := entities.NewTransaction(initiatorId, uuid.New(), uuid.New(), 100, "USD", entities.PendingStatus, "k")

		repo.On("GetById", mock.Anything, txID).Return(tx, nil)

		client.On("ReserveWithdraw", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		client.On("ReserveDeposit", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		repo.On("UpdateStatus", mock.Anything, mock.Anything, string(entities.ProcessingStatus)).Return(nil)
		client.On("ConfirmOperation", mock.Anything, mock.Anything).Return(nil)

		transactor.On("WithTx", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context) error)
			_ = fn(args.Get(0).(context.Context))
		})
		repo.On("UpdateStatus", mock.Anything, mock.Anything, string(entities.SuccessStatus)).Return(nil)
		outboxRepo.On("Save", mock.Anything, mock.MatchedBy(func(evt *outbox.Event) bool {
			return evt != nil
		})).Return(nil)

		err := svc.Recover(ctx, txID)
		assert.NoError(t, err)
	})

	t.Run("Recover Processing Transaction", func(t *testing.T) {
		svc, client, repo, outboxRepo, transactor := setup(t)
		tx, _ := entities.NewTransaction(uuid.New(), uuid.New(), uuid.New(), 100, "USD", entities.ProcessingStatus, "k")

		repo.On("GetById", mock.Anything, txID).Return(tx, nil)
		client.On("ConfirmOperation", mock.Anything, mock.Anything).Return(nil)

		transactor.On("WithTx", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context) error)
			_ = fn(args.Get(0).(context.Context))
		})
		repo.On("UpdateStatus", mock.Anything, mock.Anything, string(entities.SuccessStatus)).Return(nil)
		outboxRepo.On("Save", mock.Anything, mock.MatchedBy(func(evt *outbox.Event) bool {
			return evt != nil
		})).Return(nil)

		err := svc.Recover(ctx, txID)
		assert.NoError(t, err)
	})

	t.Run("Recover already completed - no-op", func(t *testing.T) {
		svc, _, repo, _, _ := setup(t)
		tx, _ := entities.NewTransaction(uuid.New(), uuid.New(), uuid.New(), 100, "USD", entities.SuccessStatus, "k")

		repo.On("GetById", mock.Anything, txID).Return(tx, nil)

		err := svc.Recover(ctx, txID)
		assert.NoError(t, err)
	})
}

func TestTransactionService_GetTransaction(t *testing.T) {
	ctx := context.Background()
	txID := uuid.New()

	t.Run("Success", func(t *testing.T) {
		svc, _, repo, _, _ := setup(t)
		expectedTx, _ := entities.NewTransaction(uuid.New(), uuid.New(), uuid.New(), 100, "USD", entities.SuccessStatus, "key")

		repo.On("GetById", mock.Anything, txID).Return(expectedTx, nil)

		tx, err := svc.GetTransaction(ctx, txID)
		assert.NoError(t, err)
		assert.Equal(t, expectedTx, tx)
	})

	t.Run("Not Found", func(t *testing.T) {
		svc, _, repo, _, _ := setup(t)
		repo.On("GetById", mock.Anything, txID).Return(nil, errors.New("not found"))

		tx, err := svc.GetTransaction(ctx, txID)
		assert.Error(t, err)
		assert.Nil(t, tx)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestTransactionService_GetAllTransactions(t *testing.T) {
	ctx := context.Background()
	accountID := uuid.New()

	t.Run("Success with multiple transactions", func(t *testing.T) {
		svc, _, repo, _, _ := setup(t)
		tx1, _ := entities.NewTransaction(uuid.New(), accountID, uuid.New(), 100, "USD", entities.SuccessStatus, "key1")
		tx2, _ := entities.NewTransaction(uuid.New(), accountID, uuid.New(), 200, "EUR", entities.PendingStatus, "key2")
		expected := []*entities.Transaction{tx1, tx2}

		repo.On("GetByAccountId", mock.Anything, accountID).Return(expected, nil)

		txs, err := svc.GetAllTransactions(ctx, accountID)
		assert.NoError(t, err)
		assert.Len(t, txs, 2)
		assert.Equal(t, expected, txs)
	})

	t.Run("Empty result", func(t *testing.T) {
		svc, _, repo, _, _ := setup(t)
		repo.On("GetByAccountId", mock.Anything, accountID).Return([]*entities.Transaction{}, nil)

		txs, err := svc.GetAllTransactions(ctx, accountID)
		assert.NoError(t, err)
		assert.Empty(t, txs)
	})

	t.Run("Repository error", func(t *testing.T) {
		svc, _, repo, _, _ := setup(t)
		repo.On("GetByAccountId", mock.Anything, accountID).Return(nil, errors.New("db error"))

		txs, err := svc.GetAllTransactions(ctx, accountID)
		assert.Error(t, err)
		assert.Nil(t, txs)
		assert.Contains(t, err.Error(), "db error")
	})
}
