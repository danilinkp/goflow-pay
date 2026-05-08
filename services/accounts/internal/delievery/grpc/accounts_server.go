package accountsgrpc

import (
	"accounts/internal/domain"
	"accounts/internal/domain/entities"
	"accounts/internal/services"
	"context"
	"errors"
	"fmt"
	accountsv1 "shared/pkg/gen/go/accounts/v1"
	"strings"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ctxKey string

const UserIDKey ctxKey = "user_id"

type AccountProvider interface {
	CreateAccount(ctx context.Context, companyID uuid.UUID, currency entities.Currency) (*entities.Account, error)
	SetAccountInActive(ctx context.Context, accountId, companyId uuid.UUID) (*entities.Account, error)
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

	protoStr := req.GetCurrency().String()
	cleanStr := strings.TrimPrefix(protoStr, "CURRENCY_")
	acc, err := s.svc.CreateAccount(ctx, companyID, entities.Currency(cleanStr))
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
	companyId, err := uuid.Parse(req.GetCompanyId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid company_id")
	}

	acc, err := s.svc.SetAccountInActive(ctx, accID, companyId)
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
	compID, err := uuid.Parse(req.GetCompanyId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid company_id")
	}

	input := services.LinkBankInput{
		AccountID:         accID,
		CompanyID:         compID,
		Name:              req.GetName(),
		Bic:               req.GetBic(),
		SettlementAccount: req.GetSettlementAccount(),
		Currency:          currencyFromProto(req.GetCurrency()),
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
	companyID, err := uuid.Parse(req.GetCompanyId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid company_id")
	}
	accID, err := uuid.Parse(req.GetAccountId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid account_id")
	}
	baID, err := uuid.Parse(req.GetBankAccountId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid bank_account_id")
	}
	initiatorId, err := uuid.Parse(req.GetInitiatorId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid initiator_id")
	}
	fmt.Printf("USER ID: %s", initiatorId.String())
	input := &services.BankOperationInput{
		CompanyID:      companyID,
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
	companyID, err := uuid.Parse(req.GetCompanyId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid company_id")
	}
	accID, err := uuid.Parse(req.GetAccountId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid account_id")
	}
	baID, err := uuid.Parse(req.GetBankAccountId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid bank_account_id")
	}
	initiatorId, err := uuid.Parse(req.GetInitiatorId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid initiator_id")
	}
	input := &services.BankOperationInput{
		CompanyID:      companyID,
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

func currencyFromProto(c accountsv1.Currency) entities.Currency {
	switch c {
	case accountsv1.Currency_CURRENCY_EUR:
		return entities.EUR
	case accountsv1.Currency_CURRENCY_USD:
		return entities.USD
	case accountsv1.Currency_CURRENCY_RUB:
		return entities.RUB
	case accountsv1.Currency_CURRENCY_CNY:
		return entities.CNY
	default:
		return ""
	}
}

func currencyToProto(c entities.Currency) accountsv1.Currency {
	switch c {
	case entities.EUR:
		return accountsv1.Currency_CURRENCY_EUR
	case entities.USD:
		return accountsv1.Currency_CURRENCY_USD
	case entities.RUB:
		return accountsv1.Currency_CURRENCY_RUB
	case entities.CNY:
		return accountsv1.Currency_CURRENCY_CNY
	default:
		return accountsv1.Currency_CURRENCY_UNSPECIFIED
	}
}

func accountStatusToProto(s entities.AccountStatus) accountsv1.AccountStatus {
	switch s {
	case entities.ActiveStatus:
		return accountsv1.AccountStatus_ACCOUNT_STATUS_ACTIVE
	case entities.InactiveStatus:
		return accountsv1.AccountStatus_ACCOUNT_STATUS_INACTIVE
	default:
		return accountsv1.AccountStatus_ACCOUNT_STATUS_UNSPECIFIED
	}
}

func operationTypeToProto(t entities.OperationType) accountsv1.OperationType {
	switch t {
	case entities.Deposit:
		return accountsv1.OperationType_OPERATION_TYPE_DEPOSIT
	case entities.Withdrawal:
		return accountsv1.OperationType_OPERATION_TYPE_WITHDRAWAL
	default:
		return accountsv1.OperationType_OPERATION_TYPE_UNSPECIFIED
	}
}

func operationStatusToProto(s entities.OperationStatus) accountsv1.OperationStatus {
	switch s {
	case entities.PendingStatus:
		return accountsv1.OperationStatus_OPERATION_STATUS_PENDING
	case entities.SuccessStatus:
		return accountsv1.OperationStatus_OPERATION_STATUS_SUCCESS
	case entities.FailedStatus:
		return accountsv1.OperationStatus_OPERATION_STATUS_FAILED
	default:
		return accountsv1.OperationStatus_OPERATION_STATUS_UNSPECIFIED
	}
}

func mapAccountToProto(a *entities.Account) *accountsv1.Account {
	return &accountsv1.Account{
		Id:        a.AccountId().String(),
		CompanyId: a.CompanyId().String(),
		Balance:   a.Balance(),
		Currency:  currencyToProto(a.Currency()),
		Status:    accountStatusToProto(a.Status()),
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
		Currency:          currencyToProto(ba.Currency()),
		CreatedAt:         timestamppb.New(ba.CreatedAt()),
	}
}

func mapAccountOpToProto(op *entities.AccountOperation) *accountsv1.AccountOperation {
	return &accountsv1.AccountOperation{
		OperationId:     op.AccountOperationId().String(),
		AccountId:       op.AccountId().String(),
		TransactionId:   op.TransactionId().String(),
		OperationType:   operationTypeToProto(op.OperationType()),
		OperationStatus: operationStatusToProto(op.OperationStatus()),
		Amount:          op.Amount(),
		CreatedAt:       timestamppb.New(op.CreatedAt()),
	}
}

func mapBankOpToProto(op *entities.BankOperation) *accountsv1.BankOperation {
	return &accountsv1.BankOperation{
		BankOperationId: op.BankOperationId().String(),
		AccountId:       op.AccountId().String(),
		BankAccountId:   op.BankAccountId().String(),
		InitiatorId:     op.InitiatorId().String(),
		OperationType:   operationTypeToProto(op.OperationType()),
		OperationStatus: operationStatusToProto(op.OperationStatus()),
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
