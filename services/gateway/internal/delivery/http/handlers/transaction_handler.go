package handlers

import (
	"gateway/internal/clients/grpc"
	"gateway/internal/delivery/http/middleware"
	"gateway/internal/dto"
	"net/http"

	"github.com/gin-gonic/gin"
	transactionsv1 "shared/pkg/gen/go/transactions/v1"
)

type TransactionHandler struct {
	client *grpc.TransactionGRPCClient
}

func NewTransactionHandler(client *grpc.TransactionGRPCClient) *TransactionHandler {
	return &TransactionHandler{client: client}
}

func (h *TransactionHandler) Transfer(c *gin.Context) {
	var req dto.TransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.client.Transfer(middleware.GRPCContext(c), &transactionsv1.TransferRequest{
		FromAccountId:  req.FromAccountId.String(),
		ToAccountId:    req.ToAccountId.String(),
		Amount:         req.Amount,
		Currency:       parseTransactionCurrency(req.Currency),
		IdempotencyKey: req.IdempotencyKey,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, mapTransaction(resp.Transaction))
}

func (h *TransactionHandler) GetAllTransactions(c *gin.Context) {
	accountId := c.Param("account_id")

	resp, err := h.client.GetAllTransactions(middleware.GRPCContext(c), &transactionsv1.GetAllTransactionsRequest{
		AccountId: accountId,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	transactions := make([]dto.TransactionResponse, 0, len(resp.Transactions))
	for _, t := range resp.Transactions {
		transactions = append(transactions, mapTransaction(t))
	}

	c.JSON(http.StatusOK, transactions)
}

func (h *TransactionHandler) GetTransaction(c *gin.Context) {
	txId := c.Param("tx_id")

	resp, err := h.client.GetTransaction(middleware.GRPCContext(c), &transactionsv1.GetTransactionRequest{
		TxId: txId,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, mapTransaction(resp.Transaction))
}

func mapTransaction(t *transactionsv1.Transaction) dto.TransactionResponse {
	return dto.TransactionResponse{
		TransactionId:     mustParseUUID(t.TransactionId),
		InitiatorId:       mustParseUUID(t.InitiatorId),
		FromAccountId:     mustParseUUID(t.FromAccountId),
		ToAccountId:       mustParseUUID(t.ToAccountId),
		Amount:            t.Amount,
		Currency:          t.Currency.String(),
		IdempotencyKey:    t.IdempotencyKey,
		TransactionStatus: t.TransactionStatus.String(),
		CreatedAt:         t.CreatedAt.AsTime(),
	}
}

func parseTransactionCurrency(s string) transactionsv1.Currency {
	switch s {
	case "EUR":
		return transactionsv1.Currency_CURRENCY_EUR
	case "USD":
		return transactionsv1.Currency_CURRENCY_USD
	case "RUB":
		return transactionsv1.Currency_CURRENCY_RUB
	case "CNY":
		return transactionsv1.Currency_CURRENCY_CNY
	default:
		return transactionsv1.Currency_CURRENCY_UNSPECIFIED
	}
}
