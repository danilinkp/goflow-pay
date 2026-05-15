package services

import (
	"accounts/internal/domain"
	"accounts/internal/domain/entities"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"shared/pkg/logger/sl"
	"shared/pkg/outbox"
	"sort"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotEnoughFunds              = errors.New("not enough funds")
	ErrDifferentCompanies          = errors.New("different companies")
	ErrAccountHasPendingOperations = errors.New("account has pending operations")
	ErrNotAnyOperations            = errors.New("not any operations🤑")
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
	GetByAccountIdAndPeriod(ctx context.Context, accountId uuid.UUID, from, to time.Time) ([]*entities.AccountOperation, error)
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
	GetByAccountIdAndPeriod(ctx context.Context, accountId uuid.UUID, from, to time.Time) ([]*entities.BankOperation, error)
}

type StatementRepository interface {
	Save(ctx context.Context, statement *entities.Statement) error
	GetByAccountIdAndPeriod(ctx context.Context, accountId uuid.UUID, from, to time.Time) (*entities.Statement, error)
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
	statementRepository        StatementRepository
	bankGateway                BankGateway
	outboxRepo                 OutboxRepository
	transactor                 Transactor
	log                        *slog.Logger
}

func NewAccountService(
	accountRepo AccountRepository,
	accountOperationRepo AccountOperationRepository,
	bankAccountRepository BankAccountRepository,
	bankOperationRepository BankOperationRepository,
	statementRepository StatementRepository,
	bankGateway BankGateway,
	outboxRepo OutboxRepository,
	transactor Transactor,
	log *slog.Logger,
) *AccountService {
	return &AccountService{
		accountRepository:          accountRepo,
		accountOperationRepository: accountOperationRepo,
		bankAccountRepository:      bankAccountRepository,
		bankOperationRepository:    bankOperationRepository,
		statementRepository:        statementRepository,
		bankGateway:                bankGateway,
		outboxRepo:                 outboxRepo,
		transactor:                 transactor,
		log:                        log,
	}
}

