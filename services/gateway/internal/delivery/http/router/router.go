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
	systemHandler *handlers.SystemHandler,
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

	r.POST("/api/v1/system/init", systemHandler.Init)

	api := r.Group("/api/v1")
	api.Use(middleware.Auth(validator))
	{
		api.POST("/auth/logout", authHandler.Logout)

		// only admin routes
		systemRoutes := api.Group("/system")
		systemRoutes.Use(middleware.RequireRole("admin"))
		{
			systemRoutes.POST("/admins", systemHandler.AddAdmin)
		}

		// только company_admin
		companyAdminRoutes := api.Group("")
		companyAdminRoutes.Use(middleware.RequireRole("company_admin"))
		{
			companyAdminRoutes.POST("/accounts", accountHandler.CreateAccount)
			companyAdminRoutes.POST("/accounts/bank", accountHandler.LinkBankAccount)
			companyAdminRoutes.POST("/accounts/bank/deposit", accountHandler.MakeBankDeposit)
			companyAdminRoutes.POST("/accounts/bank/withdrawal", accountHandler.MakeBankWithdrawal)

			companyAdminRoutes.POST("/transactions/transfer", txHandler.Transfer)
		}

		// company_admin и admin
		sharedAdminRoutes := api.Group("")
		sharedAdminRoutes.Use(middleware.RequireRole("company_admin", "admin"))
		{
			sharedAdminRoutes.GET("/companies/users", authHandler.GetUsersByCompanyId)
			sharedAdminRoutes.GET("/companies/invite-code", authHandler.GetInviteCode)
			sharedAdminRoutes.DELETE("/accounts/:account_id", accountHandler.SetAccountInActive)
		}

		// all auth users
		api.GET("/companies/accounts", accountHandler.GetAccounts)
		api.GET("/companies/banks", accountHandler.GetBankAccounts)

		api.GET("/accounts/:account_id/balance", accountHandler.GetBalance)

		api.GET("/transactions/:account_id", txHandler.GetAllTransactions)
		api.GET("/transactions/detail/:tx_id", txHandler.GetTransaction)
	}

	return r
}
