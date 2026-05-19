package transactor

import (
	"context"

	"github.com/avito-tech/go-transaction-manager/trm/v2/manager"
)

type Transactioner interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}

type AvitoAdapter struct {
	trm *manager.Manager
}

func NewAvitoAdapter(trm *manager.Manager) *AvitoAdapter {
	return &AvitoAdapter{trm: trm}
}

func (a *AvitoAdapter) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return a.trm.Do(ctx, fn)
}
