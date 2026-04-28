package transactionsgrpc

import (
	"context"
	"errors"
	transactionsv1 "shared/pkg/gen/go/transactions/v1"
	"transactions/internal/domain"
	"transactions/internal/domain/entities"
	"transactions/internal/service"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type TransactionProvider interface {
	Transfer(ctx context.Context, request *service.TransferInput) (*entities.Transaction, error)
	GetAllTransactions(ctx context.Context, accountId uuid.UUID) ([]*entities.Transaction, error)
	GetTransaction(ctx context.Context, txId uuid.UUID) (*entities.Transaction, error)
}

type TransactionServer struct {
	transactionsv1.UnimplementedTransactionServiceServer
	svc TransactionProvider
}

func NewTransactionServer(svc TransactionProvider) *TransactionServer {
	return &TransactionServer{
		svc: svc,
	}
}

func Register(gRPCServer *grpc.Server, svc TransactionProvider) {
	transactionsv1.RegisterTransactionServiceServer(gRPCServer, NewTransactionServer(svc))
}

func (s *TransactionServer) Transfer(ctx context.Context, req *transactionsv1.TransferRequest) (*transactionsv1.TransferResponse, error) {
	initiatorId, err := uuid.Parse(req.GetInitiatorId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid initiator_id")
	}
	fromAccId, err := uuid.Parse(req.GetFromAccountId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid from_account_id")
	}
	toAccId, err := uuid.Parse(req.GetToAccountId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid to_account_id")
	}
	input := &service.TransferInput{
		InitiatorID:    initiatorId,
		FromAccountId:  fromAccId,
		ToAccountId:    toAccId,
		Amount:         req.GetAmount(),
		Currency:       currencyFromProto(req.GetCurrency()),
		IdempotencyKey: req.GetIdempotencyKey(),
	}

	tx, err := s.svc.Transfer(ctx, input)
	if err != nil {
		return nil, mapError(err)
	}

	return &transactionsv1.TransferResponse{Transaction: mapTransactionToProto(tx)}, nil
}

func (s *TransactionServer) GetAllTransactions(ctx context.Context, req *transactionsv1.GetAllTransactionsRequest) (*transactionsv1.GetAllTransactionsResponse, error) {
	accId, err := uuid.Parse(req.GetAccountId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid account_id")
	}

	txs, err := s.svc.GetAllTransactions(ctx, accId)
	if err != nil {
		return nil, mapError(err)
	}
	pbTxs := make([]*transactionsv1.Transaction, len(txs))
	for i, tx := range txs {
		pbTxs[i] = mapTransactionToProto(tx)
	}

	return &transactionsv1.GetAllTransactionsResponse{Transactions: pbTxs}, nil
}

func (s *TransactionServer) GetTransaction(ctx context.Context, req *transactionsv1.GetTransactionRequest) (*transactionsv1.GetTransactionResponse, error) {
	txId, err := uuid.Parse(req.GetTxId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tx_id")
	}
	tx, err := s.svc.GetTransaction(ctx, txId)
	if err != nil {
		return nil, mapError(err)
	}

	return &transactionsv1.GetTransactionResponse{Transaction: mapTransactionToProto(tx)}, nil
}

func currencyFromProto(c transactionsv1.Currency) entities.Currency {
	switch c {
	case transactionsv1.Currency_CURRENCY_EUR:
		return entities.EUR
	case transactionsv1.Currency_CURRENCY_USD:
		return entities.USD
	case transactionsv1.Currency_CURRENCY_RUB:
		return entities.RUB
	case transactionsv1.Currency_CURRENCY_CNY:
		return entities.CNY
	default:
		return ""
	}
}

func currencyToProto(c entities.Currency) transactionsv1.Currency {
	switch c {
	case entities.EUR:
		return transactionsv1.Currency_CURRENCY_EUR
	case entities.USD:
		return transactionsv1.Currency_CURRENCY_USD
	case entities.RUB:
		return transactionsv1.Currency_CURRENCY_RUB
	case entities.CNY:
		return transactionsv1.Currency_CURRENCY_CNY
	default:
		return transactionsv1.Currency_CURRENCY_UNSPECIFIED
	}
}

func transactionStatusToProto(s entities.TransactionStatus) transactionsv1.TransactionStatus {
	switch s {
	case entities.PendingStatus:
		return transactionsv1.TransactionStatus_TRANSACTION_STATUS_PENDING
	case entities.SuccessStatus:
		return transactionsv1.TransactionStatus_TRANSACTION_STATUS_SUCCESS
	case entities.FailedStatus:
		return transactionsv1.TransactionStatus_TRANSACTION_STATUS_FAILED
	case entities.ProcessingStatus:
		return transactionsv1.TransactionStatus_TRANSACTION_STATUS_PROCESSING
	default:
		return transactionsv1.TransactionStatus_TRANSACTION_STATUS_UNSPECIFIED
	}
}

func mapTransactionToProto(tx *entities.Transaction) *transactionsv1.Transaction {
	return &transactionsv1.Transaction{
		TransactionId:     tx.TransactionID().String(),
		InitiatorId:       tx.InitiatorId().String(),
		FromAccountId:     tx.FromAccountID().String(),
		ToAccountId:       tx.ToAccountID().String(),
		Amount:            tx.Amount(),
		Currency:          currencyToProto(tx.Currency()),
		IdempotencyKey:    tx.IdempotencyKey(),
		TransactionStatus: transactionStatusToProto(tx.Status()),
		CreatedAt:         timestamppb.New(tx.CreatedAt()),
	}
}

func mapError(err error) error {
	switch {
	case errors.Is(err, domain.ErrTransactionNotFound):
		return status.Error(codes.NotFound, "transaction not found")
	case errors.Is(err, domain.ErrTransactionAlreadyExists):
		return status.Error(codes.AlreadyExists, "transaction already exists")
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}
