package services

import (
	"accounts/internal/domain/entities"
	"accounts/internal/dto/request"
	"accounts/internal/dto/response"
	"context"
	"errors"
	"fmt"
	"shared/outbox"

	"github.com/google/uuid"
)

var (
	ErrNotEnoughFunds              = errors.New("not enough funds")
	ErrDifferentCompanies          = errors.New("different companies")
	ErrAccountHasPendingOperations = errors.New("account has pending operations")
)

type AccountRepository interface {
	Save(ctx context.Context, account *entities.Account) error
	GetById(ctx context.Context, id uuid.UUID) (*entities.Account, error)
	GetByCompanyId(ctx context.Context, companyId uuid.UUID) ([]*entities.Account, error)
	UpdateBalance(ctx context.Context, accountId uuid.UUID, amount int64) error
	UpdateStatus(ctx context.Context, accountId uuid.UUID, status entities.AccountStatus) error
}

type AccountOperationRepository interface {
	Save(ctx context.Context, accountOp *entities.AccountOperation) error
	GetByTransactionIdAndType(ctx context.Context, transactionId uuid.UUID, operationType string) (*entities.AccountOperation, error)
	GetByTransactionId(ctx context.Context, transactionId uuid.UUID) ([]*entities.AccountOperation, error)
	UpdateStatus(ctx context.Context, operation uuid.UUID, status string) error
	ConfirmAllByTransactionId(ctx context.Context, transactionId uuid.UUID) error
	HasPendingByAccountId(ctx context.Context, accountId uuid.UUID) (bool, error)
}

type BankAccountRepository interface {
	Save(ctx context.Context, account *entities.BankAccount) error
	GetById(ctx context.Context, id uuid.UUID) (*entities.BankAccount, error)
	GetByCompanyId(ctx context.Context, companyId uuid.UUID) ([]*entities.BankAccount, error)
}

type BankOperationRepository interface {
	Save(ctx context.Context, operation *entities.BankOperation) error
	UpdateStatusAndExternalID(ctx context.Context, operationId uuid.UUID, status string, externalID string) error
	GetByIdempotencyKey(ctx context.Context, idempotencyKey string) (*entities.BankOperation, error)
}

type BankGateway interface {
	Deposit(ctx context.Context, bankAccount *entities.BankAccount, amount int64, idempotencyKey string) (string, error)
	Withdraw(ctx context.Context, bankAccount *entities.BankAccount, amount int64, idempotencyKey string) (string, error)
}

type OutboxRepository interface {
	Save(ctx context.Context, event *outbox.Event) error
}

type Transactor interface {
	WithTx(ctx context.Context, fn func(context.Context) error) error
}

type AccountService struct {
	accountRepository          AccountRepository
	accountOperationRepository AccountOperationRepository
	bankAccountRepository      BankAccountRepository
	bankOperationRepository    BankOperationRepository
	bankGateway                BankGateway
	outboxRepo                 OutboxRepository
	transactor                 Transactor
}

func NewAccountService(
	accountRepo AccountRepository,
	accountOperationRepo AccountOperationRepository,
	bankAccountRepository BankAccountRepository,
	bankOperationRepository BankOperationRepository,
	bankGateway BankGateway,
	outboxRepo OutboxRepository,
	transactor Transactor,
) *AccountService {
	return &AccountService{
		accountRepository:          accountRepo,
		accountOperationRepository: accountOperationRepo,
		bankAccountRepository:      bankAccountRepository,
		bankOperationRepository:    bankOperationRepository,
		bankGateway:                bankGateway,
		outboxRepo:                 outboxRepo,
		transactor:                 transactor,
	}
}

func (a *AccountService) CreateAccount(ctx context.Context, companyId uuid.UUID, currency entities.Currency) (*entities.Account, error) {
	op := "AccountService.CreateAccount"

	acc, err := entities.NewAccount(companyId, 0, currency, entities.ActiveStatus)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if err = a.accountRepository.Save(ctx, acc); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return acc, nil
}

