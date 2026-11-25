package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const readOnlyMessage = "read-only instance: writes are not accepted"

func ReadOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		switch c.Request.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			c.Next()
			return
		}

		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
			"code":  "SERVICE_UNAVAILABLE",
			"error": readOnlyMessage,
		})
	}
}
