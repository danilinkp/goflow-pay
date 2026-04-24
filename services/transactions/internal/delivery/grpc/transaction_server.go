package transactionsgrpc

import (
	"context"
	"errors"
	transactionsv1 "shared/pkg/gen/go/transactions/v1"
	"transactions/internal/domain"
	"transactions/internal/domain/entities"
	"transactions/internal/services"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ctxKey string

const UserIDKey ctxKey = "userID"

type TransactionProvider interface {
	Transfer(ctx context.Context, request *services.TransferInput) (*entities.Transaction, error)
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
	initiatorId, ok := ctx.Value(UserIDKey).(uuid.UUID)
	if !ok {
		return nil, status.Error(codes.Internal, "user id not found in context")
	}
	fromAccId, err := uuid.Parse(req.GetFromAccountId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid from_account_id")
	}
	toAccId, err := uuid.Parse(req.GetToAccountId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid to_account_id")
	}
	input := &services.TransferInput{
		InitiatorID:    initiatorId,
		FromAccountId:  fromAccId,
		ToAccountId:    toAccId,
		Amount:         req.GetAmount(),
		Currency:       entities.Currency(req.GetCurrency()),
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

func mapTransactionToProto(tx *entities.Transaction) *transactionsv1.Transaction {
	return &transactionsv1.Transaction{
		TransactionId:     tx.TransactionID().String(),
		InitiatorId:       tx.InitiatorId().String(),
		FromAccountId:     tx.FromAccountID().String(),
		ToAccountId:       tx.ToAccountID().String(),
		Amount:            tx.Amount(),
		Currency:          transactionsv1.Currency(transactionsv1.Currency_value[tx.Currency().String()]),
		IdempotencyKey:    tx.IdempotencyKey(),
		TransactionStatus: transactionsv1.TransactionStatus(transactionsv1.TransactionStatus_value[tx.Status().String()]),
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
