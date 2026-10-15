package grpc

import (
	"context"
	"fmt"
	accountsv1 "shared/pkg/gen/go/accounts/v1"

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

func (c *AccountGRPCClient) CreateAccount(ctx context.Context, req *accountsv1.CreateAccountRequest) (*accountsv1.CreateAccountResponse, error) {
	op := "accountGRPCClient.CreateAccount"
	resp, err := c.client.CreateAccount(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return resp, nil
}

func (c *AccountGRPCClient) SetAccountInActive(ctx context.Context, req *accountsv1.SetAccountInActiveRequest) (*accountsv1.SetAccountInActiveResponse, error) {
	op := "accountGRPCClient.SetAccountInActive"
	resp, err := c.client.SetAccountInActive(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return resp, nil
}

func (c *AccountGRPCClient) GetBalance(ctx context.Context, req *accountsv1.GetBalanceRequest) (*accountsv1.GetBalanceResponse, error) {
	op := "accountGRPCClient.GetBalance"
	resp, err := c.client.GetBalance(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return resp, nil
}

func (c *AccountGRPCClient) LinkBankAccount(ctx context.Context, req *accountsv1.LinkBankAccountRequest) (*accountsv1.LinkBankAccountResponse, error) {
	op := "accountGRPCClient.LinkBankAccount"
	resp, err := c.client.LinkBankAccount(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return resp, nil
}

func (c *AccountGRPCClient) MakeBankDeposit(ctx context.Context, req *accountsv1.MakeBankDepositRequest) (*accountsv1.MakeBankDepositResponse, error) {
	op := "accountGRPCClient.MakeBankDeposit"

	resp, err := c.client.MakeBankDeposit(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return resp, nil
}

func (c *AccountGRPCClient) MakeBankWithdrawal(ctx context.Context, req *accountsv1.MakeBankWithdrawalRequest) (*accountsv1.MakeBankWithdrawalResponse, error) {
	op := "accountGRPCClient.MakeBankWithdrawal"
	resp, err := c.client.MakeBankWithdrawal(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return resp, nil
}

func (c *AccountGRPCClient) GetAccounts(ctx context.Context, req *accountsv1.GetAccountsRequest) (*accountsv1.GetAccountsResponse, error) {
	op := "accountGRPCClient.GetAccounts"
	resp, err := c.client.GetAccounts(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return resp, nil
}

func (c *AccountGRPCClient) GetBankAccounts(ctx context.Context, req *accountsv1.GetBankAccountsRequest) (*accountsv1.GetBankAccountsResponse, error) {
	op := "accountGRPCClient.GetBankAccounts"
	resp, err := c.client.GetBankAccounts(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return resp, nil
}

func (c *AccountGRPCClient) GenerateStatement(ctx context.Context, req *accountsv1.GenerateStatementRequest) (*accountsv1.GenerateStatementResponse, error) {
	op := "accountGRPCClient.GenerateStatement"
	resp, err := c.client.GenerateStatement(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return resp, nil
}

func (c *AccountGRPCClient) GetAccount(ctx context.Context, req *accountsv1.GetAccountRequest) (*accountsv1.GetAccountResponse, error) {
	op := "accountGRPCClient.GetAccount"
	resp, err := c.client.GetAccount(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return resp, nil
}

func (c *AccountGRPCClient) UpdateAccountStatus(ctx context.Context, req *accountsv1.UpdateAccountStatusRequest) (*accountsv1.UpdateAccountStatusResponse, error) {
	op := "accountGRPCClient.UpdateAccountStatus"
	resp, err := c.client.UpdateAccountStatus(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return resp, nil
}

func (c *AccountGRPCClient) GetBankAccount(ctx context.Context, req *accountsv1.GetBankAccountRequest) (*accountsv1.GetBankAccountResponse, error) {
	op := "accountGRPCClient.GetBankAccount"
	resp, err := c.client.GetBankAccount(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return resp, nil
}

func (c *AccountGRPCClient) DeleteBankAccount(ctx context.Context, req *accountsv1.DeleteBankAccountRequest) (*accountsv1.DeleteBankAccountResponse, error) {
	op := "accountGRPCClient.DeleteBankAccount"
	resp, err := c.client.DeleteBankAccount(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return resp, nil
}
