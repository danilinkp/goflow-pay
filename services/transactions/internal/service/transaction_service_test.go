package service_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"shared/pkg/outbox"
	"testing"
	"time"
	"transactions/internal/domain/entities"
	"transactions/internal/service"
	mocks "transactions/internal/service/mocks"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	allure "github.com/allure-framework/allure-go/commons/gotest"
)

func setup(t *testing.T, now ...func() time.Time) (*service.TransactionService, *mocks.MockAccountClient, *mocks.MockTransactionRepository, *mocks.MockOutboxRepository, *mocks.MockTransactor) {
	t.Helper()
	client := mocks.NewMockAccountClient(t)
	repo := mocks.NewMockTransactionRepository(t)
	outboxRepo := mocks.NewMockOutboxRepository(t)
	transactor := mocks.NewMockTransactor(t)

	discardLogger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	svc := service.NewTransactionService(client, repo, outboxRepo, transactor, discardLogger, now...)
	return svc, client, repo, outboxRepo, transactor
}

func TestTransactionService_Transfer_Success(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		ctx := context.Background()
		fixedTime := time.Date(2026, time.January, 15, 10, 0, 0, 0, time.UTC)
		svc, client, repo, outboxRepo, transactor := setup(t, func() time.Time { return fixedTime })

		var mother TransactionMother
		req := mother.ValidTransferInput()

		a.Step("Arrange: configure idempotency and mock expectations with deterministic timestamp", func(a *allure.Context) {
			repo.On("GetByIdempotencyKey", mock.Anything, req.IdempotencyKey).Return(nil, nil)
			repo.On("Save", mock.Anything, mock.MatchedBy(func(tx *entities.Transaction) bool {
				return tx != nil &&
					tx.IdempotencyKey() == req.IdempotencyKey &&
					tx.Amount() == req.Amount &&
					tx.CreatedAt().Equal(fixedTime)
			})).Return(nil)

			client.On("ReserveWithdraw", mock.Anything, req.FromAccountId, req.ToAccountId, mock.Anything, req.Amount).Return(nil)
			client.On("ReserveDeposit", mock.Anything, req.ToAccountId, req.FromAccountId, mock.Anything, req.Amount).Return(nil)
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
		})

		var tx *entities.Transaction
		var err error
		a.Step("Act: execute Transfer", func(a *allure.Context) {
			tx, err = svc.Transfer(ctx, req)
		})

		a.Step("Assert: verify transaction status, exact timestamp, and mock expectations", func(a *allure.Context) {
			require.NoError(t, err)
			require.NotNil(t, tx)
			assert.Equal(t, entities.SuccessStatus, tx.Status())
			assert.Equal(t, req.Amount, tx.Amount())
			assert.Equal(t, fixedTime, tx.CreatedAt())
			client.AssertExpectations(t)
			repo.AssertExpectations(t)
		})
	},
		allure.WithParentSuite("Лондонская школа (Solitary)"),
		allure.WithSuite("TransactionService"),
		allure.WithSubSuite("Transfer"),
		allure.WithEpic("Transactions"),
		allure.WithFeature("Transfer"),
		allure.WithStory("Positive: Successful Money Transfer"),
	)
}

func TestTransactionService_Transfer_Idempotency_ReturnsExistingTransaction(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		ctx := context.Background()
		var mother TransactionMother
		req := mother.ValidTransferInput()
		existingTx := mother.CompletedTransaction(req)
		svc, client, repo, _, _ := setup(t)

		a.Step("Arrange: set up existing completed transaction in repository", func(a *allure.Context) {
			repo.On("GetByIdempotencyKey", mock.Anything, req.IdempotencyKey).Return(existingTx, nil)
		})

		var tx *entities.Transaction
		var err error
		a.Step("Act: execute Transfer with same idempotency key", func(a *allure.Context) {
			tx, err = svc.Transfer(ctx, req)
		})

		a.Step("Assert: verify existing transaction returned without balance operations", func(a *allure.Context) {
			require.NoError(t, err)
			require.NotNil(t, tx)
			assert.Equal(t, existingTx.TransactionID(), tx.TransactionID())
			assert.Equal(t, entities.SuccessStatus, tx.Status())
			repo.AssertNotCalled(t, "Save", mock.Anything, mock.Anything)
			client.AssertNotCalled(t, "ReserveWithdraw", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
			client.AssertNotCalled(t, "ReserveDeposit", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		})
	},
		allure.WithParentSuite("Лондонская школа (Solitary)"),
		allure.WithSuite("TransactionService"),
		allure.WithSubSuite("Transfer"),
		allure.WithEpic("Transactions"),
		allure.WithFeature("Transfer"),
		allure.WithStory("Positive: Idempotent transfer handling"),
	)
}

