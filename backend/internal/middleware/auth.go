package middleware

import (
	"net/http"
	"strings"

	"qim/internal/pkg/apperr"
	"qim/internal/pkg/jwt"
	"qim/internal/pkg/resp"

	"github.com/gin-gonic/gin"
)

func Auth(manager *jwt.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := bearerToken(c.GetHeader("Authorization"))
		if token == "" {
			token = c.Query("token")
		}
		if token == "" {
			resp.Fail(c, apperr.New(apperr.CodeUnauthorized, "missing token"))
			c.Abort()
			return
		}

		claims, err := manager.Parse(token)
		if err != nil {
			resp.Fail(c, apperr.Wrap(apperr.CodeUnauthorized, "invalid token", err))
			c.Abort()
			return
		}

		c.Set("uid", claims.UID)
		c.Set("username", claims.Username)
		c.Next()
	}
}

func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", c.GetHeader("Origin"))
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Log-ID")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func bearerToken(header string) string {
	if header == "" {
		return ""
	}
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return parts[1]
}
