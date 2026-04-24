package services_test

import (
	"accounts/internal/domain/entities"
	"accounts/internal/services"
	mocks "accounts/internal/services/mocks"
	"context"
	"errors"
	"shared/pkg/outbox"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	testBIC        = "044525225"
	testAccountRUB = "40702810200000000001"
	testAccountUSD = "40702840500000000001"
	testAccountEUR = "40702978100000000001"
)

func setupAccountService(t *testing.T) (
	*services.AccountService,
	*mocks.MockAccountRepository,
	*mocks.MockAccountOperationRepository,
	*mocks.MockBankAccountRepository,
	*mocks.MockBankOperationRepository,
	*mocks.MockBankGateway,
	*mocks.MockOutboxRepository,
	*mocks.MockTransactor,
) {
	accountRepo := mocks.NewMockAccountRepository(t)
	accountOpRepo := mocks.NewMockAccountOperationRepository(t)
	bankAccountRepo := mocks.NewMockBankAccountRepository(t)
	bankOpRepo := mocks.NewMockBankOperationRepository(t)
	bankGateway := mocks.NewMockBankGateway(t)
	outboxRepo := mocks.NewMockOutboxRepository(t)
	transactor := mocks.NewMockTransactor(t)

	svc := services.NewAccountService(
		accountRepo, accountOpRepo, bankAccountRepo, bankOpRepo,
		bankGateway, outboxRepo, transactor,
	)
	return svc, accountRepo, accountOpRepo, bankAccountRepo, bankOpRepo, bankGateway, outboxRepo, transactor
}

func runWithTx(transactor *mocks.MockTransactor) {
	transactor.On("WithTx", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		fn := args.Get(1).(func(context.Context) error)
		_ = fn(args.Get(0).(context.Context))
	})
}

func runWithTxOnce(transactor *mocks.MockTransactor) {
	transactor.On("WithTx", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		fn := args.Get(1).(func(context.Context) error)
		_ = fn(args.Get(0).(context.Context))
	}).Once()
}

func validCompanyID() uuid.UUID { return uuid.New() }

func validBankOpRequest(accountID, bankAccountID, initiatorId uuid.UUID, amount int64) *services.BankOperationInput {
	return &services.BankOperationInput{
		AccountID:      accountID,
		BankAccountID:  bankAccountID,
		InitiatorID:    initiatorId,
		Amount:         amount,
		IdempotencyKey: uuid.New().String(),
	}
}

func makeBankAccount(t *testing.T, currency entities.Currency, settlementAccount string) *entities.BankAccount {
	t.Helper()
	ba, err := entities.NewBankAccount(validCompanyID(), "Test Bank", testBIC, settlementAccount, currency)
	assert.NoError(t, err)
	return ba
}

func TestAccountService_CreateAccount(t *testing.T) {
	ctx := context.Background()
	companyID := validCompanyID()

	t.Run("Success", func(t *testing.T) {
		svc, accountRepo, _, _, _, _, _, _ := setupAccountService(t)

		accountRepo.On("Save", mock.Anything, mock.MatchedBy(func(a *entities.Account) bool {
			return a != nil && a.CompanyId() == companyID && a.Currency() == entities.RUB && a.Status() == entities.ActiveStatus
		})).Return(nil)

		acc, err := svc.CreateAccount(ctx, companyID, entities.RUB)
		assert.NoError(t, err)
		assert.NotNil(t, acc)
		assert.Equal(t, companyID, acc.CompanyId())
		assert.Equal(t, entities.RUB, acc.Currency())
		assert.Equal(t, int64(0), acc.Balance())
		assert.Equal(t, entities.ActiveStatus, acc.Status())
	})

	t.Run("Repository save error", func(t *testing.T) {
		svc, accountRepo, _, _, _, _, _, _ := setupAccountService(t)
		accountRepo.On("Save", mock.Anything, mock.Anything).Return(errors.New("db error"))

		acc, err := svc.CreateAccount(ctx, companyID, entities.RUB)
		assert.Error(t, err)
		assert.Nil(t, acc)
		assert.Contains(t, err.Error(), "db error")
	})

	t.Run("Invalid currency", func(t *testing.T) {
		svc, _, _, _, _, _, _, _ := setupAccountService(t)
		acc, err := svc.CreateAccount(ctx, companyID, "INVALID")
		assert.Error(t, err)
		assert.Nil(t, acc)
		assert.Contains(t, err.Error(), "currency must be valid")
	})
}