func TestTransactionService_Transfer_CompensationOnDepositFailure(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		ctx := context.Background()
		req := NewTransferInputBuilder().
			WithAmount(5000).
			WithIdempotencyKey("deposit-fail-test-key").
			Build()
		svc, client, repo, outboxRepo, transactor := setup(t)

		a.Step("Arrange: configure deposit reservation failure and compensation mocks", func(a *allure.Context) {
			repo.On("GetByIdempotencyKey", mock.Anything, req.IdempotencyKey).Return(nil, nil)
			repo.On("Save", mock.Anything, mock.Anything).Return(nil)
			client.On("ReserveWithdraw", mock.Anything, mock.Anything, mock.Anything, mock.Anything, req.Amount).Return(nil)
			client.On("ReserveDeposit", mock.Anything, mock.Anything, mock.Anything, mock.Anything, req.Amount).Return(errors.New("deposit endpoint unavailable"))
			client.On("CancelOperation", mock.Anything, mock.Anything).Return(nil)

			transactor.On("WithTx", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
				fn := args.Get(1).(func(context.Context) error)
				_ = fn(args.Get(0).(context.Context))
			})
			repo.On("UpdateStatus", mock.Anything, mock.Anything, string(entities.FailedStatus)).Return(nil)
			outboxRepo.On("Save", mock.Anything, mock.Anything).Return(nil)
		})

		var tx *entities.Transaction
		var err error
		a.Step("Act: execute Transfer", func(a *allure.Context) {
			tx, err = svc.Transfer(ctx, req)
		})

		a.Step("Assert: verify error returned, withdrawal cancelled, and status marked Failed", func(a *allure.Context) {
			require.Error(t, err)
			assert.Nil(t, tx)
			assert.Contains(t, err.Error(), "deposit reservation")
			client.AssertCalled(t, "CancelOperation", mock.Anything, mock.Anything)
			repo.AssertCalled(t, "UpdateStatus", mock.Anything, mock.Anything, string(entities.FailedStatus))
		})
	},
		allure.WithParentSuite("Лондонская школа (Solitary)"),
		allure.WithSuite("TransactionService"),
		allure.WithSubSuite("Transfer"),
		allure.WithEpic("Transactions"),
		allure.WithFeature("Transfer"),
		allure.WithStory("Negative: Compensating transaction on deposit failure"),
	)
}

func TestTransactionService_Transfer_InvalidAmount(t *testing.T) {
	testCases := []struct {
		name   string
		amount int64
	}{
		{name: "ZeroAmount", amount: 0},
		{name: "NegativeAmount", amount: -1},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			allure.Wrap(t, func(a *allure.Context) {
				ctx := context.Background()
				req := NewTransferInputBuilder().WithAmount(tc.amount).Build()
				svc, client, repo, _, _ := setup(t)

				a.Step("Arrange: configure repository mock for idempotency check", func(a *allure.Context) {
					repo.On("GetByIdempotencyKey", mock.Anything, req.IdempotencyKey).Return(nil, nil)
				})

				var tx *entities.Transaction
				var err error
				a.Step("Act: execute Transfer with non-positive amount", func(a *allure.Context) {
					tx, err = svc.Transfer(ctx, req)
				})

				a.Step("Assert: verify validation error and ensure no balance or repository operations", func(a *allure.Context) {
					require.Error(t, err)
					assert.Nil(t, tx)
					assert.Contains(t, err.Error(), "amount must be positive")
					repo.AssertNotCalled(t, "Save", mock.Anything, mock.Anything)
					client.AssertNotCalled(t, "ReserveWithdraw", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
				})
			},
				allure.WithParentSuite("Лондонская школа (Solitary)"),
				allure.WithSuite("TransactionService"),
				allure.WithSubSuite("Transfer"),
				allure.WithEpic("Transactions"),
				allure.WithFeature("Transfer"),
				allure.WithStory("Negative: Boundary validation of transfer amount"),
			)
		})
	}
}

