package service_test

import (
	"transactions/internal/domain/entities"
	"transactions/internal/service"

	"github.com/google/uuid"
)

type TransactionMother struct{}

func (TransactionMother) ValidTransferInput() *service.TransferInput {
	return NewTransferInputBuilder().Build()
}

func (TransactionMother) ZeroAmountTransferInput() *service.TransferInput {
	return NewTransferInputBuilder().WithAmount(0).Build()
}

func (TransactionMother) NegativeAmountTransferInput() *service.TransferInput {
	return NewTransferInputBuilder().WithAmount(-100).Build()
}

func (TransactionMother) SameAccountTransferInput() *service.TransferInput {
	sameID := uuid.New()
	return NewTransferInputBuilder().
		WithFromAccountID(sameID).
		WithToAccountID(sameID).
		Build()
}

func (TransactionMother) PendingTransaction(req *service.TransferInput) *entities.Transaction {
	tx, err := entities.NewTransaction(
		req.InitiatorID,
		req.FromAccountId,
		req.ToAccountId,
		req.Amount,
		req.Currency,
		entities.PendingStatus,
		req.IdempotencyKey,
	)
	if err != nil {
		panic("TransactionMother.PendingTransaction: " + err.Error())
	}
	return tx
}

func (TransactionMother) ProcessingTransaction(req *service.TransferInput) *entities.Transaction {
	tx, err := entities.NewTransaction(
		req.InitiatorID,
		req.FromAccountId,
		req.ToAccountId,
		req.Amount,
		req.Currency,
		entities.ProcessingStatus,
		req.IdempotencyKey,
	)
	if err != nil {
		panic("TransactionMother.ProcessingTransaction: " + err.Error())
	}
	return tx
}

func (TransactionMother) CompletedTransaction(req *service.TransferInput) *entities.Transaction {
	tx, err := entities.NewTransaction(
		req.InitiatorID,
		req.FromAccountId,
		req.ToAccountId,
		req.Amount,
		req.Currency,
		entities.SuccessStatus,
		req.IdempotencyKey,
	)
	if err != nil {
		panic("TransactionMother.CompletedTransaction: " + err.Error())
	}
	return tx
}
