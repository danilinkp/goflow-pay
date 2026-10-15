package v2

import (
	"fmt"
	"net/http"
	accountsv1 "shared/pkg/gen/go/accounts/v1"
	transactionsv1 "shared/pkg/gen/go/transactions/v1"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *Server) ListAccounts(c *gin.Context, params ListAccountsParams) {
	_, userCompanyID, _ := getContext(c)

	resp, err := s.accountClient.GetAccounts(c.Request.Context(), &accountsv1.GetAccountsRequest{
		CompanyId: userCompanyID.String(),
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	filtered := make([]AccountResponse, 0, len(resp.Accounts))
	for _, a := range resp.Accounts {
		accResp := mapAccountToResponse(a)
		if params.Currency != nil && accResp.Currency != *params.Currency {
			continue
		}
		if params.Status != nil && accResp.Status != *params.Status {
			continue
		}
		filtered = append(filtered, accResp)
	}

	total := len(filtered)
	offset := 0
	if params.Offset != nil {
		offset = *params.Offset
	}
	limit := 20
	if params.Limit != nil {
		limit = *params.Limit
	}
	if offset > total {
		offset = total
	}
	end := offset + limit
	if end > total {
		end = total
	}

	c.JSON(http.StatusOK, AccountListResponse{
		Items: filtered[offset:end],
		Total: total,
	})
}

func (s *Server) CreateAccount(c *gin.Context) {
	_, userCompanyID, role := getContext(c)
	if role != "company_admin" && role != "admin" {
		sendError(c, http.StatusForbidden, "FORBIDDEN", "only company_admin can create accounts")
		return
	}

	var req CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	protoCurr := toProtoCurrency(req.Currency)
	resp, err := s.accountClient.CreateAccount(c.Request.Context(), &accountsv1.CreateAccountRequest{
		CompanyId: userCompanyID.String(),
		Currency:  protoCurr,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	accResp := mapAccountToResponse(resp.Account)
	if req.Name != nil {
		accResp.Name = req.Name
	}

	c.Header("Location", fmt.Sprintf("/api/v2/accounts/%s", resp.Account.Id))
	c.JSON(http.StatusCreated, accResp)
}

func (s *Server) GetAccountById(c *gin.Context, accountId AccountIdPath) {
	resp, err := s.accountClient.GetAccount(c.Request.Context(), &accountsv1.GetAccountRequest{
		AccountId: accountId.String(),
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, mapAccountToResponse(resp.Account))
}

func (s *Server) DeleteAccount(c *gin.Context, accountId AccountIdPath) {
	_, userCompanyID, role := getContext(c)
	if role != "company_admin" && role != "admin" {
		sendError(c, http.StatusForbidden, "FORBIDDEN", "only company_admin can deactivate accounts")
		return
	}

	_, err := s.accountClient.SetAccountInActive(c.Request.Context(), &accountsv1.SetAccountInActiveRequest{
		AccountId: accountId.String(),
		CompanyId: userCompanyID.String(),
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (s *Server) PatchAccount(c *gin.Context, accountId AccountIdPath) {
	_, _, role := getContext(c)
	if role != "company_admin" && role != "admin" {
		sendError(c, http.StatusForbidden, "FORBIDDEN", "only company_admin can update accounts")
		return
	}

	var req PatchAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	if req.Status != nil {
		protoStatus := toProtoAccountStatus(*req.Status)
		_, err := s.accountClient.UpdateAccountStatus(c.Request.Context(), &accountsv1.UpdateAccountStatusRequest{
			AccountId: accountId.String(),
			Status:    protoStatus,
		})
		if err != nil {
			handleGRPCError(c, err)
			return
		}
	}

	resp, err := s.accountClient.GetAccount(c.Request.Context(), &accountsv1.GetAccountRequest{
		AccountId: accountId.String(),
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	accResp := mapAccountToResponse(resp.Account)
	if req.Name != nil {
		accResp.Name = req.Name
	}
	c.JSON(http.StatusOK, accResp)
}

func (s *Server) UpdateAccount(c *gin.Context, accountId AccountIdPath) {
	_, _, role := getContext(c)
	if role != "company_admin" && role != "admin" {
		sendError(c, http.StatusForbidden, "FORBIDDEN", "only company_admin can update accounts")
		return
	}

	var req UpdateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	protoStatus := toProtoAccountStatus(req.Status)
	_, err := s.accountClient.UpdateAccountStatus(c.Request.Context(), &accountsv1.UpdateAccountStatusRequest{
		AccountId: accountId.String(),
		Status:    protoStatus,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	resp, err := s.accountClient.GetAccount(c.Request.Context(), &accountsv1.GetAccountRequest{
		AccountId: accountId.String(),
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	accResp := mapAccountToResponse(resp.Account)
	accResp.Name = &req.Name
	c.JSON(http.StatusOK, accResp)
}

func (s *Server) GenerateAccountStatement(c *gin.Context, accountId AccountIdPath) {
	userID, userCompanyID, _ := getContext(c)

	var req GenerateStatementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	resp, err := s.accountClient.GenerateStatement(c.Request.Context(), &accountsv1.GenerateStatementRequest{
		AccountId:   accountId.String(),
		CompanyId:   userCompanyID.String(),
		InitiatorId: userID.String(),
		PeriodFrom:  timestamppb.New(req.PeriodFrom),
		PeriodTo:    timestamppb.New(req.PeriodTo),
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	entries := make([]StatementEntry, len(resp.Statement.GetEntries()))
	for i, entry := range resp.Statement.GetEntries() {
		var d time.Time
		if entry.Date != nil {
			d = entry.Date.AsTime()
		}
		entries[i] = StatementEntry{
			Date:         d,
			EntryType:    entry.EntryType.String(),
			Amount:       entry.Amount,
			BalanceAfter: entry.BalanceAfter,
			Counterparty: entry.Counterparty,
		}
	}

	stmtID, _ := uuid.Parse(resp.Statement.StatementId)
	accID, _ := uuid.Parse(resp.Statement.AccountId)
	compID, _ := uuid.Parse(resp.Statement.CompanyId)
	initID, _ := uuid.Parse(resp.Statement.InitiatorId)

	var pFrom, pTo time.Time
	if resp.Statement.PeriodFrom != nil {
		pFrom = resp.Statement.PeriodFrom.AsTime()
	}
	if resp.Statement.PeriodTo != nil {
		pTo = resp.Statement.PeriodTo.AsTime()
	}

	c.JSON(http.StatusOK, StatementResponse{
		StatementId:    stmtID,
		AccountId:      accID,
		CompanyId:      compID,
		InitiatorId:    initID,
		Currency:       parseProtoCurrency(resp.Statement.Currency),
		PeriodFrom:     pFrom,
		PeriodTo:       pTo,
		OpeningBalance: resp.Statement.OpeningBalance,
		ClosingBalance: resp.Statement.ClosingBalance,
		TotalDebit:     resp.Statement.TotalDebit,
		TotalCredit:    resp.Statement.TotalCredit,
		Entries:        entries,
	})
}

func (s *Server) ListAccountTransactions(c *gin.Context, accountId AccountIdPath, params ListAccountTransactionsParams) {
	resp, err := s.txClient.GetAllTransactions(c.Request.Context(), &transactionsv1.GetAllTransactionsRequest{
		AccountId: accountId.String(),
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	allTxs := make([]TransactionResponse, len(resp.Transactions))
	for i, t := range resp.Transactions {
		allTxs[i] = mapTxToResponse(t)
	}

	total := len(allTxs)
	offset := 0
	if params.Offset != nil {
		offset = *params.Offset
	}
	limit := 20
	if params.Limit != nil {
		limit = *params.Limit
	}
	if offset > total {
		offset = total
	}
	end := offset + limit
	if end > total {
		end = total
	}

	c.JSON(http.StatusOK, TransactionListResponse{
		Items: allTxs[offset:end],
		Total: total,
	})
}