func TestTransactionService_Recover_PendingStatus_Success(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		ctx := context.Background()
		var mother TransactionMother
		req := mother.ValidTransferInput()
		pendingTx := mother.PendingTransaction(req)
		svc, client, repo, outboxRepo, transactor := setup(t)

		a.Step("Arrange: prepare pending transaction and mock successful saga completion", func(a *allure.Context) {
			repo.On("GetById", mock.Anything, pendingTx.TransactionID()).Return(pendingTx, nil)
			client.On("ReserveWithdraw", mock.Anything, pendingTx.FromAccountID(), pendingTx.ToAccountID(), pendingTx.TransactionID(), pendingTx.Amount()).Return(nil)
			client.On("ReserveDeposit", mock.Anything, pendingTx.ToAccountID(), pendingTx.FromAccountID(), pendingTx.TransactionID(), pendingTx.Amount()).Return(nil)
			repo.On("UpdateStatus", mock.Anything, pendingTx.TransactionID(), string(entities.ProcessingStatus)).Return(nil)
			client.On("ConfirmOperation", mock.Anything, pendingTx.TransactionID()).Return(nil)

			transactor.On("WithTx", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
				fn := args.Get(1).(func(context.Context) error)
				_ = fn(args.Get(0).(context.Context))
			})
			repo.On("UpdateStatus", mock.Anything, pendingTx.TransactionID(), string(entities.SuccessStatus)).Return(nil)
			outboxRepo.On("Save", mock.Anything, mock.Anything).Return(nil)
		})

		var err error
		a.Step("Act: execute Recover on pending transaction", func(a *allure.Context) {
			err = svc.Recover(ctx, pendingTx.TransactionID())
		})

		a.Step("Assert: verify transaction status updated to Success and expectations met", func(a *allure.Context) {
			require.NoError(t, err)
			assert.Equal(t, entities.SuccessStatus, pendingTx.Status())
			client.AssertExpectations(t)
			repo.AssertExpectations(t)
		})
	},
		allure.WithParentSuite("Лондонская школа (Solitary)"),
		allure.WithSuite("TransactionService"),
		allure.WithSubSuite("Recover"),
		allure.WithEpic("Transactions"),
		allure.WithFeature("Recover"),
		allure.WithStory("Positive: Resume pending transaction saga"),
	)
}

func TestTransactionService_Recover_TransactionNotFound_ThrowsError(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		ctx := context.Background()
		txID := uuid.New()
		svc, _, repo, _, _ := setup(t)

		a.Step("Arrange: mock repository returning not found error", func(a *allure.Context) {
			repo.On("GetById", mock.Anything, txID).Return(nil, errors.New("transaction not found"))
		})

		var err error
		a.Step("Act: execute Recover with non-existent transaction ID", func(a *allure.Context) {
			err = svc.Recover(ctx, txID)
		})

		a.Step("Assert: verify error returned and contains not found message", func(a *allure.Context) {
			require.Error(t, err)
			assert.Contains(t, err.Error(), "transaction not found")
		})
	},
		allure.WithParentSuite("Лондонская школа (Solitary)"),
		allure.WithSuite("TransactionService"),
		allure.WithSubSuite("Recover"),
		allure.WithEpic("Transactions"),
		allure.WithFeature("Recover"),
		allure.WithStory("Negative: Recovery for non-existent transaction"),
	)
}