func (a *AccountService) CreateAccount(ctx context.Context, companyId uuid.UUID, currency entities.Currency) (*entities.Account, error) {
	op := "AccountService.CreateAccount"
	start := time.Now()
	log := a.log.With(
		sl.Op(op),
		sl.EventID(),
	)
	log.Info("create account attempt")

	acc, err := entities.NewAccount(companyId, 0, currency, entities.ActiveStatus)
	if err != nil {
		log.Error("failed to create account",
			sl.ErrWithStack(err),
			sl.Duration(time.Since(start)),
		)
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if err = a.accountRepository.Save(ctx, acc); err != nil {
		log.Error("failed to save account",
			sl.ErrWithStack(err),
			sl.Duration(time.Since(start)),
		)
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("account created",
		slog.String("account_id", acc.AccountId().String()),
		sl.Duration(time.Since(start)))

	return acc, nil
}

func (a *AccountService) SetAccountInActive(ctx context.Context, accountId, companyId uuid.UUID) (*entities.Account, error) {
	op := "AccountService.SetAccountInactive"
	start := time.Now()
	log := a.log.With(
		sl.Op(op),
		sl.EventID(),
	)

	log.Info("set account inactive attempt")

	acc, err := a.accountRepository.GetById(ctx, accountId)
	if err != nil {
		log.Error("failed to get account",
			sl.ErrWithStack(err),
			sl.Duration(time.Since(start)),
		)
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if acc.CompanyId() != companyId {
		log.Warn("account does not belong to company",
			sl.ErrWithStack(ErrDifferentCompanies),
			sl.Duration(time.Since(start)),
		)
		return nil, fmt.Errorf("%s: %w", op, ErrDifferentCompanies)
	}

	hasPending, err := a.accountOperationRepository.HasPendingByAccountId(ctx, accountId)
	if err != nil {
		log.Error("failed to check if account has pending operation",
			sl.ErrWithStack(err),
			sl.Duration(time.Since(start)),
		)
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	if hasPending {
		log.Warn("account has pending operation",
			sl.ErrWithStack(ErrAccountHasPendingOperations),
			sl.Duration(time.Since(start)),
		)
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
		log.Error("failed to update account status",
			sl.ErrWithStack(err),
			sl.Duration(time.Since(start)),
		)
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("account updated",
		slog.String("account_id", acc.AccountId().String()),
		slog.String("status", acc.Status().String()),
		sl.Duration(time.Since(start)),
	)

	return acc, nil
}

func (a *AccountService) GetBalance(ctx context.Context, accountId uuid.UUID) (int64, error) {
	op := "AccountService.GetBalance"

	start := time.Now()
	log := a.log.With(
		sl.Op(op),
		sl.EventID(),
	)

	log.Info("get account balance")

	acc, err := a.accountRepository.GetById(ctx, accountId)
	if err != nil {
		log.Error("failed to get account",
			sl.ErrWithStack(err),
			sl.Duration(time.Since(start)))
		return -1, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("get account balance successfully",
		slog.String("account_id", acc.AccountId().String()),
		sl.Duration(time.Since(start)),
	)

	return acc.Balance(), nil
}

func (a *AccountService) LinkBankAccount(ctx context.Context, request LinkBankInput) (*entities.BankAccount, error) {
	op := "AccountService.LinkBankAccount"

	start := time.Now()
	log := a.log.With(
		sl.Op(op),
		sl.EventID(),
	)

	log.Info("link bank account attempt")

	acc, err := a.accountRepository.GetById(ctx, request.AccountID)
	if err != nil {
		log.Error("failed to get account",
			sl.ErrWithStack(err),
			sl.Duration(time.Since(start)),
		)
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if acc.CompanyId() != request.CompanyID {
		log.Warn("account does not belong to company",
			sl.ErrWithStack(ErrDifferentCompanies),
			sl.Duration(time.Since(start)),
		)
		return nil, fmt.Errorf("%s: %w", op, ErrDifferentCompanies)
	}

	bankAcc, err := entities.NewBankAccount(request.CompanyID, request.Name, request.Bic, request.SettlementAccount, request.Currency)
	if err != nil {
		log.Error("failed to create bank account",
			sl.ErrWithStack(err),
			sl.Duration(time.Since(start)),
		)
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if err = a.bankAccountRepository.Save(ctx, bankAcc); err != nil {
		log.Error("failed to save bank account",
			sl.ErrWithStack(err),
			sl.Duration(time.Since(start)),
		)
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("bank account created",
		slog.String("account_id", acc.AccountId().String()),
		slog.String("bank_account_id", bankAcc.BankAccountId().String()),
		sl.Duration(time.Since(start)))

	return bankAcc, nil
}

func (a *AccountService) ReserveWithdraw(ctx context.Context, accountId, counterpartyId uuid.UUID, txId uuid.UUID, amount int64) (*entities.AccountOperation, error) {
	op := "AccountService.ReserveWithdraw"

	start := time.Now()
	log := a.log.With(
		sl.Op(op),
		sl.EventID(),
	)

	log.Info("reserve withdraw attempt")

	acc, err := a.accountRepository.GetById(ctx, accountId)
	if err != nil {
		log.Error("failed to get account",
			sl.ErrWithStack(err),
			sl.Duration(time.Since(start)),
		)
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	existing, _ := a.accountOperationRepository.GetByTransactionIdAndType(ctx, txId, string(entities.Withdrawal))
	if existing != nil {
		log.Info("account has already withdrawn amount",
			slog.String("account_id", acc.AccountId().String()),
			sl.Duration(time.Since(start)))
		return existing, nil
	}

	if acc.Balance() < amount {
		log.Warn("account has not enough balance",
			slog.String("account_id", acc.AccountId().String()),
			sl.Duration(time.Since(start)),
		)
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

		accOp, err = entities.NewAccountOperation(accountId, counterpartyId, txId, entities.Withdrawal, entities.PendingStatus, amount, acc.Balance())
		if err != nil {
			return err
		}

		if err = a.accountOperationRepository.Save(txCtx, accOp); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		log.Error("failed to update account operation",
			sl.ErrWithStack(err),
			sl.Duration(time.Since(start)),
		)
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("withdraw reserved",
		slog.String("account_id", acc.AccountId().String()),
		sl.Duration(time.Since(start)),
	)

	return accOp, nil
}

func (a *AccountService) ReserveDeposit(ctx context.Context, accountId, counterpartyId uuid.UUID, txId uuid.UUID, amount int64) (*entities.AccountOperation, error) {
	op := "AccountService.ReserveDeposit"

	start := time.Now()
	log := a.log.With(
		sl.Op(op),
		sl.EventID(),
	)

	log.Info("reserve deposit attempt")

	acc, err := a.accountRepository.GetById(ctx, accountId)
	if err != nil {
		log.Error("failed to get account",
			sl.ErrWithStack(err),
			sl.Duration(time.Since(start)),
		)
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	existing, _ := a.accountOperationRepository.GetByTransactionIdAndType(ctx, txId, string(entities.Deposit))
	if existing != nil {
		log.Info("account has already withdrawn amount",
			slog.String("account_id", acc.AccountId().String()),
			sl.Duration(time.Since(start)),
		)
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

		accOp, err = entities.NewAccountOperation(accountId, counterpartyId, txId, entities.Deposit, entities.PendingStatus, amount, acc.Balance())
		if err != nil {
			return err
		}

		if err = a.accountOperationRepository.Save(txCtx, accOp); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		log.Error("failed to update account operation",
			sl.ErrWithStack(err),
			sl.Duration(time.Since(start)),
		)
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("deposit reserved",
		slog.String("account_id", acc.AccountId().String()),
		sl.Duration(time.Since(start)),
	)

	return accOp, nil
}

func (a *AccountService) ConfirmOperation(ctx context.Context, txId uuid.UUID) error {
	op := "AccountService.ConfirmOperation"

	start := time.Now()
	log := a.log.With(
		sl.Op(op),
		sl.EventID(),
	)

	log.Info("confirm operation attempt")

	err := a.accountOperationRepository.ConfirmAllByTransactionId(ctx, txId)
	if err != nil {
		log.Error("failed to confirm operation",
			sl.ErrWithStack(err),
			sl.Duration(time.Since(start)),
		)
		return fmt.Errorf("%s: %w", op, err)
	}

	log.Info("operation confirmed",
		sl.Duration(time.Since(start)))

	return nil
}

func (a *AccountService) CancelOperation(ctx context.Context, txId uuid.UUID) error {
	op := "AccountService.CancelOperation"

	start := time.Now()
	log := a.log.With(
		sl.Op(op),
		sl.EventID(),
	)

	log.Info("cancel operation attempt", sl.Duration(time.Since(start)))

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
		log.Error("failed to update account operation",
			sl.ErrWithStack(err),
			sl.Duration(time.Since(start)),
		)
		return fmt.Errorf("%s: %w", op, err)
	}

	log.Info("operation canceled",
		sl.Duration(time.Since(start)))

	return nil
}

func (a *AccountService) MakeBankDeposit(ctx context.Context, request *BankOperationInput) (*entities.BankOperation, error) {
	op := "AccountService.MakeBankDeposit"

	start := time.Now()
	log := a.log.With(
		sl.Op(op),
		sl.EventID(),
	)

	log.Info("make bank deposit attempt")

	existing, _ := a.bankOperationRepository.GetByIdempotencyKey(ctx, request.IdempotencyKey)
	if existing != nil {
		log.Info("account has already withdrawn amount",
			sl.Duration(time.Since(start)))
		return existing, nil
	}

	acc, err := a.accountRepository.GetById(ctx, request.AccountID)
	if err != nil {
		log.Error("failed to get account",
			sl.ErrWithStack(err),
			sl.Duration(time.Since(start)),
		)
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if acc.CompanyId() != request.CompanyID {
		log.Warn("account don't have the same company",
			sl.Duration(time.Since(start)),
		)
		return nil, fmt.Errorf("%s: %w", op, ErrDifferentCompanies)
	}

	bankAccount, err := a.bankAccountRepository.GetById(ctx, request.BankAccountID)
	if err != nil {
		log.Error("failed to get bank account",
			sl.ErrWithStack(err),
			sl.Duration(time.Since(start)))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	externalID, err := a.bankGateway.Deposit(ctx, bankAccount, request.Amount, request.IdempotencyKey)
	if err != nil {
		log.Error("failed to bank deposit",
			sl.ErrWithStack(err),
			sl.Duration(time.Since(start)),
		)
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

		bo, err = entities.NewBankOperation(request.AccountID, request.BankAccountID, request.InitiatorID, bankAccount.Name(),
			entities.Deposit, entities.SuccessStatus, request.Amount, acc.Balance(), request.IdempotencyKey, externalID)
		if err != nil {
			return err
		}

		if err = a.bankOperationRepository.Save(txCtx, bo); err != nil {
			return err
		}

		return a.saveOutboxEvent(txCtx, bo)
	})

	if err != nil {
		log.Error("failed to bank deposit",
			sl.ErrWithStack(err),
			sl.Duration(time.Since(start)),
		)
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("bank deposit succeeded",
		sl.Duration(time.Since(start)),
	)

	return bo, nil
}

func (a *AccountService) MakeBankWithdrawal(ctx context.Context, request *BankOperationInput) (*entities.BankOperation, error) {
	op := "AccountService.MakeBankWithdrawal"

	start := time.Now()
	log := a.log.With(
		sl.Op(op),
		sl.EventID(),
	)

	log.Info("make bank withdrawal attempt")

	existing, _ := a.bankOperationRepository.GetByIdempotencyKey(ctx, request.IdempotencyKey)
	if existing != nil {
		log.Info("account has already withdrawn amount",
			sl.Duration(time.Since(start)))
		return existing, nil
	}

	acc, err := a.accountRepository.GetById(ctx, request.AccountID)
	if err != nil {
		log.Error("failed to get account",
			sl.ErrWithStack(err),
			sl.Duration(time.Since(start)))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if acc.CompanyId() != request.CompanyID {
		log.Warn("account don't have the same company",
			sl.Duration(time.Since(start)))
		return nil, fmt.Errorf("%s: %w", op, ErrDifferentCompanies)
	}

	bankAccount, err := a.bankAccountRepository.GetById(ctx, request.BankAccountID)
	if err != nil {
		log.Error("failed to get bank account",
			sl.ErrWithStack(err),
			sl.Duration(time.Since(start)),
		)
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if acc.Balance() < request.Amount {
		log.Warn("account has not enough balance",
			sl.Duration(time.Since(start)))
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

		bo, err = entities.NewBankOperation(request.AccountID, request.BankAccountID, request.InitiatorID, bankAccount.Name(),
			entities.Withdrawal, entities.PendingStatus, request.Amount, acc.Balance(), request.IdempotencyKey, "")
		if err != nil {
			return err
		}

		if err = a.bankOperationRepository.Save(txCtx, bo); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		log.Error("failed to reserve bank withdrawal",
			sl.ErrWithStack(err),
			sl.Duration(time.Since(start)))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	externalID, err := a.bankGateway.Withdraw(ctx, bankAccount, request.Amount, request.IdempotencyKey)
	if err != nil {
		if err2 := a.compensateBankWithdrawal(ctx, acc, bo, externalID); err2 != nil {
			log.Error("failed to compensate bank withdrawal",
				sl.ErrWithStack(err2),
				sl.Duration(time.Since(start)))
			return nil, fmt.Errorf("%s: %w, %w", op, err, err2)
		}
		log.Error("failed to bank withdrawal",
			sl.ErrWithStack(err),
			sl.Duration(time.Since(start)),
		)
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
		log.Error("failed to success bank withdrawal",
			sl.ErrWithStack(err),
			sl.Duration(time.Since(start)))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("bank withdrawal succeeded",
		sl.Duration(time.Since(start)),
	)

	return bo, nil
}

func (a *AccountService) GetAccounts(ctx context.Context, companyId uuid.UUID) ([]*entities.Account, error) {
	op := "AccountService.GetAccounts"

	start := time.Now()
	log := a.log.With(
		sl.Op(op),
		sl.EventID(),
	)

	log.Info("get accounts")

	accounts, err := a.accountRepository.GetByCompanyId(ctx, companyId)
	if err != nil {
		log.Error("failed to get accounts",
			sl.ErrWithStack(err),
			sl.Duration(time.Since(start)))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("get accounts succeeded",
		sl.Duration(time.Since(start)))

	return accounts, nil
}

func (a *AccountService) GetBankAccounts(ctx context.Context, companyId uuid.UUID) ([]*entities.BankAccount, error) {
	op := "AccountService.GetBankAccounts"

	start := time.Now()
	log := a.log.With(
		sl.Op(op),
		sl.EventID(),
	)

	log.Info("get bank accounts")

	accounts, err := a.bankAccountRepository.GetByCompanyId(ctx, companyId)
	if err != nil {
		log.Error("failed to get bank accounts",
			sl.ErrWithStack(err),
			sl.Duration(time.Since(start)))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("get bank accounts succeeded",
		sl.Duration(time.Since(start)))

	return accounts, nil
}

func (a *AccountService) GenerateStatement(ctx context.Context, accountId, initiatorId, companyId uuid.UUID, periodFrom, periodTo time.Time) (*entities.Statement, error) {
	op := "AccountService.GenerateStatement"

	start := time.Now()
	log := a.log.With(
		sl.Op(op),
		sl.EventID(),
	)

	log.Info("trying to generate statement")

	exist, err := a.statementRepository.GetByAccountIdAndPeriod(ctx, accountId, periodFrom, periodTo)
	if err != nil && !errors.Is(err, domain.ErrStatementNotFound) {
		log.Error("failed to get statement",
			sl.ErrWithStack(err),
			sl.Duration(time.Since(start)))
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	if exist != nil {
		err = a.saveOutboxEvent(ctx, exist)
		if err != nil {
			log.Error("failed to save outbox event",
				sl.ErrWithStack(err),
				sl.Duration(time.Since(start)))
		}
		log.Info("statement generated successfully",
			sl.Duration(time.Since(start)))

		return exist, nil
	}

	acc, err := a.accountRepository.GetById(ctx, accountId)
	if err != nil {
		log.Error("failed to get account",
			sl.ErrWithStack(err),
			sl.Duration(time.Since(start)),
		)
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	entries := make([]entities.StatementEntry, 0)
	accOps, err := a.accountOperationRepository.GetByAccountIdAndPeriod(ctx, accountId, periodFrom, periodTo)
	if err != nil {
		log.Error("failed to get account operations",
			sl.ErrWithStack(err),
			sl.Duration(time.Since(start)))
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	bankOps, err := a.bankOperationRepository.GetByAccountIdAndPeriod(ctx, accountId, periodFrom, periodTo)
	if err != nil {
		log.Error("failed to get bank operations",
			sl.ErrWithStack(err),
			sl.Duration(time.Since(start)))
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	totalDebit := int64(0)
	totalCredit := int64(0)
	for _, operation := range accOps {
		entry := entities.StatementEntry{
			Date:         operation.CreatedAt(),
			Amount:       operation.Amount(),
			BalanceAfter: operation.BalanceAfter(),
			Counterparty: operation.CounterpartyId().String(),
		}
		if operation.OperationType() == entities.Deposit {
			entry.EntryType = entities.EntryTypeTransferIn
			totalDebit += operation.Amount()
		} else {
			entry.EntryType = entities.EntryTypeTransferOut
			totalCredit += operation.Amount()
		}
		entries = append(entries, entry)
	}

	for _, operation := range bankOps {
		entry := entities.StatementEntry{
			Date:         operation.CreatedAt(),
			Amount:       operation.Amount(),
			BalanceAfter: operation.BalanceAfter(),
			Counterparty: operation.BankName(),
		}
		if operation.OperationType() == entities.Deposit {
			entry.EntryType = entities.EntryTypeBankDeposit
			totalDebit += operation.Amount()
		} else {
			entry.EntryType = entities.EntryTypeBankWithdrawal
			totalCredit += operation.Amount()
		}
		entries = append(entries, entry)
	}
	if len(entries) == 0 {
		log.Warn("no statement entries", sl.Duration(time.Since(start)))
		return nil, fmt.Errorf("%s: %w", op, ErrNotAnyOperations)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Date.Before(entries[j].Date)
	})

	var openingBalance int64
	if entries[0].EntryType == entities.EntryTypeBankDeposit || entries[0].EntryType == entities.EntryTypeTransferIn {
		openingBalance = entries[0].BalanceAfter - entries[0].Amount
	} else {
		openingBalance = entries[0].BalanceAfter + entries[0].Amount
	}
	closedBalance := entries[len(entries)-1].BalanceAfter

	var statement *entities.Statement
	err = a.transactor.WithTx(ctx, func(txCtx context.Context) error {
		statement, err = entities.NewStatement(accountId, companyId, initiatorId, periodFrom, periodTo, openingBalance,
			closedBalance, totalDebit, totalCredit, acc.Currency(), entries)
		if err != nil {
			log.Error("failed to create statement",
				sl.ErrWithStack(err),
				sl.Duration(time.Since(start)))
			return err
		}

		err = a.statementRepository.Save(txCtx, statement)
		if err != nil {
			log.Error("failed to save statement",
				sl.ErrWithStack(err),
				sl.Duration(time.Since(start)),
			)
			return err
		}
		return a.saveOutboxEvent(ctx, statement)
	})
	if err != nil {
		log.Error("failed to save statement",
			sl.ErrWithStack(err),
			sl.Duration(time.Since(start)),
		)
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("statement generated successfully",
		sl.Duration(time.Since(start)))

	return statement, nil
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
