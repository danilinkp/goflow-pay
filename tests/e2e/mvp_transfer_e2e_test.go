//go:build e2e

package e2e_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	allure "github.com/allure-framework/allure-go/commons/gotest"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getGatewayURL() string {
	url := os.Getenv("GATEWAY_URL")
	if url == "" {
		return "http://127.0.0.1:8080"
	}
	return url
}

type httpClient struct {
	baseURL string
	client  *http.Client
	token   string
}

func newHTTPClient(baseURL string) *httpClient {
	return &httpClient{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *httpClient) do(method, path string, body interface{}, target interface{}) (int, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return 0, fmt.Errorf("marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, c.baseURL+path, bodyReader)
	if err != nil {
		return 0, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, fmt.Errorf("read response body: %w", err)
	}

	if target != nil && len(respBytes) > 0 {
		if err := json.Unmarshal(respBytes, target); err != nil {
			return resp.StatusCode, fmt.Errorf("unmarshal response (%s): %w", string(respBytes), err)
		}
	}

	return resp.StatusCode, nil
}

// DTO responses
type authResponse struct {
	UserID    string `json:"user_id"`
	CompanyID string `json:"company_id"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	Token     string `json:"token"`
}

type accountResponse struct {
	AccountID string    `json:"account_id"`
	CompanyID string    `json:"company_id"`
	Balance   int64     `json:"balance"`
	Currency  string    `json:"currency"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type bankAccountResponse struct {
	BankAccountID     string    `json:"bank_account_id"`
	CompanyID         string    `json:"company_id"`
	Name              string    `json:"name"`
	BIC               string    `json:"bic"`
	SettlementAccount string    `json:"settlement_account"`
	Currency          string    `json:"currency"`
	CreatedAt         time.Time `json:"created_at"`
}

type bankOperationResponse struct {
	BankOperationID string    `json:"bank_operation_id"`
	AccountID       string    `json:"account_id"`
	BankAccountID   string    `json:"bank_account_id"`
	InitiatorID     string    `json:"initiator_id"`
	OperationType   string    `json:"operation_type"`
	OperationStatus string    `json:"operation_status"`
	Amount          int64     `json:"amount"`
	IdempotencyKey  string    `json:"idempotency_key"`
	ExternalID      string    `json:"external_id"`
	CreatedAt       time.Time `json:"created_at"`
}

type transactionResponse struct {
	TransactionID     string    `json:"transaction_id"`
	InitiatorID       string    `json:"initiator_id"`
	FromAccountID     string    `json:"from_account_id"`
	ToAccountID       string    `json:"to_account_id"`
	Amount            int64     `json:"amount"`
	Currency          string    `json:"currency"`
	IdempotencyKey    string    `json:"idempotency_key"`
	TransactionStatus string    `json:"transaction_status"`
	CreatedAt         time.Time `json:"created_at"`
}

type balanceResponse struct {
	Balance int64 `json:"balance"`
}

type statementEntryResponse struct {
	Date         time.Time `json:"date"`
	EntryType    string    `json:"entry_type"`
	Amount       int64     `json:"amount"`
	BalanceAfter int64     `json:"balance_after"`
	Counterparty string    `json:"counterparty"`
}

type statementResponse struct {
	StatementID    string                   `json:"statement_id"`
	AccountID      string                   `json:"account_id"`
	CompanyID      string                   `json:"company_id"`
	InitiatorID    string                   `json:"initiator_id"`
	PeriodFrom     time.Time                `json:"period_from"`
	PeriodTo       time.Time                `json:"period_to"`
	OpeningBalance int64                    `json:"opening_balance"`
	ClosingBalance int64                    `json:"closing_balance"`
	TotalDebit     int64                    `json:"total_debit"`
	TotalCredit    int64                    `json:"total_credit"`
	Currency       string                   `json:"currency"`
	Entries        []statementEntryResponse `json:"entries"`
}

func TestE2E_CompanyTransferAndStatementFlow(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		if os.Getenv("SKIP_E2E_TESTS") == "true" {
			t.Skip("Этап сквозного E2E тестирования пропущен: сбой на предыдущем этапе пирамиды тестирования")
			return
		}
		baseURL := getGatewayURL()
		clientA := newHTTPClient(baseURL)
		clientB := newHTTPClient(baseURL)

		runID := uuid.New().String()[:8]
		companyAName := fmt.Sprintf("Acme Corp E2E %s", runID)
		companyBName := fmt.Sprintf("Beta LLC E2E %s", runID)
		adminAEmail := fmt.Sprintf("admin_a_%s@example.com", runID)
		adminBEmail := fmt.Sprintf("admin_b_%s@example.com", runID)
		password := "SecurePass123!"

		depositAmount := int64(50000)
		transferAmount := int64(15000)
		expectedClosingBalance := depositAmount - transferAmount

		var authA authResponse
		var accountA accountResponse
		var bankAccountA bankAccountResponse
		var depositOp bankOperationResponse

		var authB authResponse
		var accountB accountResponse
		var txResp transactionResponse
		var statementResp statementResponse

		// Arrange
		a.Step("Arrange: Проверка доступности API Gateway и подготовка тестовых учетных записей", func(a *allure.Context) {
			resp, err := http.Get(baseURL + "/swagger")
			require.NoError(t, err, "Gateway должен быть доступен")
			require.Equal(t, http.StatusOK, resp.StatusCode, "Gateway Swagger должен возвращать HTTP 200")
			resp.Body.Close()
		})

		// Act 1: Регистрация компании А
		var statusAuthA int
		var errAuthA error
		a.Step("Act 1: Регистрация компании-отправителя (Company A) и получение JWT токена", func(a *allure.Context) {
			regReq := map[string]string{
				"login":        "admin_a_" + runID,
				"email":        adminAEmail,
				"password":     password,
				"company_name": companyAName,
			}
			statusAuthA, errAuthA = clientA.do(http.MethodPost, "/api/v1/auth/register/company", regReq, &authA)
			clientA.token = authA.Token
		})

		// Assert 1: Проверка регистрации компании А
		a.Step("Assert 1: Компания А успешно создана с ролью company_admin, выдан валидный JWT токен", func(a *allure.Context) {
			require.NoError(t, errAuthA)
			assert.Equal(t, http.StatusCreated, statusAuthA)
			assert.NotEmpty(t, authA.Token)
			assert.NotEmpty(t, authA.UserID)
			assert.NotEmpty(t, authA.CompanyID)
			assert.Equal(t, "company_admin", authA.Role)
		})

		// Act 2: Открытие расчетного счета для Компании А
		var statusAccA int
		var errAccA error
		a.Step("Act 2: Открытие расчетного счета в валюте RUB для Компании А", func(a *allure.Context) {
			accReq := map[string]string{
				"currency": "RUB",
			}
			statusAccA, errAccA = clientA.do(http.MethodPost, "/api/v1/accounts", accReq, &accountA)
		})

		// Assert 2: Проверка счета Компании А
		a.Step("Assert 2: Расчетный счет Компании А успешно открыт с нулевым балансом", func(a *allure.Context) {
			require.NoError(t, errAccA)
			assert.Equal(t, http.StatusCreated, statusAccA)
			assert.NotEmpty(t, accountA.AccountID)
			assert.Equal(t, authA.CompanyID, accountA.CompanyID)
			assert.Equal(t, int64(0), accountA.Balance)
			assert.Equal(t, "ACCOUNT_STATUS_ACTIVE", accountA.Status)
		})

		// Act 3: Привязка банковского счета для Компании А
		var statusBankA int
		var errBankA error
		a.Step("Act 3: Привязка внешнего банковского счета к аккаунту Компании А", func(a *allure.Context) {
			linkReq := map[string]interface{}{
				"account_id":         accountA.AccountID,
				"name":               "Sberbank Corporate",
				"bic":                "044525225",
				"settlement_account": "40702643000000000009",
				"currency":           "RUB",
			}
			statusBankA, errBankA = clientA.do(http.MethodPost, "/api/v1/accounts/bank", linkReq, &bankAccountA)
		})

		// Assert 3: Проверка привязки банковского счета
		a.Step("Assert 3: Банковский счет успешно привязан с кодом HTTP 201", func(a *allure.Context) {
			require.NoError(t, errBankA)
			require.Equal(t, http.StatusCreated, statusBankA)
			assert.NotEmpty(t, bankAccountA.BankAccountID)
			assert.Equal(t, "044525225", bankAccountA.BIC)
		})

		// Act 4: Пополнение счета Компании А через банковский депозит (50000 RUB)
		var statusDeposit int
		var errDeposit error
		a.Step("Act 4: Внесение депозита (MakeBankDeposit) на сумму 50 000 RUB на счет Компании А", func(a *allure.Context) {
			depReq := map[string]interface{}{
				"account_id":      accountA.AccountID,
				"bank_account_id": bankAccountA.BankAccountID,
				"amount":          depositAmount,
				"idempotency_key": uuid.New().String(),
			}
			statusDeposit, errDeposit = clientA.do(http.MethodPost, "/api/v1/accounts/bank/deposit", depReq, &depositOp)
		})

		// Assert 4: Проверка статуса депозита и обновленного баланса Компании А
		a.Step("Assert 4: Депозит успешно подтвержден, баланс Компании А равен 50 000 RUB", func(a *allure.Context) {
			require.NoError(t, errDeposit)
			assert.Equal(t, http.StatusOK, statusDeposit)
			assert.Equal(t, "OPERATION_STATUS_SUCCESS", depositOp.OperationStatus)
			assert.Equal(t, depositAmount, depositOp.Amount)

			var balResp balanceResponse
			balStatus, balErr := clientA.do(http.MethodGet, fmt.Sprintf("/api/v1/accounts/%s/balance", accountA.AccountID), nil, &balResp)
			require.NoError(t, balErr)
			assert.Equal(t, http.StatusOK, balStatus)
			assert.Equal(t, depositAmount, balResp.Balance)
		})

		// Act 5: Регистрация компании Б и открытие счета в RUB
		var statusAuthB int
		var errAuthB error
		var statusAccB int
		var errAccB error
		a.Step("Act 5: Регистрация компании-получателя (Company B) и открытие расчетного счета в RUB", func(a *allure.Context) {
			regReqB := map[string]string{
				"login":        "admin_b_" + runID,
				"email":        adminBEmail,
				"password":     password,
				"company_name": companyBName,
			}
			statusAuthB, errAuthB = clientB.do(http.MethodPost, "/api/v1/auth/register/company", regReqB, &authB)
			clientB.token = authB.Token

			accReqB := map[string]string{
				"currency": "RUB",
			}
			statusAccB, errAccB = clientB.do(http.MethodPost, "/api/v1/accounts", accReqB, &accountB)
		})

		// Assert 5: Проверка готовности Компании Б
		a.Step("Assert 5: Компания Б успешно зарегистрирована, расчетный счет активен с нулевым балансом", func(a *allure.Context) {
			require.NoError(t, errAuthB)
			assert.Equal(t, http.StatusCreated, statusAuthB)
			assert.NotEmpty(t, authB.Token)

			require.NoError(t, errAccB)
			assert.Equal(t, http.StatusCreated, statusAccB)
			assert.NotEmpty(t, accountB.AccountID)
			assert.Equal(t, int64(0), accountB.Balance)
		})

		// Act 6: Межбанковский перевод 15 000 RUB со счета А на счет Б
		var statusTx int
		var errTx error
		a.Step("Act 6: Выполнение межбанковского перевода (Transfer) на сумму 15 000 RUB с аккаунта А на аккаунт Б", func(a *allure.Context) {
			transferReq := map[string]interface{}{
				"from_account_id": accountA.AccountID,
				"to_account_id":   accountB.AccountID,
				"amount":          transferAmount,
				"currency":        "RUB",
				"idempotency_key": uuid.New().String(),
			}
			statusTx, errTx = clientA.do(http.MethodPost, "/api/v1/transactions/transfer", transferReq, &txResp)
		})

		// Assert 6: Проверка статуса транзакции
		a.Step("Assert 6: Перевод выполнен успешно, статус транзакции TRANSACTION_STATUS_SUCCESS", func(a *allure.Context) {
			require.NoError(t, errTx)
			assert.Equal(t, http.StatusOK, statusTx)
			assert.NotEmpty(t, txResp.TransactionID)
			assert.Equal(t, accountA.AccountID, txResp.FromAccountID)
			assert.Equal(t, accountB.AccountID, txResp.ToAccountID)
			assert.Equal(t, transferAmount, txResp.Amount)
			assert.Equal(t, "TRANSACTION_STATUS_SUCCESS", txResp.TransactionStatus)
		})

		// Act 7: Получение и проверка остатков на обоих счетах после перевода
		var balA balanceResponse
		var balB balanceResponse
		var statusBalA, statusBalB int
		var errBalA, errBalB error
		a.Step("Act 7: Запрос балансов счетов Компании А и Компании Б через REST API", func(a *allure.Context) {
			statusBalA, errBalA = clientA.do(http.MethodGet, fmt.Sprintf("/api/v1/accounts/%s/balance", accountA.AccountID), nil, &balA)
			statusBalB, errBalB = clientB.do(http.MethodGet, fmt.Sprintf("/api/v1/accounts/%s/balance", accountB.AccountID), nil, &balB)
		})

		// Assert 7: Проверка корректности изменения балансов участников
		a.Step("Assert 7: Баланс счета А уменьшился на 15 000 (остаток 35 000 RUB), баланс счета Б равен 15 000 RUB", func(a *allure.Context) {
			require.NoError(t, errBalA)
			assert.Equal(t, http.StatusOK, statusBalA)
			assert.Equal(t, expectedClosingBalance, balA.Balance)

			require.NoError(t, errBalB)
			assert.Equal(t, http.StatusOK, statusBalB)
			assert.Equal(t, transferAmount, balB.Balance)
		})

		// Act 8: Генерация финансовой выписки за период
		var statusStmt int
		var errStmt error
		a.Step("Act 8: Формирование финансовой выписки (Statement) по счету Компании А", func(a *allure.Context) {
			stmtReq := map[string]interface{}{
				"period_from": time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339),
				"period_to":   time.Now().UTC().Add(2 * time.Hour).Format(time.RFC3339),
			}
			statusStmt, errStmt = clientA.do(http.MethodPost, fmt.Sprintf("/api/v1/accounts/%s/statement", accountA.AccountID), stmtReq, &statementResp)
		})

		// Assert 8: Проверка корректности финансовой выписки (дебет, кредит, входящий/исходящий остаток)
		a.Step("Assert 8: Выписка содержит 2 операции: Opening=0, TotalDebit=50000, TotalCredit=15000, Closing=35000", func(a *allure.Context) {
			require.NoError(t, errStmt)
			assert.Equal(t, http.StatusOK, statusStmt)
			assert.NotEmpty(t, statementResp.StatementID)
			assert.Equal(t, accountA.AccountID, statementResp.AccountID)
			assert.Equal(t, authA.CompanyID, statementResp.CompanyID)
			assert.Equal(t, int64(0), statementResp.OpeningBalance)
			assert.Equal(t, depositAmount, statementResp.TotalDebit)
			assert.Equal(t, transferAmount, statementResp.TotalCredit)
			assert.Equal(t, expectedClosingBalance, statementResp.ClosingBalance)
			assert.Len(t, statementResp.Entries, 2)
		})
	},
		allure.WithParentSuite("E2E Тестирование платформы"),
		allure.WithSuite("GoFlow Pay Platform"),
		allure.WithSubSuite("Межбанковские расчеты"),
		allure.WithEpic("GoFlow Pay E2E"),
		allure.WithFeature("Движение денежных средств (MVP)"),
		allure.WithStory("Сквозной сценарий: Регистрация компаний -> Пополнение счета -> Перевод между счетами -> Финансовая выписка"),
	)
}