func TestAccountService_SetAccountInActive(t *testing.T) {
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		svc, accountRepo, accountOpRepo, _, _, _, _, transactor := setupAccountService(t)
		acc, _ := entities.NewAccount(validCompanyID(), 1000, entities.USD, entities.ActiveStatus)
		accountID := acc.AccountId()

		accountRepo.On("GetById", mock.Anything, accountID).Return(acc, nil)
		accountOpRepo.On("HasPendingByAccountId", mock.Anything, accountID).Return(false, nil)
		runWithTx(transactor)
		accountRepo.On("UpdateStatus", mock.Anything, accountID, entities.InactiveStatus).Return(nil)

		result, err := svc.SetAccountInActive(ctx, accountID)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, entities.InactiveStatus, result.Status())
	})

	t.Run("Account not found", func(t *testing.T) {
		svc, accountRepo, _, _, _, _, _, _ := setupAccountService(t)
		accountID := uuid.New()
		accountRepo.On("GetById", mock.Anything, accountID).Return(nil, errors.New("not found"))

		result, err := svc.SetAccountInActive(ctx, accountID)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("Has pending operations", func(t *testing.T) {
		svc, accountRepo, accountOpRepo, _, _, _, _, _ := setupAccountService(t)
		acc, _ := entities.NewAccount(validCompanyID(), 1000, entities.USD, entities.ActiveStatus)
		accountID := acc.AccountId()

		accountRepo.On("GetById", mock.Anything, accountID).Return(acc, nil)
		accountOpRepo.On("HasPendingByAccountId", mock.Anything, accountID).Return(true, nil)

		result, err := svc.SetAccountInActive(ctx, accountID)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, services.ErrAccountHasPendingOperations)
	})

	t.Run("Transaction error", func(t *testing.T) {
		svc, accountRepo, accountOpRepo, _, _, _, _, transactor := setupAccountService(t)
		acc, _ := entities.NewAccount(validCompanyID(), 1000, entities.USD, entities.ActiveStatus)
		accountID := acc.AccountId()

		accountRepo.On("GetById", mock.Anything, accountID).Return(acc, nil)
		accountOpRepo.On("HasPendingByAccountId", mock.Anything, accountID).Return(false, nil)
		transactor.On("WithTx", mock.Anything, mock.Anything).Return(errors.New("tx failed"))

		result, err := svc.SetAccountInActive(ctx, accountID)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "tx failed")
	})
}

func TestAccountService_GetBalance(t *testing.T) {
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		svc, accountRepo, _, _, _, _, _, _ := setupAccountService(t)
		acc, _ := entities.NewAccount(validCompanyID(), 5000, entities.USD, entities.ActiveStatus)
		accountRepo.On("GetById", mock.Anything, acc.AccountId()).Return(acc, nil)

		balance, err := svc.GetBalance(ctx, acc.AccountId())
		assert.NoError(t, err)
		assert.Equal(t, int64(5000), balance)
	})

	t.Run("Account not found", func(t *testing.T) {
		svc, accountRepo, _, _, _, _, _, _ := setupAccountService(t)
		accountID := uuid.New()
		accountRepo.On("GetById", mock.Anything, accountID).Return(nil, errors.New("not found"))

		balance, err := svc.GetBalance(ctx, accountID)
		assert.Error(t, err)
		assert.Equal(t, int64(-1), balance)
	})
}

