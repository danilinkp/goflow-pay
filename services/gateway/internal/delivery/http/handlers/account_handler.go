package handlers

import (
	"gateway/internal/clients/grpc"
	"gateway/internal/delivery/http/middleware"
	"gateway/internal/dto"
	"net/http"
	accountsv1 "shared/pkg/gen/go/accounts/v1"

	"github.com/gin-gonic/gin"
)

type AccountHandler struct {
	client *grpc.AccountGRPCClient
}

func NewAccountHandler(client *grpc.AccountGRPCClient) *AccountHandler {
	return &AccountHandler{client: client}
}

func (h *AccountHandler) CreateAccount(c *gin.Context) {
	companyId := c.Param("company_id")

	var req struct {
		Currency string `json:"currency"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.client.CreateAccount(middleware.GRPCContext(c), &accountsv1.CreateAccountRequest{
		CompanyId: companyId,
		Currency:  parseCurrency(req.Currency),
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	c.JSON(http.StatusCreated, mapAccount(resp.Account))
}

func (h *AccountHandler) SetAccountInActive(c *gin.Context) {
	accountId := c.Param("account_id")

	resp, err := h.client.SetAccountInActive(middleware.GRPCContext(c), &accountsv1.SetAccountInActiveRequest{
		AccountId: accountId,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, mapAccount(resp.Account))
}

func (h *AccountHandler) GetBalance(c *gin.Context) {
	accountId := c.Param("account_id")

	resp, err := h.client.GetBalance(middleware.GRPCContext(c), &accountsv1.GetBalanceRequest{
		AccountId: accountId,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"balance": resp.Balance})
}

func (h *AccountHandler) GetAccounts(c *gin.Context) {
	companyId := c.Param("company_id")

	resp, err := h.client.GetAccounts(middleware.GRPCContext(c), &accountsv1.GetAccountsRequest{
		CompanyId: companyId,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	accounts := make([]dto.AccountResponse, 0, len(resp.Accounts))
	for _, a := range resp.Accounts {
		accounts = append(accounts, mapAccount(a))
	}

	c.JSON(http.StatusOK, accounts)
}

func (h *AccountHandler) LinkBankAccount(c *gin.Context) {
	var req dto.LinkBankAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.client.LinkBankAccount(middleware.GRPCContext(c), &accountsv1.LinkBankAccountRequest{
		AccountId:         req.AccountId.String(),
		BankId:            req.BankId.String(),
		CompanyId:         req.CompanyId.String(),
		Name:              req.Name,
		Bic:               req.BIC,
		SettlementAccount: req.SettlementAccount.String(),
		Currency:          parseCurrency(req.Currency),
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	c.JSON(http.StatusCreated, mapBankAccount(resp.BankAccount))
}

func (h *AccountHandler) GetBankAccounts(c *gin.Context) {
	companyId := c.Param("company_id")

	resp, err := h.client.GetBankAccounts(middleware.GRPCContext(c), &accountsv1.GetBankAccountsRequest{
		CompanyId: companyId,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	bankAccounts := make([]dto.BankAccountResponse, 0, len(resp.BankAccounts))
	for _, ba := range resp.BankAccounts {
		bankAccounts = append(bankAccounts, mapBankAccount(ba))
	}

	c.JSON(http.StatusOK, bankAccounts)
}

func (h *AccountHandler) MakeBankDeposit(c *gin.Context) {
	var req dto.MakeBankOperationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.client.MakeBankDeposit(middleware.GRPCContext(c), &accountsv1.MakeBankDepositRequest{
		AccountId:      req.AccountId.String(),
		BankAccountId:  req.BankAccountId.String(),
		Amount:         req.Amount,
		IdempotencyKey: req.IdempotencyKey,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, mapBankOperation(resp.BankOperation))
}

func (h *AccountHandler) MakeBankWithdrawal(c *gin.Context) {
	var req dto.MakeBankOperationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.client.MakeBankWithdrawal(middleware.GRPCContext(c), &accountsv1.MakeBankWithdrawalRequest{
		AccountId:      req.AccountId.String(),
		BankAccountId:  req.BankAccountId.String(),
		Amount:         req.Amount,
		IdempotencyKey: req.IdempotencyKey,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, mapBankOperation(resp.BankOperation))
}

// mappers
func mapAccount(a *accountsv1.Account) dto.AccountResponse {
	return dto.AccountResponse{
		AccountId: mustParseUUID(a.Id),
		CompanyId: mustParseUUID(a.CompanyId),
		Balance:   a.Balance,
		Currency:  a.Currency.String(),
		Status:    a.Status.String(),
		CreatedAt: a.CreatedAt.AsTime(),
	}
}

func mapBankAccount(ba *accountsv1.BankAccount) dto.BankAccountResponse {
	return dto.BankAccountResponse{
		BankAccountId:     mustParseUUID(ba.BankAccountId),
		CompanyId:         mustParseUUID(ba.CompanyId),
		Name:              ba.Name,
		BIC:               ba.Bic,
		SettlementAccount: mustParseUUID(ba.SettlementAccount),
		Currency:          ba.Currency.String(),
		CreatedAt:         ba.CreatedAt.AsTime(),
	}
}

func mapBankOperation(bo *accountsv1.BankOperation) dto.BankOperationResponse {
	return dto.BankOperationResponse{
		BankOperationId: mustParseUUID(bo.BankOperationId),
		AccountId:       mustParseUUID(bo.AccountId),
		BankAccountId:   mustParseUUID(bo.BankAccountId),
		OperationType:   bo.OperationType.String(),
		OperationStatus: bo.OperationStatus.String(),
		Amount:          bo.Amount,
		IdempotencyKey:  bo.IdempotencyKey,
		ExternalId:      bo.ExternalId,
		CreatedAt:       bo.CreatedAt.AsTime(),
	}
}

func parseCurrency(s string) accountsv1.Currency {
	switch s {
	case "EUR":
		return accountsv1.Currency_CURRENCY_EUR
	case "USD":
		return accountsv1.Currency_CURRENCY_USD
	case "RUB":
		return accountsv1.Currency_CURRENCY_RUB
	case "CNY":
		return accountsv1.Currency_CURRENCY_CNY
	default:
		return accountsv1.Currency_CURRENCY_UNSPECIFIED
	}
}
