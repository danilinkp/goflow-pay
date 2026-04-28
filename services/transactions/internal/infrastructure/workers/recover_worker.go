package workers

import (
	"context"
	"time"
	"transactions/internal/domain/entities"
	"transactions/internal/service"
)

type RecoverWorker struct {
	txRepo     service.TransactionRepository
	txService  *service.TransactionService
	interval   time.Duration
	staleAfter time.Duration
}

func NewRecoverWorker(txRepo service.TransactionRepository, txService *service.TransactionService, interval, staleAfter time.Duration) *RecoverWorker {
	return &RecoverWorker{
		txRepo:     txRepo,
		txService:  txService,
		interval:   interval,
		staleAfter: staleAfter,
	}
}

func (w *RecoverWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.processBatch(ctx)
		}
	}
}

func (w *RecoverWorker) processBatch(ctx context.Context) {
	txs, err := w.txRepo.GetStale(ctx, w.staleAfter, []string{
		string(entities.PendingStatus),
		string(entities.ProcessingStatus),
	})
	if err != nil {
		return
	}
	for _, tx := range txs {
		_ = w.txService.Recover(ctx, tx.TransactionID())
	}
}