func TestAccountService_LinkBankAccount(t *testing.T) {
	ctx := context.Background()
	companyID := validCompanyID()

	t.Run("Success", func(t *testing.T) {
		svc, accountRepo, _, bankAccountRepo, _, _, _, _ := setupAccountService(t)
		acc, _ := entities.NewAccount(companyID, 1000, entities.USD, entities.ActiveStatus)

		req := services.LinkBankInput{
			AccountID:         acc.AccountId(),
			CompanyID:         companyID,
			Name:              "Test Bank",
			Bic:               testBIC,
			SettlementAccount: testAccountUSD,
			Currency:          entities.USD,
		}

		accountRepo.On("GetById", mock.Anything, acc.AccountId()).Return(acc, nil)
		bankAccountRepo.On("Save", mock.Anything, mock.MatchedBy(func(ba *entities.BankAccount) bool {
			return ba != nil && ba.CompanyId() == companyID && ba.Name() == "Test Bank"
		})).Return(nil)

		resp, err := svc.LinkBankAccount(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, acc.CompanyId(), resp.CompanyId())
		assert.Equal(t, "Test Bank", resp.Name())
	})

	t.Run("Account not found", func(t *testing.T) {
		svc, accountRepo, _, _, _, _, _, _ := setupAccountService(t)
		accountID := uuid.New()
		accountRepo.On("GetById", mock.Anything, accountID).Return(nil, errors.New("not found"))

		resp, err := svc.LinkBankAccount(ctx, services.LinkBankInput{AccountID: accountID, CompanyID: companyID})
		assert.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("Different companies", func(t *testing.T) {
		svc, accountRepo, _, _, _, _, _, _ := setupAccountService(t)
		acc, _ := entities.NewAccount(validCompanyID(), 1000, entities.USD, entities.ActiveStatus)
		accountRepo.On("GetById", mock.Anything, acc.AccountId()).Return(acc, nil)

		resp, err := svc.LinkBankAccount(ctx, services.LinkBankInput{
			AccountID: acc.AccountId(),
			CompanyID: uuid.New(),
		})
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.ErrorIs(t, err, services.ErrDifferentCompanies)
	})

	t.Run("Invalid BIC", func(t *testing.T) {
		svc, accountRepo, _, _, _, _, _, _ := setupAccountService(t)
		acc, _ := entities.NewAccount(companyID, 1000, entities.USD, entities.ActiveStatus)
		accountRepo.On("GetById", mock.Anything, acc.AccountId()).Return(acc, nil)

		resp, err := svc.LinkBankAccount(ctx, services.LinkBankInput{
			AccountID:         acc.AccountId(),
			CompanyID:         companyID,
			Name:              "Bank",
			Bic:               "INVALID",
			SettlementAccount: testAccountUSD,
			Currency:          entities.USD,
		})
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "invalid bic format")
	})

	t.Run("Currency mismatch", func(t *testing.T) {
		svc, accountRepo, _, _, _, _, _, _ := setupAccountService(t)
		acc, _ := entities.NewAccount(companyID, 1000, entities.USD, entities.ActiveStatus)
		accountRepo.On("GetById", mock.Anything, acc.AccountId()).Return(acc, nil)

		resp, err := svc.LinkBankAccount(ctx, services.LinkBankInput{
			AccountID:         acc.AccountId(),
			CompanyID:         companyID,
			Name:              "Bank",
			Bic:               testBIC,
			SettlementAccount: testAccountRUB,
			Currency:          entities.USD,
		})
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "settlement account currency does not match")
	})
}

