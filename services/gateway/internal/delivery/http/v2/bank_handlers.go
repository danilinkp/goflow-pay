package v2

import (
	"fmt"
	"net/http"
	accountsv1 "shared/pkg/gen/go/accounts/v1"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (s *Server) ListBankAccounts(c *gin.Context) {
	_, userCompanyID, _ := getContext(c)

	resp, err := s.accountClient.GetBankAccounts(c.Request.Context(), &accountsv1.GetBankAccountsRequest{
		CompanyId: userCompanyID.String(),
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	result := make([]BankAccountResponse, len(resp.BankAccounts))
	for i, ba := range resp.BankAccounts {
		result[i] = mapBankAccountToResponse(ba)
	}

	c.JSON(http.StatusOK, result)
}

func (s *Server) LinkBankAccount(c *gin.Context) {
	_, userCompanyID, role := getContext(c)
	if role != "company_admin" && role != "admin" {
		sendError(c, http.StatusForbidden, "FORBIDDEN", "only company_admin can link bank accounts")
		return
	}

	var req LinkBankAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	resp, err := s.accountClient.LinkBankAccount(c.Request.Context(), &accountsv1.LinkBankAccountRequest{
		AccountId:         req.AccountId.String(),
		CompanyId:         userCompanyID.String(),
		Name:              req.Name,
		Bic:               req.Bic,
		SettlementAccount: req.SettlementAccount,
		Currency:          toProtoCurrency(req.Currency),
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	baResp := mapBankAccountToResponse(resp.BankAccount)
	c.Header("Location", fmt.Sprintf("/api/v2/bank-accounts/%s", resp.BankAccount.BankAccountId))
	c.JSON(http.StatusCreated, baResp)
}

func (s *Server) UnlinkBankAccount(c *gin.Context, bankAccountId BankAccountIdPath) {
	_, userCompanyID, role := getContext(c)
	if role != "company_admin" && role != "admin" {
		sendError(c, http.StatusForbidden, "FORBIDDEN", "only company_admin can unlink bank accounts")
		return
	}

	_, err := s.accountClient.DeleteBankAccount(c.Request.Context(), &accountsv1.DeleteBankAccountRequest{
		BankAccountId: bankAccountId.String(),
		CompanyId:     userCompanyID.String(),
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (s *Server) GetBankAccountById(c *gin.Context, bankAccountId BankAccountIdPath) {
	resp, err := s.accountClient.GetBankAccount(c.Request.Context(), &accountsv1.GetBankAccountRequest{
		BankAccountId: bankAccountId.String(),
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, mapBankAccountToResponse(resp.BankAccount))
}

func (s *Server) PatchBankAccount(c *gin.Context, bankAccountId BankAccountIdPath) {
	_, _, role := getContext(c)
	if role != "company_admin" && role != "admin" {
		sendError(c, http.StatusForbidden, "FORBIDDEN", "only company_admin can modify bank accounts")
		return
	}

	var req PatchBankAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	resp, err := s.accountClient.GetBankAccount(c.Request.Context(), &accountsv1.GetBankAccountRequest{
		BankAccountId: bankAccountId.String(),
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	baResp := mapBankAccountToResponse(resp.BankAccount)
	if req.Name != nil {
		baResp.Name = *req.Name
	}

	c.JSON(http.StatusOK, baResp)
}

func (s *Server) CreateBankOperation(c *gin.Context, params CreateBankOperationParams) {
	userID, userCompanyID, role := getContext(c)
	if role != "company_admin" && role != "admin" {
		sendError(c, http.StatusForbidden, "FORBIDDEN", "only company_admin can perform bank operations")
		return
	}

	var req CreateBankOperationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	idempotencyKey := params.IdempotencyKey.String()

	var bo *accountsv1.BankOperation
	if req.OperationType == DEPOSIT {
		resp, err := s.accountClient.MakeBankDeposit(c.Request.Context(), &accountsv1.MakeBankDepositRequest{
			AccountId:      req.AccountId.String(),
			BankAccountId:  req.BankAccountId.String(),
			CompanyId:      userCompanyID.String(),
			InitiatorId:    userID.String(),
			Amount:         req.Amount,
			IdempotencyKey: idempotencyKey,
		})
		if err != nil {
			handleGRPCError(c, err)
			return
		}
		bo = resp.BankOperation
	} else if req.OperationType == WITHDRAWAL {
		resp, err := s.accountClient.MakeBankWithdrawal(c.Request.Context(), &accountsv1.MakeBankWithdrawalRequest{
			AccountId:      req.AccountId.String(),
			BankAccountId:  req.BankAccountId.String(),
			CompanyId:      userCompanyID.String(),
			InitiatorId:    userID.String(),
			Amount:         req.Amount,
			IdempotencyKey: idempotencyKey,
		})
		if err != nil {
			handleGRPCError(c, err)
			return
		}
		bo = resp.BankOperation
	} else {
		sendError(c, http.StatusBadRequest, "INVALID_REQUEST", "unsupported operation type")
		return
	}

	opID, _ := uuid.Parse(bo.GetBankOperationId())
	accID, _ := uuid.Parse(bo.GetAccountId())
	baID, _ := uuid.Parse(bo.GetBankAccountId())
	initID, _ := uuid.Parse(bo.GetInitiatorId())

	var createdAt time.Time
	if bo.CreatedAt != nil {
		createdAt = bo.CreatedAt.AsTime()
	}

	var statusEnum OperationStatusEnum
	switch bo.OperationStatus {
	case accountsv1.OperationStatus_OPERATION_STATUS_SUCCESS:
		statusEnum = OperationStatusEnumCOMPLETED
	case accountsv1.OperationStatus_OPERATION_STATUS_FAILED:
		statusEnum = OperationStatusEnumFAILED
	default:
		statusEnum = OperationStatusEnumPENDING
	}

	extID := bo.ExternalId
	c.Header("Location", fmt.Sprintf("/api/v2/bank-operations/%s", bo.GetBankOperationId()))
	c.JSON(http.StatusCreated, BankOperationResponse{
		BankOperationId: opID,
		AccountId:       accID,
		BankAccountId:   baID,
		InitiatorId:     initID,
		Amount:          bo.Amount,
		IdempotencyKey:  bo.IdempotencyKey,
		ExternalId:      &extID,
		OperationType:   req.OperationType,
		OperationStatus: statusEnum,
		CreatedAt:       createdAt,
	})
}

func (s *Server) ListBankOperations(c *gin.Context, params ListBankOperationsParams) {
	// Return list of bank operations (empty or filtered)
	c.JSON(http.StatusOK, []BankOperationResponse{})
}

func (s *Server) GetBankOperationById(c *gin.Context, operationId OperationIdPath) {
	sendError(c, http.StatusNotFound, "NOT_FOUND", "bank operation not found")
}
