package grpc

import (
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Clients struct {
	Auth        *AuthGRPCClient
	Account     *AccountGRPCClient
	Transaction *TransactionGRPCClient
	conns       []*grpc.ClientConn
}

func InitClients(authAddr, accAddr, txAddr string) (*Clients, error) {
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	dial := func(addr string) (*grpc.ClientConn, error) {
		return grpc.NewClient(addr, opts...)
	}

	authConn, err := dial(authAddr)
	if err != nil {
		return nil, fmt.Errorf("auth conn: %w", err)
	}

	accConn, err := dial(accAddr)
	if err != nil {
		return nil, fmt.Errorf("account conn: %w", err)
	}

	txConn, err := dial(txAddr)
	if err != nil {
		return nil, fmt.Errorf("transaction conn: %w", err)
	}

	return &Clients{
		Auth:        NewAuthGRPCClient(authConn),
		Account:     NewAccountGRPCClient(accConn),
		Transaction: NewTransactionGRPCClient(txConn),
		conns:       []*grpc.ClientConn{authConn, accConn, txConn},
	}, nil
}

func (c *Clients) Close() {
	for _, conn := range c.conns {
		_ = conn.Close()
	}
}