func TestAccountService_ReserveWithdraw(t *testing.T) {
	ctx := context.Background()
	txID := uuid.New()
	amount := int64(1000)

	t.Run("Success", func(t *testing.T) {
		svc, accountRepo, accountOpRepo, _, _, _, _, transactor := setupAccountService(t)
		acc, _ := entities.NewAccount(validCompanyID(), 5000, entities.USD, entities.ActiveStatus)

		accountRepo.On("GetById", mock.Anything, acc.AccountId()).Return(acc, nil)
		accountOpRepo.On("GetByTransactionIdAndType", mock.Anything, txID, string(entities.Withdrawal)).Return(nil, nil)
		runWithTx(transactor)
		accountRepo.On("UpdateBalance", mock.Anything, acc.AccountId(), -amount).Return(nil)
		accountOpRepo.On("Save", mock.Anything, mock.MatchedBy(func(ao *entities.AccountOperation) bool {
			return ao != nil &&
				ao.AccountId() == acc.AccountId() &&
				ao.TransactionId() == txID &&
				ao.OperationType() == entities.Withdrawal &&
				ao.OperationStatus() == entities.PendingStatus &&
				ao.Amount() == amount
		})).Return(nil)

		op, err := svc.ReserveWithdraw(ctx, acc.AccountId(), txID, amount)
		assert.NoError(t, err)
		assert.NotNil(t, op)
		assert.Equal(t, entities.PendingStatus, op.OperationStatus())
	})

	t.Run("Idempotency", func(t *testing.T) {
		svc, accountRepo, accountOpRepo, _, _, _, _, _ := setupAccountService(t)
		acc, _ := entities.NewAccount(validCompanyID(), 5000, entities.USD, entities.ActiveStatus)
		existingOp, _ := entities.NewAccountOperation(acc.AccountId(), txID, entities.Withdrawal, entities.PendingStatus, amount)

		accountRepo.On("GetById", mock.Anything, acc.AccountId()).Return(acc, nil)
		accountOpRepo.On("GetByTransactionIdAndType", mock.Anything, txID, string(entities.Withdrawal)).Return(existingOp, nil)

		op, err := svc.ReserveWithdraw(ctx, acc.AccountId(), txID, amount)
		assert.NoError(t, err)
		assert.Equal(t, existingOp, op)
	})

	t.Run("Not enough funds", func(t *testing.T) {
		svc, accountRepo, accountOpRepo, _, _, _, _, _ := setupAccountService(t)
		acc, _ := entities.NewAccount(validCompanyID(), 500, entities.USD, entities.ActiveStatus)

		accountRepo.On("GetById", mock.Anything, acc.AccountId()).Return(acc, nil)
		accountOpRepo.On("GetByTransactionIdAndType", mock.Anything, txID, string(entities.Withdrawal)).Return(nil, nil)

		op, err := svc.ReserveWithdraw(ctx, acc.AccountId(), txID, amount)
		assert.Error(t, err)
		assert.Nil(t, op)
		assert.ErrorIs(t, err, services.ErrNotEnoughFunds)
	})

	t.Run("Account not found", func(t *testing.T) {
		svc, accountRepo, _, _, _, _, _, _ := setupAccountService(t)
		accountID := uuid.New()
		accountRepo.On("GetById", mock.Anything, accountID).Return(nil, errors.New("not found"))

		op, err := svc.ReserveWithdraw(ctx, accountID, txID, amount)
		assert.Error(t, err)
		assert.Nil(t, op)
	})
}

func TestAccountService_ReserveDeposit(t *testing.T) {
	ctx := context.Background()
	txID := uuid.New()
	amount := int64(1000)

	t.Run("Success", func(t *testing.T) {
		svc, accountRepo, accountOpRepo, _, _, _, _, transactor := setupAccountService(t)
		acc, _ := entities.NewAccount(validCompanyID(), 0, entities.USD, entities.ActiveStatus)

		accountRepo.On("GetById", mock.Anything, acc.AccountId()).Return(acc, nil)
		accountOpRepo.On("GetByTransactionIdAndType", mock.Anything, txID, string(entities.Deposit)).Return(nil, nil)
		runWithTx(transactor)
		accountRepo.On("UpdateBalance", mock.Anything, acc.AccountId(), amount).Return(nil)
		accountOpRepo.On("Save", mock.Anything, mock.MatchedBy(func(ao *entities.AccountOperation) bool {
			return ao != nil &&
				ao.AccountId() == acc.AccountId() &&
				ao.OperationType() == entities.Deposit &&
				ao.Amount() == amount
		})).Return(nil)

		op, err := svc.ReserveDeposit(ctx, acc.AccountId(), txID, amount)
		assert.NoError(t, err)
		assert.NotNil(t, op)
		assert.Equal(t, entities.PendingStatus, op.OperationStatus())
	})

	t.Run("Idempotency", func(t *testing.T) {
		svc, accountRepo, accountOpRepo, _, _, _, _, _ := setupAccountService(t)
		acc, _ := entities.NewAccount(validCompanyID(), 0, entities.USD, entities.ActiveStatus)
		existingOp, _ := entities.NewAccountOperation(acc.AccountId(), txID, entities.Deposit, entities.PendingStatus, amount)

		accountRepo.On("GetById", mock.Anything, acc.AccountId()).Return(acc, nil)
		accountOpRepo.On("GetByTransactionIdAndType", mock.Anything, txID, string(entities.Deposit)).Return(existingOp, nil)

		op, err := svc.ReserveDeposit(ctx, acc.AccountId(), txID, amount)
		assert.NoError(t, err)
		assert.Equal(t, existingOp, op)
	})
}