func TestTransactionService_GetAllTransactions_Success(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		ctx := context.Background()
		accountID := uuid.New()
		var mother TransactionMother
		expectedTxs := []*entities.Transaction{
			mother.CompletedTransaction(mother.ValidTransferInput()),
		}
		svc, _, repo, _, _ := setup(t)

		a.Step("Arrange: mock repository returning transaction list", func(a *allure.Context) {
			repo.On("GetByAccountId", mock.Anything, accountID).Return(expectedTxs, nil)
		})

		var txs []*entities.Transaction
		var err error
		a.Step("Act: execute GetAllTransactions", func(a *allure.Context) {
			txs, err = svc.GetAllTransactions(ctx, accountID)
		})

		a.Step("Assert: verify transaction list matches expected data", func(a *allure.Context) {
			require.NoError(t, err)
			assert.Equal(t, expectedTxs, txs)
			repo.AssertExpectations(t)
		})
	},
		allure.WithParentSuite("Лондонская школа (Solitary)"),
		allure.WithSuite("TransactionService"),
		allure.WithSubSuite("GetAllTransactions"),
		allure.WithEpic("Transactions"),
		allure.WithFeature("History"),
		allure.WithStory("Positive: Retrieve account transaction history"),
	)
}

func TestTransactionService_GetAllTransactions_RepositoryError_ThrowsError(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		ctx := context.Background()
		accountID := uuid.New()
		svc, _, repo, _, _ := setup(t)

		a.Step("Arrange: mock repository failure", func(a *allure.Context) {
			repo.On("GetByAccountId", mock.Anything, accountID).Return(nil, errors.New("database connection failed"))
		})

		var txs []*entities.Transaction
		var err error
		a.Step("Act: execute GetAllTransactions", func(a *allure.Context) {
			txs, err = svc.GetAllTransactions(ctx, accountID)
		})

		a.Step("Assert: verify repository error is propagated", func(a *allure.Context) {
			require.Error(t, err)
			assert.Nil(t, txs)
			assert.Contains(t, err.Error(), "database connection failed")
		})
	},
		allure.WithParentSuite("Лондонская школа (Solitary)"),
		allure.WithSuite("TransactionService"),
		allure.WithSubSuite("GetAllTransactions"),
		allure.WithEpic("Transactions"),
		allure.WithFeature("History"),
		allure.WithStory("Negative: Database error handling when fetching history"),
	)
}

func TestTransactionService_GetTransaction_Success(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		ctx := context.Background()
		var mother TransactionMother
		expectedTx := mother.CompletedTransaction(mother.ValidTransferInput())
		svc, _, repo, _, _ := setup(t)

		a.Step("Arrange: mock repository returning existing transaction", func(a *allure.Context) {
			repo.On("GetById", mock.Anything, expectedTx.TransactionID()).Return(expectedTx, nil)
		})

		var tx *entities.Transaction
		var err error
		a.Step("Act: execute GetTransaction", func(a *allure.Context) {
			tx, err = svc.GetTransaction(ctx, expectedTx.TransactionID())
		})

		a.Step("Assert: verify transaction details returned successfully", func(a *allure.Context) {
			require.NoError(t, err)
			assert.Equal(t, expectedTx, tx)
		})
	},
		allure.WithParentSuite("Лондонская школа (Solitary)"),
		allure.WithSuite("TransactionService"),
		allure.WithSubSuite("GetTransaction"),
		allure.WithEpic("Transactions"),
		allure.WithFeature("Details"),
		allure.WithStory("Positive: Retrieve transaction by ID"),
	)
}

func TestTransactionService_GetTransaction_NotFound(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		ctx := context.Background()
		txID := uuid.New()
		svc, _, repo, _, _ := setup(t)

		a.Step("Arrange: mock repository returning not found error", func(a *allure.Context) {
			repo.On("GetById", mock.Anything, txID).Return(nil, errors.New("not found"))
		})

		var tx *entities.Transaction
		var err error
		a.Step("Act: execute GetTransaction with non-existent ID", func(a *allure.Context) {
			tx, err = svc.GetTransaction(ctx, txID)
		})

		a.Step("Assert: verify not found error is propagated", func(a *allure.Context) {
			require.Error(t, err)
			assert.Nil(t, tx)
			assert.Contains(t, err.Error(), "not found")
		})
	},
		allure.WithParentSuite("Лондонская школа (Solitary)"),
		allure.WithSuite("TransactionService"),
		allure.WithSubSuite("GetTransaction"),
		allure.WithEpic("Transactions"),
		allure.WithFeature("Details"),
		allure.WithStory("Negative: Transaction not found error"),
	)
}
