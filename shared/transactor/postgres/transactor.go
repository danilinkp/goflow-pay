package postgres

import (
	"context"

	"github.com/avito-tech/go-transaction-manager/trm/v2/manager"
)

type TransactionAdapter struct {
	trm *manager.Manager
}

func NewTransactionAdapter(trm *manager.Manager) *TransactionAdapter {
	return &TransactionAdapter{trm: trm}
}

func (a *TransactionAdapter) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return a.trm.Do(ctx, fn)
}