func TestAccountService_ConfirmOperation(t *testing.T) {
	ctx := context.Background()
	txID := uuid.New()

	t.Run("Success", func(t *testing.T) {
		svc, _, accountOpRepo, _, _, _, _, _ := setupAccountService(t)
		accountOpRepo.On("ConfirmAllByTransactionId", mock.Anything, txID).Return(nil)

		err := svc.ConfirmOperation(ctx, txID)
		assert.NoError(t, err)
	})

	t.Run("Repository error", func(t *testing.T) {
		svc, _, accountOpRepo, _, _, _, _, _ := setupAccountService(t)
		accountOpRepo.On("ConfirmAllByTransactionId", mock.Anything, txID).Return(errors.New("db error"))

		err := svc.ConfirmOperation(ctx, txID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "db error")
	})
}

func TestAccountService_CancelOperation(t *testing.T) {
	ctx := context.Background()
	txID := uuid.New()

	t.Run("Success - rollback deposit", func(t *testing.T) {
		svc, accountRepo, accountOpRepo, _, _, _, _, transactor := setupAccountService(t)
		acc, _ := entities.NewAccount(validCompanyID(), 1000, entities.USD, entities.ActiveStatus)
		accOp, _ := entities.NewAccountOperation(acc.AccountId(), txID, entities.Deposit, entities.PendingStatus, 500)

		accountOpRepo.On("GetByTransactionId", mock.Anything, txID).Return([]*entities.AccountOperation{accOp}, nil)
		runWithTx(transactor)
		accountRepo.On("GetById", mock.Anything, acc.AccountId()).Return(acc, nil)
		accountRepo.On("UpdateBalance", mock.Anything, acc.AccountId(), -int64(500)).Return(nil)
		accountOpRepo.On("UpdateStatus", mock.Anything, accOp.AccountOperationId(), string(entities.FailedStatus)).Return(nil)

		err := svc.CancelOperation(ctx, txID)
		assert.NoError(t, err)
	})

	t.Run("Success - rollback withdrawal", func(t *testing.T) {
		svc, accountRepo, accountOpRepo, _, _, _, _, transactor := setupAccountService(t)
		acc, _ := entities.NewAccount(validCompanyID(), 1000, entities.USD, entities.ActiveStatus)
		accOp, _ := entities.NewAccountOperation(acc.AccountId(), txID, entities.Withdrawal, entities.PendingStatus, 500)

		accountOpRepo.On("GetByTransactionId", mock.Anything, txID).Return([]*entities.AccountOperation{accOp}, nil)
		runWithTx(transactor)
		accountRepo.On("GetById", mock.Anything, acc.AccountId()).Return(acc, nil)
		accountRepo.On("UpdateBalance", mock.Anything, acc.AccountId(), int64(500)).Return(nil)
		accountOpRepo.On("UpdateStatus", mock.Anything, accOp.AccountOperationId(), string(entities.FailedStatus)).Return(nil)

		err := svc.CancelOperation(ctx, txID)
		assert.NoError(t, err)
	})

	t.Run("Skip already failed operations", func(t *testing.T) {
		svc, _, accountOpRepo, _, _, _, _, transactor := setupAccountService(t)
		accOp, _ := entities.NewAccountOperation(uuid.New(), txID, entities.Deposit, entities.PendingStatus, 500)
		_ = accOp.UpdateOperationStatus(entities.FailedStatus)

		accountOpRepo.On("GetByTransactionId", mock.Anything, txID).Return([]*entities.AccountOperation{accOp}, nil)
		runWithTx(transactor)

		err := svc.CancelOperation(ctx, txID)
		assert.NoError(t, err)
	})

	t.Run("GetByTransactionId error", func(t *testing.T) {
		svc, _, accountOpRepo, _, _, _, _, _ := setupAccountService(t)
		accountOpRepo.On("GetByTransactionId", mock.Anything, txID).Return(nil, errors.New("db error"))

		err := svc.CancelOperation(ctx, txID)
		assert.Error(t, err)
	})
}

