package router

import (
	"gateway/internal/clients/grpc"
	"gateway/internal/delivery/http/handlers"
	"gateway/internal/delivery/http/middleware"
	v2 "gateway/internal/delivery/http/v2"
	"net/http"

	jwtValidator "shared/pkg/jwt"

	"github.com/gin-gonic/gin"
)

func NewRouter(
	validator *jwtValidator.Validator,
	authHandler *handlers.AuthHandler,
	authClient *grpc.AuthGRPCClient,
	systemHandler *handlers.SystemHandler,
	accountHandler *handlers.AccountHandler,
	txHandler *handlers.TransactionHandler,
	v2Server *v2.Server,
) *gin.Engine {
	r := gin.New()

	r.Use(gin.Recovery())

	// CORS middleware
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, Idempotency-Key")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, PATCH, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	})

	// Swagger UI & OpenAPI Specification routes
	registerSwagger(r)

	// ==================== API v1 (Legacy) ====================
	// public
	auth := r.Group("/api/v1/auth")
	{
		auth.POST("/register/company", authHandler.RegisterWithNewCompany)
		auth.POST("/register", authHandler.RegisterWithExistingCompany)
		auth.POST("/login", authHandler.Login)
	}

	r.POST("/api/v1/system/init", systemHandler.Init)

	api := r.Group("/api/v1")
	api.Use(middleware.Auth(validator, authClient))
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
		api.POST("/accounts/:account_id/statement", accountHandler.GenerateStatement)

		api.GET("/transactions/:account_id", txHandler.GetAllTransactions)
		api.GET("/transactions/detail/:tx_id", txHandler.GetTransaction)
	}

	// ==================== API v2 (REST 3.1) ====================
	if v2Server != nil {
		v2AuthMiddleware := func(c *gin.Context) {
			if _, ok := c.Get(v2.BearerAuthScopes); ok {
				middleware.Auth(validator, authClient)(c)
			}
		}

		v2.RegisterHandlersWithOptions(r, v2Server, v2.GinServerOptions{
			BaseURL: "",
			Middlewares: []v2.MiddlewareFunc{
				v2AuthMiddleware,
			},
		})
	}

	return r
}

func registerSwagger(r *gin.Engine) {
	r.GET("/docs/openapi.json", func(c *gin.Context) {
		spec, err := v2.GetSwagger()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, spec)
	})

	r.GET("/docs/openapi.yaml", func(c *gin.Context) {
		c.File("docs/openapi/openapi.yaml")
	})

	swaggerHTML := `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>GoFlow Pay API Documentation (OpenAPI 3.1)</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" />
  <style>
    body { margin: 0; background: #fafafa; }
    .topbar { display: none !important; }
  </style>
</head>
<body>
<div id="swagger-ui"></div>
<script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js" crossorigin></script>
<script>
  window.onload = () => {
    window.ui = SwaggerUIBundle({
      url: '/docs/openapi.json',
      dom_id: '#swagger-ui',
      presets: [
        SwaggerUIBundle.presets.apis,
        SwaggerUIBundle.SwaggerUIStandalonePreset
      ],
      layout: "BaseLayout",
      deepLinking: true,
      persistAuthorization: true
    });
  };
</script>
</body>
</html>`

	r.GET("/swagger", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(swaggerHTML))
	})
	r.GET("/swagger/*any", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(swaggerHTML))
	})
	r.GET("/docs", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(swaggerHTML))
	})
}
