package grpc

import (
	"context"
	"fmt"
	accountsv1 "shared/pkg/gen/go/accounts/v1"

	"github.com/google/uuid"
	"google.golang.org/grpc"
)

type AccountGRPCClient struct {
	client accountsv1.AccountServiceClient
}

func NewAccountGRPCClient(conn *grpc.ClientConn) *AccountGRPCClient {
	return &AccountGRPCClient{
		client: accountsv1.NewAccountServiceClient(conn),
	}
}

func (c *AccountGRPCClient) ReserveWithdraw(ctx context.Context, accountId uuid.UUID, txId uuid.UUID, amount int64) error {
	op := "AccountGRPCClient.ReserveWithdraw"
	_, err := c.client.ReserveWithdraw(ctx, &accountsv1.ReserveWithdrawRequest{
		AccountId: accountId.String(),
		TxId:      txId.String(),
		Amount:    amount,
	})

	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (c *AccountGRPCClient) ReserveDeposit(ctx context.Context, accountId uuid.UUID, txId uuid.UUID, amount int64) error {
	op := "AccountGRPCClient.ReserveDeposit"

	_, err := c.client.ReserveDeposit(ctx, &accountsv1.ReserveDepositRequest{
		AccountId: accountId.String(),
		TxId:      txId.String(),
		Amount:    amount,
	})

	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (c *AccountGRPCClient) ConfirmOperation(ctx context.Context, txId uuid.UUID) error {
	op := "AccountGRPCClient.ConfirmOperation"
	_, err := c.client.ConfirmOperation(ctx, &accountsv1.ConfirmOperationRequest{
		TxId: txId.String(),
	})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (c *AccountGRPCClient) CancelOperation(ctx context.Context, txId uuid.UUID) error {
	op := "AccountGRPCClient.CancelOperation"
	_, err := c.client.CancelOperation(ctx, &accountsv1.CancelOperationRequest{
		TxId: txId.String(),
	})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}