func TestAccountService_MakeBankDeposit(t *testing.T) {
	ctx := context.Background()
	amount := int64(2000)

	t.Run("Success", func(t *testing.T) {
		svc, accountRepo, _, bankAccountRepo, bankOpRepo, bankGateway, outboxRepo, transactor := setupAccountService(t)
		acc, _ := entities.NewAccount(validCompanyID(), 1000, entities.USD, entities.ActiveStatus)
		bankAcc := makeBankAccount(t, entities.USD, testAccountUSD)
		userId := uuid.New()
		req := validBankOpRequest(acc.AccountId(), bankAcc.BankAccountId(), userId, amount)

		bankOpRepo.On("GetByIdempotencyKey", mock.Anything, req.IdempotencyKey).Return(nil, nil)
		accountRepo.On("GetById", mock.Anything, acc.AccountId()).Return(acc, nil)
		bankAccountRepo.On("GetById", mock.Anything, bankAcc.BankAccountId()).Return(bankAcc, nil)
		bankGateway.On("Deposit", mock.Anything, bankAcc, amount, req.IdempotencyKey).Return("ext_123", nil)
		runWithTx(transactor)
		accountRepo.On("UpdateBalance", mock.Anything, acc.AccountId(), amount).Return(nil)
		bankOpRepo.On("Save", mock.Anything, mock.MatchedBy(func(bo *entities.BankOperation) bool {
			return bo != nil &&
				bo.OperationType() == entities.Deposit &&
				bo.OperationStatus() == entities.SuccessStatus &&
				bo.Amount() == amount
		})).Return(nil)
		outboxRepo.On("Save", mock.Anything, mock.MatchedBy(func(e *outbox.Event) bool {
			return e != nil && e.AggregateType == "bank_operation"
		})).Return(nil)

		bo, err := svc.MakeBankDeposit(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, bo)
		assert.Equal(t, entities.SuccessStatus, bo.OperationStatus())
		assert.Equal(t, "ext_123", bo.ExternalId())
	})

	t.Run("Idempotency", func(t *testing.T) {
		svc, _, _, _, bankOpRepo, _, _, _ := setupAccountService(t)
		acc, _ := entities.NewAccount(validCompanyID(), 1000, entities.USD, entities.ActiveStatus)
		req := validBankOpRequest(acc.AccountId(), uuid.New(), uuid.New(), amount)
		existingBo, _ := entities.NewBankOperation(acc.AccountId(), uuid.New(), uuid.New(), entities.Deposit, entities.SuccessStatus, amount, req.IdempotencyKey, "ext_123")

		bankOpRepo.On("GetByIdempotencyKey", mock.Anything, req.IdempotencyKey).Return(existingBo, nil)

		bo, err := svc.MakeBankDeposit(ctx, req)
		assert.NoError(t, err)
		assert.Equal(t, existingBo, bo)
	})

	t.Run("Bank gateway error", func(t *testing.T) {
		svc, accountRepo, _, bankAccountRepo, bankOpRepo, bankGateway, _, _ := setupAccountService(t)
		acc, _ := entities.NewAccount(validCompanyID(), 1000, entities.USD, entities.ActiveStatus)
		bankAcc := makeBankAccount(t, entities.USD, testAccountUSD)
		req := validBankOpRequest(acc.AccountId(), bankAcc.BankAccountId(), uuid.New(), amount)

		bankOpRepo.On("GetByIdempotencyKey", mock.Anything, req.IdempotencyKey).Return(nil, nil)
		accountRepo.On("GetById", mock.Anything, acc.AccountId()).Return(acc, nil)
		bankAccountRepo.On("GetById", mock.Anything, bankAcc.BankAccountId()).Return(bankAcc, nil)
		bankGateway.On("Deposit", mock.Anything, bankAcc, amount, req.IdempotencyKey).Return("", errors.New("gateway timeout"))

		bo, err := svc.MakeBankDeposit(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, bo)
		assert.Contains(t, err.Error(), "gateway timeout")
	})
}

