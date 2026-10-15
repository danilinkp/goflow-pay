package v2

import (
	"fmt"
	"net/http"
	transactionsv1 "shared/pkg/gen/go/transactions/v1"

	"github.com/gin-gonic/gin"
)

func (s *Server) CreateTransaction(c *gin.Context, params CreateTransactionParams) {
	userID, _, role := getContext(c)
	if role != "company_admin" && role != "admin" {
		sendError(c, http.StatusForbidden, "FORBIDDEN", "only company_admin can initiate transfers")
		return
	}

	var req CreateTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	idempotencyKey := params.IdempotencyKey.String()
	protoCurr := toProtoTxCurrency(req.Currency)

	resp, err := s.txClient.Transfer(c.Request.Context(), &transactionsv1.TransferRequest{
		FromAccountId:  req.FromAccountId.String(),
		ToAccountId:    req.ToAccountId.String(),
		InitiatorId:    userID.String(),
		Amount:         req.Amount,
		Currency:       protoCurr,
		IdempotencyKey: idempotencyKey,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	txResp := mapTxToResponse(resp.Transaction)
	txResp.Purpose = req.Purpose

	c.Header("Location", fmt.Sprintf("/api/v2/transactions/%s", resp.Transaction.TransactionId))
	c.JSON(http.StatusCreated, txResp)
}

func (s *Server) GetTransactionById(c *gin.Context, transactionId TransactionIdPath) {
	resp, err := s.txClient.GetTransaction(c.Request.Context(), &transactionsv1.GetTransactionRequest{
		TxId: transactionId.String(),
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, mapTxToResponse(resp.Transaction))
}
