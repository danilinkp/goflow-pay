package accountsgrpc

import (
	"accounts/internal/domain"
	"accounts/internal/domain/entities"
	"accounts/internal/services"
	"context"
	"errors"
	accountsv1 "shared/pkg/gen/go/accounts/v1"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ctxKey string

const UserIDKey ctxKey = "userID"

type AccountProvider interface {
	CreateAccount(ctx context.Context, companyID uuid.UUID, currency entities.Currency) (*entities.Account, error)
	SetAccountInActive(ctx context.Context, accountID uuid.UUID) (*entities.Account, error)
	GetBalance(ctx context.Context, accountID uuid.UUID) (int64, error)
	LinkBankAccount(ctx context.Context, in services.LinkBankInput) (*entities.BankAccount, error)
	ReserveWithdraw(ctx context.Context, accountID uuid.UUID, txID uuid.UUID, amount int64) (*entities.AccountOperation, error)
	ReserveDeposit(ctx context.Context, accountID uuid.UUID, txID uuid.UUID, amount int64) (*entities.AccountOperation, error)
	ConfirmOperation(ctx context.Context, txID uuid.UUID) error
	CancelOperation(ctx context.Context, txID uuid.UUID) error
	MakeBankDeposit(ctx context.Context, in *services.BankOperationInput) (*entities.BankOperation, error)
	MakeBankWithdrawal(ctx context.Context, in *services.BankOperationInput) (*entities.BankOperation, error)
	GetAccounts(ctx context.Context, companyID uuid.UUID) ([]*entities.Account, error)
	GetBankAccounts(ctx context.Context, companyID uuid.UUID) ([]*entities.BankAccount, error)
}

type AccountServer struct {
	accountsv1.UnimplementedAccountServiceServer
	svc AccountProvider
}

func NewAccountServer(svc AccountProvider) *AccountServer {
	return &AccountServer{svc: svc}
}

func Register(gRPCServer *grpc.Server, svc AccountProvider) {
	accountsv1.RegisterAccountServiceServer(gRPCServer, NewAccountServer(svc))
}

func (s *AccountServer) CreateAccount(ctx context.Context, req *accountsv1.CreateAccountRequest) (*accountsv1.CreateAccountResponse, error) {
	companyID, err := uuid.Parse(req.GetCompanyId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid company_id")
	}

	acc, err := s.svc.CreateAccount(ctx, companyID, entities.Currency(req.GetCurrency().String()))
	if err != nil {
		return nil, mapError(err)
	}

	return &accountsv1.CreateAccountResponse{Account: mapAccountToProto(acc)}, nil
}

func (s *AccountServer) SetAccountInActive(ctx context.Context, req *accountsv1.SetAccountInActiveRequest) (*accountsv1.SetAccountInActiveResponse, error) {
	accID, err := uuid.Parse(req.GetAccountId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid account_id")
	}

	acc, err := s.svc.SetAccountInActive(ctx, accID)
	if err != nil {
		return nil, mapError(err)
	}

	return &accountsv1.SetAccountInActiveResponse{Account: mapAccountToProto(acc)}, nil
}

func (s *AccountServer) GetBalance(ctx context.Context, req *accountsv1.GetBalanceRequest) (*accountsv1.GetBalanceResponse, error) {
	accID, err := uuid.Parse(req.GetAccountId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid account_id")
	}

	balance, err := s.svc.GetBalance(ctx, accID)
	if err != nil {
		return nil, mapError(err)
	}

	return &accountsv1.GetBalanceResponse{Balance: balance}, nil
}

func (s *AccountServer) LinkBankAccount(ctx context.Context, req *accountsv1.LinkBankAccountRequest) (*accountsv1.LinkBankAccountResponse, error) {
	accID, err := uuid.Parse(req.GetAccountId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid account_id")
	}
	bankID, err := uuid.Parse(req.GetBankId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid bank_id")
	}
	compID, err := uuid.Parse(req.GetCompanyId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid company_id")
	}

	input := services.LinkBankInput{
		AccountID:         accID,
		BankID:            bankID,
		CompanyID:         compID,
		Name:              req.GetName(),
		Bic:               req.GetBic(),
		SettlementAccount: req.GetSettlementAccount(),
		Currency:          entities.Currency(req.GetCurrency().String()),
	}

	ba, err := s.svc.LinkBankAccount(ctx, input)
	if err != nil {
		return nil, mapError(err)
	}

	return &accountsv1.LinkBankAccountResponse{BankAccount: mapBankAccountToProto(ba)}, nil
}

func (s *AccountServer) ReserveWithdraw(ctx context.Context, req *accountsv1.ReserveWithdrawRequest) (*accountsv1.ReserveWithdrawResponse, error) {
	accID, txID, err := parseIDs(req.GetAccountId(), req.GetTxId())
	if err != nil {
		return nil, err
	}

	op, err := s.svc.ReserveWithdraw(ctx, accID, txID, req.GetAmount())
	if err != nil {
		return nil, mapError(err)
	}

	return &accountsv1.ReserveWithdrawResponse{AccountOperation: mapAccountOpToProto(op)}, nil
}

func (s *AccountServer) ReserveDeposit(ctx context.Context, req *accountsv1.ReserveDepositRequest) (*accountsv1.ReserveDepositResponse, error) {
	accID, txID, err := parseIDs(req.GetAccountId(), req.GetTxId())
	if err != nil {
		return nil, err
	}

	op, err := s.svc.ReserveDeposit(ctx, accID, txID, req.GetAmount())
	if err != nil {
		return nil, mapError(err)
	}

	return &accountsv1.ReserveDepositResponse{AccountOperation: mapAccountOpToProto(op)}, nil
}

func (s *AccountServer) ConfirmOperation(ctx context.Context, req *accountsv1.ConfirmOperationRequest) (*accountsv1.ConfirmOperationResponse, error) {
	txID, err := uuid.Parse(req.GetTxId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tx_id")
	}

	if err := s.svc.ConfirmOperation(ctx, txID); err != nil {
		return nil, mapError(err)
	}

	return &accountsv1.ConfirmOperationResponse{}, nil
}

func (s *AccountServer) CancelOperation(ctx context.Context, req *accountsv1.CancelOperationRequest) (*accountsv1.CancelOperationResponse, error) {
	txID, err := uuid.Parse(req.GetTxId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tx_id")
	}

	if err := s.svc.CancelOperation(ctx, txID); err != nil {
		return nil, mapError(err)
	}

	return &accountsv1.CancelOperationResponse{}, nil
}

func (s *AccountServer) MakeBankDeposit(ctx context.Context, req *accountsv1.MakeBankDepositRequest) (*accountsv1.MakeBankDepositResponse, error) {
	accID, err := uuid.Parse(req.GetAccountId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid account_id")
	}
	baID, err := uuid.Parse(req.GetBankAccountId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid bank_account_id")
	}

	initiatorId, ok := ctx.Value(UserIDKey).(uuid.UUID)
	if !ok {
		return nil, status.Error(codes.Internal, "user id not found in context")
	}

	input := &services.BankOperationInput{
		AccountID:      accID,
		BankAccountID:  baID,
		InitiatorID:    initiatorId,
		Amount:         req.GetAmount(),
		IdempotencyKey: req.GetIdempotencyKey(),
	}

	op, err := s.svc.MakeBankDeposit(ctx, input)
	if err != nil {
		return nil, mapError(err)
	}

	return &accountsv1.MakeBankDepositResponse{BankOperation: mapBankOpToProto(op)}, nil
}

func (s *AccountServer) MakeBankWithdrawal(ctx context.Context, req *accountsv1.MakeBankWithdrawalRequest) (*accountsv1.MakeBankWithdrawalResponse, error) {
	accID, err := uuid.Parse(req.GetAccountId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid account_id")
	}
	baID, err := uuid.Parse(req.GetBankAccountId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid bank_account_id")
	}
	initiatorId, ok := ctx.Value(UserIDKey).(uuid.UUID)
	if !ok {
		return nil, status.Error(codes.Internal, "user id not found in context")
	}

	input := &services.BankOperationInput{
		AccountID:      accID,
		BankAccountID:  baID,
		InitiatorID:    initiatorId,
		Amount:         req.GetAmount(),
		IdempotencyKey: req.GetIdempotencyKey(),
	}

	op, err := s.svc.MakeBankWithdrawal(ctx, input)
	if err != nil {
		return nil, mapError(err)
	}

	return &accountsv1.MakeBankWithdrawalResponse{BankOperation: mapBankOpToProto(op)}, nil
}

func (s *AccountServer) GetAccounts(ctx context.Context, req *accountsv1.GetAccountsRequest) (*accountsv1.GetAccountsResponse, error) {
	compID, err := uuid.Parse(req.GetCompanyId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid company_id")
	}

	accounts, err := s.svc.GetAccounts(ctx, compID)
	if err != nil {
		return nil, mapError(err)
	}

	pbAccounts := make([]*accountsv1.Account, len(accounts))
	for i, a := range accounts {
		pbAccounts[i] = mapAccountToProto(a)
	}

	return &accountsv1.GetAccountsResponse{Accounts: pbAccounts}, nil
}

func (s *AccountServer) GetBankAccounts(ctx context.Context, req *accountsv1.GetBankAccountsRequest) (*accountsv1.GetBankAccountsResponse, error) {
	compID, err := uuid.Parse(req.GetCompanyId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid company_id")
	}

	bankAccounts, err := s.svc.GetBankAccounts(ctx, compID)
	if err != nil {
		return nil, mapError(err)
	}

	pbBankAccounts := make([]*accountsv1.BankAccount, len(bankAccounts))
	for i, ba := range bankAccounts {
		pbBankAccounts[i] = mapBankAccountToProto(ba)
	}

	return &accountsv1.GetBankAccountsResponse{BankAccounts: pbBankAccounts}, nil
}

func parseIDs(accStr, txStr string) (uuid.UUID, uuid.UUID, error) {
	accID, err := uuid.Parse(accStr)
	if err != nil {
		return uuid.Nil, uuid.Nil, status.Error(codes.InvalidArgument, "invalid account_id")
	}
	txID, err := uuid.Parse(txStr)
	if err != nil {
		return uuid.Nil, uuid.Nil, status.Error(codes.InvalidArgument, "invalid tx_id")
	}
	return accID, txID, nil
}

func mapAccountToProto(a *entities.Account) *accountsv1.Account {
	return &accountsv1.Account{
		Id:        a.AccountId().String(),
		CompanyId: a.CompanyId().String(),
		Balance:   a.Balance(),
		Currency:  accountsv1.Currency(accountsv1.Currency_value[string(a.Currency())]),
		Status:    accountsv1.AccountStatus(accountsv1.AccountStatus_value[string(a.Status())]),
		CreatedAt: timestamppb.New(a.CreatedAt()),
	}
}

func mapBankAccountToProto(ba *entities.BankAccount) *accountsv1.BankAccount {
	return &accountsv1.BankAccount{
		BankAccountId:     ba.BankAccountId().String(),
		CompanyId:         ba.CompanyId().String(),
		Name:              ba.Name(),
		Bic:               ba.BIC(),
		SettlementAccount: ba.SettlementAccount(),
		Currency:          accountsv1.Currency(accountsv1.Currency_value[string(ba.Currency())]),
		CreatedAt:         timestamppb.New(ba.CreatedAt()),
	}
}

func mapAccountOpToProto(op *entities.AccountOperation) *accountsv1.AccountOperation {
	return &accountsv1.AccountOperation{
		OperationId:     op.AccountOperationId().String(),
		AccountId:       op.AccountId().String(),
		TransactionId:   op.TransactionId().String(),
		OperationType:   accountsv1.OperationType(accountsv1.OperationType_value[string(op.OperationType())]),
		OperationStatus: accountsv1.OperationStatus(accountsv1.OperationStatus_value[string(op.OperationStatus())]),
		Amount:          op.Amount(),
		CreatedAt:       timestamppb.New(op.CreatedAt()),
	}
}

func mapBankOpToProto(op *entities.BankOperation) *accountsv1.BankOperation {
	return &accountsv1.BankOperation{
		BankOperationId: op.BankOperationId().String(),
		AccountId:       op.AccountId().String(),
		BankAccountId:   op.BankAccountId().String(),
		OperationType:   accountsv1.OperationType(accountsv1.OperationType_value[string(op.OperationType())]),
		OperationStatus: accountsv1.OperationStatus(accountsv1.OperationStatus_value[string(op.OperationStatus())]),
		Amount:          op.Amount(),
		IdempotencyKey:  op.IdempotencyKey(),
		ExternalId:      op.ExternalId(),
		CreatedAt:       timestamppb.New(op.CreatedAt()),
	}
}

func mapError(err error) error {
	switch {
	case errors.Is(err, domain.ErrAccountNotFound),
		errors.Is(err, domain.ErrBankAccountNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, services.ErrNotEnoughFunds):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, services.ErrDifferentCompanies):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, services.ErrAccountHasPendingOperations):
		return status.Error(codes.Aborted, err.Error())
	case errors.Is(err, domain.ErrAccountAlreadyExists):
		return status.Error(codes.AlreadyExists, err.Error())
	default:
		return status.Error(codes.Internal, "internal error")
	}
}