func TestAccountService_MakeBankWithdrawal(t *testing.T) {
	ctx := context.Background()
	amount := int64(2000)

	t.Run("Success", func(t *testing.T) {
		svc, accountRepo, _, bankAccountRepo, bankOpRepo, bankGateway, outboxRepo, transactor := setupAccountService(t)
		acc, _ := entities.NewAccount(validCompanyID(), 5000, entities.USD, entities.ActiveStatus)
		bankAcc := makeBankAccount(t, entities.USD, testAccountUSD)
		req := validBankOpRequest(acc.AccountId(), bankAcc.BankAccountId(), uuid.New(), amount)

		bankOpRepo.On("GetByIdempotencyKey", mock.Anything, req.IdempotencyKey).Return(nil, nil)
		accountRepo.On("GetById", mock.Anything, acc.AccountId()).Return(acc, nil)
		bankAccountRepo.On("GetById", mock.Anything, bankAcc.BankAccountId()).Return(bankAcc, nil)
		runWithTxOnce(transactor)
		accountRepo.On("UpdateBalance", mock.Anything, acc.AccountId(), -amount).Return(nil)
		bankOpRepo.On("Save", mock.Anything, mock.MatchedBy(func(bo *entities.BankOperation) bool {
			return bo != nil &&
				bo.OperationType() == entities.Withdrawal &&
				bo.OperationStatus() == entities.PendingStatus
		})).Return(nil)
		bankGateway.On("Withdraw", mock.Anything, bankAcc, amount, req.IdempotencyKey).Return("ext_456", nil)
		runWithTxOnce(transactor)
		bankOpRepo.On("UpdateStatusAndExternalID", mock.Anything, mock.Anything, string(entities.SuccessStatus), "ext_456").Return(nil)
		outboxRepo.On("Save", mock.Anything, mock.MatchedBy(func(e *outbox.Event) bool {
			return e != nil && e.AggregateType == "bank_operation"
		})).Return(nil)

		bo, err := svc.MakeBankWithdrawal(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, bo)
		assert.Equal(t, entities.SuccessStatus, bo.OperationStatus())
		assert.Equal(t, "ext_456", bo.ExternalId())
	})

	t.Run("Not enough funds", func(t *testing.T) {
		svc, accountRepo, _, bankAccountRepo, bankOpRepo, _, _, _ := setupAccountService(t)
		acc, _ := entities.NewAccount(validCompanyID(), 500, entities.USD, entities.ActiveStatus)
		bankAcc := makeBankAccount(t, entities.USD, testAccountUSD)
		req := validBankOpRequest(acc.AccountId(), bankAcc.BankAccountId(), uuid.New(), amount)

		bankOpRepo.On("GetByIdempotencyKey", mock.Anything, req.IdempotencyKey).Return(nil, nil)
		accountRepo.On("GetById", mock.Anything, acc.AccountId()).Return(acc, nil)
		bankAccountRepo.On("GetById", mock.Anything, bankAcc.BankAccountId()).Return(bankAcc, nil)

		bo, err := svc.MakeBankWithdrawal(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, bo)
		assert.ErrorIs(t, err, services.ErrNotEnoughFunds)
	})

	t.Run("Gateway error - compensation", func(t *testing.T) {
		svc, accountRepo, _, bankAccountRepo, bankOpRepo, bankGateway, outboxRepo, transactor := setupAccountService(t)
		acc, _ := entities.NewAccount(validCompanyID(), 5000, entities.USD, entities.ActiveStatus)
		bankAcc := makeBankAccount(t, entities.USD, testAccountUSD)
		req := validBankOpRequest(acc.AccountId(), bankAcc.BankAccountId(), uuid.New(), amount)

		bankOpRepo.On("GetByIdempotencyKey", mock.Anything, req.IdempotencyKey).Return(nil, nil)
		accountRepo.On("GetById", mock.Anything, acc.AccountId()).Return(acc, nil)
		bankAccountRepo.On("GetById", mock.Anything, bankAcc.BankAccountId()).Return(bankAcc, nil)
		runWithTxOnce(transactor)
		accountRepo.On("UpdateBalance", mock.Anything, acc.AccountId(), -amount).Return(nil)
		bankOpRepo.On("Save", mock.Anything, mock.Anything).Return(nil)
		bankGateway.On("Withdraw", mock.Anything, bankAcc, amount, req.IdempotencyKey).Return("", errors.New("gateway error"))
		runWithTxOnce(transactor)
		accountRepo.On("UpdateBalance", mock.Anything, acc.AccountId(), amount).Return(nil)
		bankOpRepo.On("UpdateStatusAndExternalID", mock.Anything, mock.Anything, string(entities.FailedStatus), "").Return(nil)
		outboxRepo.On("Save", mock.Anything, mock.Anything).Return(nil)

		bo, err := svc.MakeBankWithdrawal(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, bo)
		assert.Contains(t, err.Error(), "gateway error")
	})

	t.Run("Idempotency", func(t *testing.T) {
		svc, _, _, _, bankOpRepo, _, _, _ := setupAccountService(t)
		acc, _ := entities.NewAccount(validCompanyID(), 5000, entities.USD, entities.ActiveStatus)
		req := validBankOpRequest(acc.AccountId(), uuid.New(), uuid.New(), amount)
		existingBo, _ := entities.NewBankOperation(acc.AccountId(), uuid.New(), uuid.New(), entities.Withdrawal, entities.SuccessStatus, amount, req.IdempotencyKey, "ext_456")

		bankOpRepo.On("GetByIdempotencyKey", mock.Anything, req.IdempotencyKey).Return(existingBo, nil)

		bo, err := svc.MakeBankWithdrawal(ctx, req)
		assert.NoError(t, err)
		assert.Equal(t, existingBo, bo)
	})
}

