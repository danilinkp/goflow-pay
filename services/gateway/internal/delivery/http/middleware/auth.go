package middleware

import (
	"gateway/internal/clients/grpc"
	"net/http"
	authv1 "shared/pkg/gen/go/auth/v1"
	"strings"

	jwtValidator "shared/pkg/jwt"

	"github.com/gin-gonic/gin"
)

const (
	ContextUserID    = "user_id"
	ContextCompanyID = "company_id"
	ContextRole      = "role"
)

func Auth(validator *jwtValidator.Validator, authClient *grpc.AuthGRPCClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractToken(c)
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			return
		}

		claims, err := validator.Validate(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		_, err = authClient.IsTokenValid(c.Request.Context(), &authv1.IsTokenValidRequest{Token: token})
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token revoked"})
			return
		}

		c.Set(ContextUserID, claims.UserID)
		c.Set(ContextCompanyID, claims.CompanyID)
		c.Set(ContextRole, claims.Role)

		c.Next()
	}
}

func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get(ContextRole)
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		roleStr := role.(string)
		for _, r := range roles {
			if roleStr == r {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
	}
}

func extractToken(c *gin.Context) string {
	bearer := c.GetHeader("Authorization")
	if bearer == "" {
		return ""
	}
	parts := strings.SplitN(bearer, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return ""
	}
	return parts[1]
}
