package router

import (
	"gateway/internal/delivery/http/handlers"
	"gateway/internal/delivery/http/middleware"

	jwtValidator "shared/pkg/jwt"

	"github.com/gin-gonic/gin"
)

func NewRouter(
	validator *jwtValidator.Validator,
	authHandler *handlers.AuthHandler,
	accountHandler *handlers.AccountHandler,
	txHandler *handlers.TransactionHandler,
) *gin.Engine {
	r := gin.New()

	r.Use(gin.Recovery())

	// public
	auth := r.Group("/api/v1/auth")
	{
		auth.POST("/register/company", authHandler.RegisterWithNewCompany)
		auth.POST("/register", authHandler.RegisterWithExistingCompany)
		auth.POST("/login", authHandler.Login)
	}

	// private
	// TODO: Проверить company_id, правильнее его как будто просто через JWT передавать.
	api := r.Group("/api/v1")
	api.Use(middleware.Auth(validator))
	{
		api.POST("/auth/logout", authHandler.Logout)

		// only company_admin и admin
		adminRoutes := api.Group("")
		adminRoutes.Use(middleware.RequireRole("company_admin", "admin"))
		{
			adminRoutes.GET("/companies/:company_id/users", authHandler.GetUsersByCompanyId)
			adminRoutes.GET("/companies/:company_id/invite-code", authHandler.GetInviteCode)
			adminRoutes.POST("/accounts", accountHandler.CreateAccount)
			adminRoutes.DELETE("/accounts/:account_id", accountHandler.SetAccountInActive)
			adminRoutes.POST("/accounts/bank", accountHandler.LinkBankAccount)
			adminRoutes.POST("/accounts/bank/deposit", accountHandler.MakeBankDeposit)
			adminRoutes.POST("/accounts/bank/withdrawal", accountHandler.MakeBankWithdrawal)
		}

		// all auth users
		api.GET("/accounts/:company_id", accountHandler.GetAccounts)
		api.GET("/accounts/:account_id/balance", accountHandler.GetBalance)
		api.GET("/accounts/:company_id/bank", accountHandler.GetBankAccounts)
		api.POST("/transactions/transfer", txHandler.Transfer)
		api.GET("/transactions/:account_id", txHandler.GetAllTransactions)
		api.GET("/transactions/detail/:tx_id", txHandler.GetTransaction)
	}

	return r
}