func TestAccountService_GetAccounts(t *testing.T) {
	ctx := context.Background()
	companyID := validCompanyID()

	t.Run("Success", func(t *testing.T) {
		svc, accountRepo, _, _, _, _, _, _ := setupAccountService(t)
		acc1, _ := entities.NewAccount(companyID, 1000, entities.USD, entities.ActiveStatus)
		acc2, _ := entities.NewAccount(companyID, 2000, entities.EUR, entities.ActiveStatus)
		accountRepo.On("GetByCompanyId", mock.Anything, companyID).Return([]*entities.Account{acc1, acc2}, nil)

		accounts, err := svc.GetAccounts(ctx, companyID)
		assert.NoError(t, err)
		assert.Len(t, accounts, 2)
	})

	t.Run("Repository error", func(t *testing.T) {
		svc, accountRepo, _, _, _, _, _, _ := setupAccountService(t)
		accountRepo.On("GetByCompanyId", mock.Anything, companyID).Return(nil, errors.New("db error"))

		accounts, err := svc.GetAccounts(ctx, companyID)
		assert.Error(t, err)
		assert.Nil(t, accounts)
	})
}

func TestAccountService_GetBankAccounts(t *testing.T) {
	ctx := context.Background()
	companyID := validCompanyID()

	t.Run("Success", func(t *testing.T) {
		svc, _, _, bankAccountRepo, _, _, _, _ := setupAccountService(t)
		ba1 := makeBankAccount(t, entities.USD, testAccountUSD)
		ba2 := makeBankAccount(t, entities.EUR, testAccountEUR)
		bankAccountRepo.On("GetByCompanyId", mock.Anything, companyID).Return([]*entities.BankAccount{ba1, ba2}, nil)

		accounts, err := svc.GetBankAccounts(ctx, companyID)
		assert.NoError(t, err)
		assert.Len(t, accounts, 2)
	})

	t.Run("Repository error", func(t *testing.T) {
		svc, _, _, bankAccountRepo, _, _, _, _ := setupAccountService(t)
		bankAccountRepo.On("GetByCompanyId", mock.Anything, companyID).Return(nil, errors.New("db error"))

		accounts, err := svc.GetBankAccounts(ctx, companyID)
		assert.Error(t, err)
		assert.Nil(t, accounts)
	})
}

func TestAccountService_GetCompanyIdByAccountId(t *testing.T) {
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		svc, accountRepo, _, _, _, _, _, _ := setupAccountService(t)
		acc, _ := entities.NewAccount(validCompanyID(), 1000, entities.USD, entities.ActiveStatus)
		accountRepo.On("GetById", mock.Anything, acc.AccountId()).Return(acc, nil)

		result, err := svc.GetCompanyIdByAccountId(ctx, acc.AccountId())
		assert.NoError(t, err)
		assert.Equal(t, acc.CompanyId(), result)
	})

	t.Run("Account not found", func(t *testing.T) {
		svc, accountRepo, _, _, _, _, _, _ := setupAccountService(t)
		accountID := uuid.New()
		accountRepo.On("GetById", mock.Anything, accountID).Return(nil, errors.New("not found"))

		result, err := svc.GetCompanyIdByAccountId(ctx, accountID)
		assert.Error(t, err)
		assert.Equal(t, uuid.Nil, result)
	})
}