func (a *AccountService) SetAccountInActive(ctx context.Context, accountId uuid.UUID) (*entities.Account, error) {
	op := "AccountService.SetAccountInactive"

	acc, err := a.accountRepository.GetById(ctx, accountId)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	hasPending, err := a.accountOperationRepository.HasPendingByAccountId(ctx, accountId)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	if hasPending {
		return nil, fmt.Errorf("%s: %w", op, ErrAccountHasPendingOperations)
	}
	err = a.transactor.WithTx(ctx, func(txCtx context.Context) error {
		if err = acc.UpdateStatus(entities.InactiveStatus); err != nil {
			return err
		}

		if err = a.accountRepository.UpdateStatus(ctx, accountId, entities.InactiveStatus); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return acc, nil
}

func (a *AccountService) GetBalance(ctx context.Context, accountId uuid.UUID) (int64, error) {
	op := "AccountService.GetBalance"
	acc, err := a.accountRepository.GetById(ctx, accountId)
	if err != nil {
		return -1, fmt.Errorf("%s: %w", op, err)
	}

	return acc.Balance(), nil
}

func (a *AccountService) LinkBankAccount(ctx context.Context, request request.LinkBankRequest) (*response.LinkBankResponse, error) {
	op := "AccountService.LinkBankAccount"

	acc, err := a.accountRepository.GetById(ctx, request.AccountID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if acc.CompanyId() != request.CompanyID {
		return nil, fmt.Errorf("%s: %w", op, ErrDifferentCompanies)
	}

	bankAcc, err := entities.NewBankAccount(request.CompanyID, request.Name, request.Bic, request.SettlementAccount, request.Currency)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if err = a.bankAccountRepository.Save(ctx, bankAcc); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &response.LinkBankResponse{
		AccountID:     acc.AccountId(),
		BankAccountID: bankAcc.BankAccountId(),
		BankName:      bankAcc.Name(),
	}, nil
}

func (a *AccountService) ReserveWithdraw(ctx context.Context, accountId uuid.UUID, txId uuid.UUID, amount int64) (*entities.AccountOperation, error) {
	op := "AccountService.ReserveWithdraw"
	acc, err := a.accountRepository.GetById(ctx, accountId)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	existing, _ := a.accountOperationRepository.GetByTransactionIdAndType(ctx, txId, string(entities.Withdrawal))
	if existing != nil {
		return existing, nil
	}

	if acc.Balance() < amount {
		return nil, fmt.Errorf("%s: %w", op, ErrNotEnoughFunds)
	}

	var accOp *entities.AccountOperation
	err = a.transactor.WithTx(ctx, func(txCtx context.Context) error {
		if err = acc.RemoveBalance(amount); err != nil {
			return err
		}

		if err = a.accountRepository.UpdateBalance(txCtx, acc.AccountId(), -amount); err != nil {
			return err
		}

		accOp, err = entities.NewAccountOperation(accountId, txId, entities.Withdrawal, entities.PendingStatus, amount)
		if err != nil {
			return err
		}

		if err = a.accountOperationRepository.Save(txCtx, accOp); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return accOp, nil
}

func (a *AccountService) ReserveDeposit(ctx context.Context, accountId uuid.UUID, txId uuid.UUID, amount int64) (*entities.AccountOperation, error) {
	op := "AccountService.ReserveDeposit"
	acc, err := a.accountRepository.GetById(ctx, accountId)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	existing, _ := a.accountOperationRepository.GetByTransactionIdAndType(ctx, txId, string(entities.Deposit))
	if existing != nil {
		return existing, nil
	}

	var accOp *entities.AccountOperation
	err = a.transactor.WithTx(ctx, func(txCtx context.Context) error {
		if err = acc.AddBalance(amount); err != nil {
			return err
		}

		if err = a.accountRepository.UpdateBalance(txCtx, acc.AccountId(), amount); err != nil {
			return err
		}

		accOp, err = entities.NewAccountOperation(accountId, txId, entities.Deposit, entities.PendingStatus, amount)
		if err != nil {
			return err
		}

		if err = a.accountOperationRepository.Save(txCtx, accOp); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return accOp, nil
}

func (a *AccountService) ConfirmOperation(ctx context.Context, txId uuid.UUID) error {
	op := "AccountService.ConfirmOperation"

	err := a.accountOperationRepository.ConfirmAllByTransactionId(ctx, txId)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (a *AccountService) CancelOperation(ctx context.Context, txId uuid.UUID) error {
	op := "AccountService.CancelOperation"
	accOps, err := a.accountOperationRepository.GetByTransactionId(ctx, txId)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	err = a.transactor.WithTx(ctx, func(txCtx context.Context) error {
		for _, accOp := range accOps {
			if accOp.OperationStatus() == entities.FailedStatus {
				continue
			}

			acc, err := a.accountRepository.GetById(txCtx, accOp.AccountId())
			if err != nil {
				return fmt.Errorf("get account: %w", err)
			}

			switch accOp.OperationType() {
			case entities.Deposit:
				if err = acc.RemoveBalance(accOp.Amount()); err != nil {
					return err
				}
				if err = a.accountRepository.UpdateBalance(txCtx, acc.AccountId(), -accOp.Amount()); err != nil {
					return err
				}
			case entities.Withdrawal:
				if err = acc.AddBalance(accOp.Amount()); err != nil {
					return err
				}
				if err = a.accountRepository.UpdateBalance(txCtx, acc.AccountId(), accOp.Amount()); err != nil {
					return err
				}
			}

			if err = accOp.UpdateOperationStatus(entities.FailedStatus); err != nil {
				return err
			}
			if err = a.accountOperationRepository.UpdateStatus(txCtx, accOp.AccountOperationId(), string(entities.FailedStatus)); err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (a *AccountService) MakeBankDeposit(ctx context.Context, request *request.BankOperationRequest) (*entities.BankOperation, error) {
	op := "AccountService.MakeBankDeposit"

	existing, _ := a.bankOperationRepository.GetByIdempotencyKey(ctx, request.IdempotencyKey)
	if existing != nil {
		return existing, nil
	}

	acc, err := a.accountRepository.GetById(ctx, request.AccountID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	bankAccount, err := a.bankAccountRepository.GetById(ctx, request.BankAccountID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	externalID, err := a.bankGateway.Deposit(ctx, bankAccount, request.Amount, request.IdempotencyKey)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	var bo *entities.BankOperation

	err = a.transactor.WithTx(ctx, func(txCtx context.Context) error {
		err = acc.AddBalance(request.Amount)
		if err != nil {
			return err
		}

		if err = a.accountRepository.UpdateBalance(txCtx, acc.AccountId(), request.Amount); err != nil {
			return err
		}

		bo, err = entities.NewBankOperation(request.AccountID, request.BankAccountID, entities.Deposit, entities.SuccessStatus, request.Amount, request.IdempotencyKey, externalID)
		if err != nil {
			return err
		}

		if err = a.bankOperationRepository.Save(txCtx, bo); err != nil {
			return err
		}

		return a.saveOutboxEvent(txCtx, bo)
	})

	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return bo, nil
}

func (a *AccountService) MakeBankWithdrawal(ctx context.Context, request *request.BankOperationRequest) (*entities.BankOperation, error) {
	op := "AccountService.MakeBankWithdrawal"

	existing, _ := a.bankOperationRepository.GetByIdempotencyKey(ctx, request.IdempotencyKey)
	if existing != nil {
		return existing, nil
	}

	acc, err := a.accountRepository.GetById(ctx, request.AccountID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	bankAccount, err := a.bankAccountRepository.GetById(ctx, request.BankAccountID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if acc.Balance() < request.Amount {
		return nil, fmt.Errorf("%s: %w", op, ErrNotEnoughFunds)
	}

	var bo *entities.BankOperation

	err = a.transactor.WithTx(ctx, func(txCtx context.Context) error {
		err = acc.RemoveBalance(request.Amount)
		if err != nil {
			return err
		}

		if err = a.accountRepository.UpdateBalance(txCtx, acc.AccountId(), -request.Amount); err != nil {
			return err
		}

		bo, err = entities.NewBankOperation(request.AccountID, request.BankAccountID, entities.Withdrawal, entities.PendingStatus, request.Amount, request.IdempotencyKey, "")
		if err != nil {
			return err
		}

		if err = a.bankOperationRepository.Save(txCtx, bo); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	externalID, err := a.bankGateway.Withdraw(ctx, bankAccount, request.Amount, request.IdempotencyKey)
	if err != nil {
		if err2 := a.compensateBankWithdrawal(ctx, acc, bo, externalID); err2 != nil {
			return nil, fmt.Errorf("%s: %w, %w", op, err, err2)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	err = a.transactor.WithTx(ctx, func(txCtx context.Context) error {
		if err = bo.UpdateOperationStatus(entities.SuccessStatus); err != nil {
			return err
		}
		if err = bo.UpdateExternalId(externalID); err != nil {
			return err
		}

		if err = a.bankOperationRepository.UpdateStatusAndExternalID(txCtx, bo.BankOperationId(), string(entities.SuccessStatus), externalID); err != nil {
			return err
		}
		return a.saveOutboxEvent(txCtx, bo)
	})

	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return bo, nil
}

func (a *AccountService) GetAccounts(ctx context.Context, companyId uuid.UUID) ([]*entities.Account, error) {
	op := "AccountService.GetAccounts"

	accounts, err := a.accountRepository.GetByCompanyId(ctx, companyId)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return accounts, nil
}

func (a *AccountService) GetBankAccounts(ctx context.Context, companyId uuid.UUID) ([]*entities.BankAccount, error) {
	op := "AccountService.GetBankAccounts"

	accounts, err := a.bankAccountRepository.GetByCompanyId(ctx, companyId)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return accounts, nil
}

func (a *AccountService) GetCompanyIdByAccountId(ctx context.Context, accountId uuid.UUID) (uuid.UUID, error) {
	op := "AccountService.GetCompanyIdByAccountId"

	account, err := a.accountRepository.GetById(ctx, accountId)
	if err != nil {
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	return account.CompanyId(), nil
}

func (a *AccountService) compensateBankWithdrawal(ctx context.Context, acc *entities.Account, bo *entities.BankOperation, externalID string) error {
	op := "AccountService.compensateWithdrawal"

	return a.transactor.WithTx(ctx, func(txCtx context.Context) error {
		if err := acc.AddBalance(bo.Amount()); err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}

		if err := a.accountRepository.UpdateBalance(txCtx, acc.AccountId(), bo.Amount()); err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}

		if err := bo.UpdateOperationStatus(entities.FailedStatus); err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}

		if externalID != "" {
			_ = bo.UpdateExternalId(externalID)
		}

		if err := a.bankOperationRepository.UpdateStatusAndExternalID(txCtx, bo.BankOperationId(), string(entities.FailedStatus), externalID); err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}

		if err := a.saveOutboxEvent(txCtx, bo); err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
		return nil
	})
}

func (a *AccountService) saveOutboxEvent(ctx context.Context, o outbox.Outboxable) error {
	evt, err := o.ToOutboxEvent()
	if err != nil {
		return fmt.Errorf("build outbox event: %w", err)
	}
	return a.outboxRepo.Save(ctx, evt)
}
