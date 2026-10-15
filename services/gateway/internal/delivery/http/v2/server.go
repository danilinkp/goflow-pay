package v2

import (
	"gateway/internal/clients/grpc"
	"gateway/internal/delivery/http/middleware"
	"net/http"
	accountsv1 "shared/pkg/gen/go/accounts/v1"
	transactionsv1 "shared/pkg/gen/go/transactions/v1"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	authClient    *grpc.AuthGRPCClient
	accountClient *grpc.AccountGRPCClient
	txClient      *grpc.TransactionGRPCClient
}

func NewServer(
	authClient *grpc.AuthGRPCClient,
	accountClient *grpc.AccountGRPCClient,
	txClient *grpc.TransactionGRPCClient,
) *Server {
	return &Server{
		authClient:    authClient,
		accountClient: accountClient,
		txClient:      txClient,
	}
}

func sendError(c *gin.Context, statusCode int, code string, msg string) {
	c.JSON(statusCode, ErrorResponse{
		Code:  code,
		Error: msg,
	})
}

func handleGRPCError(c *gin.Context, err error) {
	st, ok := status.FromError(err)
	if !ok {
		sendError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	switch st.Code() {
	case codes.NotFound:
		sendError(c, http.StatusNotFound, "NOT_FOUND", st.Message())
	case codes.AlreadyExists:
		sendError(c, http.StatusConflict, "CONFLICT", st.Message())
	case codes.InvalidArgument:
		sendError(c, http.StatusBadRequest, "BAD_REQUEST", st.Message())
	case codes.Unauthenticated:
		sendError(c, http.StatusUnauthorized, "UNAUTHORIZED", st.Message())
	case codes.PermissionDenied:
		sendError(c, http.StatusForbidden, "FORBIDDEN", st.Message())
	default:
		sendError(c, http.StatusInternalServerError, "INTERNAL_ERROR", st.Message())
	}
}

func getContext(c *gin.Context) (userID uuid.UUID, companyID uuid.UUID, role string) {
	if uid, ok := c.Get(middleware.ContextUserID); ok {
		if u, ok := uid.(uuid.UUID); ok {
			userID = u
		}
	}
	if cid, ok := c.Get(middleware.ContextCompanyID); ok {
		if comp, ok := cid.(uuid.UUID); ok {
			companyID = comp
		}
	}
	if r, ok := c.Get(middleware.ContextRole); ok {
		if s, ok := r.(string); ok {
			role = s
		}
	}
	return
}

func parseProtoCurrency(c accountsv1.Currency) CurrencyEnum {
	switch c {
	case accountsv1.Currency_CURRENCY_EUR:
		return EUR
	case accountsv1.Currency_CURRENCY_USD:
		return USD
	case accountsv1.Currency_CURRENCY_RUB:
		return RUB
	case accountsv1.Currency_CURRENCY_CNY:
		return CNY
	default:
		return RUB
	}
}

func toProtoCurrency(c CurrencyEnum) accountsv1.Currency {
	switch c {
	case EUR:
		return accountsv1.Currency_CURRENCY_EUR
	case USD:
		return accountsv1.Currency_CURRENCY_USD
	case RUB:
		return accountsv1.Currency_CURRENCY_RUB
	case CNY:
		return accountsv1.Currency_CURRENCY_CNY
	default:
		return accountsv1.Currency_CURRENCY_UNSPECIFIED
	}
}

func toProtoTxCurrency(c CurrencyEnum) transactionsv1.Currency {
	switch c {
	case EUR:
		return transactionsv1.Currency_CURRENCY_EUR
	case USD:
		return transactionsv1.Currency_CURRENCY_USD
	case RUB:
		return transactionsv1.Currency_CURRENCY_RUB
	case CNY:
		return transactionsv1.Currency_CURRENCY_CNY
	default:
		return transactionsv1.Currency_CURRENCY_UNSPECIFIED
	}
}

func parseProtoAccountStatus(s accountsv1.AccountStatus) AccountStatusEnum {
	switch s {
	case accountsv1.AccountStatus_ACCOUNT_STATUS_ACTIVE:
		return ACTIVE
	case accountsv1.AccountStatus_ACCOUNT_STATUS_INACTIVE:
		return INACTIVE
	default:
		return INACTIVE
	}
}

func toProtoAccountStatus(s AccountStatusEnum) accountsv1.AccountStatus {
	switch s {
	case ACTIVE:
		return accountsv1.AccountStatus_ACCOUNT_STATUS_ACTIVE
	case INACTIVE, BLOCKED:
		return accountsv1.AccountStatus_ACCOUNT_STATUS_INACTIVE
	default:
		return accountsv1.AccountStatus_ACCOUNT_STATUS_UNSPECIFIED
	}
}

func mapAccountToResponse(a *accountsv1.Account) AccountResponse {
	accID, _ := uuid.Parse(a.Id)
	compID, _ := uuid.Parse(a.CompanyId)
	var createdAt time.Time
	if a.CreatedAt != nil {
		createdAt = a.CreatedAt.AsTime()
	}
	return AccountResponse{
		AccountId: accID,
		CompanyId: compID,
		Balance:   a.Balance,
		Currency:  parseProtoCurrency(a.Currency),
		Status:    parseProtoAccountStatus(a.Status),
		CreatedAt: createdAt,
	}
}

func mapBankAccountToResponse(ba *accountsv1.BankAccount) BankAccountResponse {
	baID, _ := uuid.Parse(ba.BankAccountId)
	compID, _ := uuid.Parse(ba.CompanyId)
	var createdAt time.Time
	if ba.CreatedAt != nil {
		createdAt = ba.CreatedAt.AsTime()
	}
	return BankAccountResponse{
		BankAccountId:     baID,
		CompanyId:         compID,
		Name:              ba.Name,
		Bic:               ba.Bic,
		SettlementAccount: ba.SettlementAccount,
		Currency:          parseProtoCurrency(ba.Currency),
		CreatedAt:         createdAt,
	}
}

func mapTxToResponse(t *transactionsv1.Transaction) TransactionResponse {
	txID, _ := uuid.Parse(t.TransactionId)
	initID, _ := uuid.Parse(t.InitiatorId)
	fromID, _ := uuid.Parse(t.FromAccountId)
	toID, _ := uuid.Parse(t.ToAccountId)
	var createdAt time.Time
	if t.CreatedAt != nil {
		createdAt = t.CreatedAt.AsTime()
	}

	var status TransactionStatusEnum
	switch t.TransactionStatus {
	case transactionsv1.TransactionStatus_TRANSACTION_STATUS_SUCCESS:
		status = TransactionStatusEnumCOMPLETED
	case transactionsv1.TransactionStatus_TRANSACTION_STATUS_FAILED:
		status = TransactionStatusEnumFAILED
	case transactionsv1.TransactionStatus_TRANSACTION_STATUS_PENDING, transactionsv1.TransactionStatus_TRANSACTION_STATUS_PROCESSING:
		status = TransactionStatusEnumPENDING
	default:
		status = TransactionStatusEnumPENDING
	}

	var curr CurrencyEnum
	switch t.Currency {
	case transactionsv1.Currency_CURRENCY_EUR:
		curr = EUR
	case transactionsv1.Currency_CURRENCY_USD:
		curr = USD
	case transactionsv1.Currency_CURRENCY_RUB:
		curr = RUB
	case transactionsv1.Currency_CURRENCY_CNY:
		curr = CNY
	default:
		curr = RUB
	}

	return TransactionResponse{
		TransactionId:     txID,
		InitiatorId:       initID,
		FromAccountId:     fromID,
		ToAccountId:       toID,
		Amount:            t.Amount,
		Currency:          curr,
		IdempotencyKey:    t.IdempotencyKey,
		TransactionStatus: status,
		CreatedAt:         createdAt,
	}
}
