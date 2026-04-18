package grpc

import (
	"context"
	"fmt"
	transactionsv1 "shared/pkg/gen/go/transactions/v1"

	"google.golang.org/grpc"
)

type TransactionGRPCClient struct {
	client transactionsv1.TransactionServiceClient
}

func NewTransactionGRPCClient(conn *grpc.ClientConn) *TransactionGRPCClient {
	return &TransactionGRPCClient{
		client: transactionsv1.NewTransactionServiceClient(conn),
	}
}

func (c *TransactionGRPCClient) Transfer(ctx context.Context, request *transactionsv1.TransferRequest) (*transactionsv1.TransferResponse, error) {
	op := "TransactionGRPCClient.Transfer"
	resp, err := c.client.Transfer(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return resp, nil
}

func (c *TransactionGRPCClient) GetAllTransactions(ctx context.Context, request *transactionsv1.GetAllTransactionsRequest) (*transactionsv1.GetAllTransactionsResponse, error) {
	op := "TransactionGRPCClient.GetAllTransactions"
	resp, err := c.client.GetAllTransactions(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return resp, nil
}

func (c *TransactionGRPCClient) GetTransaction(ctx context.Context, request *transactionsv1.GetTransactionRequest) (*transactionsv1.GetTransactionResponse, error) {
	op := "TransactionGRPCClient.GetTransaction"
	resp, err := c.client.GetTransaction(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return resp, nil
}
