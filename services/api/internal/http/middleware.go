package httpapi

import (
	"net/http"
	"strings"

	"github.com/fenco/trademate/services/api/internal/auth"
	"github.com/gin-gonic/gin"
)

const (
	ctxUserIDKey      = "user_id"
	ctxActiveStoreKey = "active_store_id"
	ctxRoleCodeKey    = "role_code"
	ctxBearerTokenKey = "bearer_token"
)

func authMiddleware(tokenService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if path == "/health" || path == "/api/v1/auth/login" {
			c.Next()
			return
		}

		token := extractToken(c)
		if token == "" {
			respondErrorCode(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing bearer token")
			return
		}

		claims, err := tokenService.Parse(token)
		if err != nil {
			respondErrorCode(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid token")
			return
		}

		c.Set(ctxUserIDKey, claims.UserID)
		c.Set(ctxActiveStoreKey, claims.ActiveStoreID)
		c.Set(ctxRoleCodeKey, claims.RoleCode)
		c.Set(ctxBearerTokenKey, token)
		c.Next()
	}
}

func extractToken(c *gin.Context) string {
	header := c.GetHeader("Authorization")
	if strings.HasPrefix(header, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
	}

	if c.Request.URL.Path == "/api/v1/ws" {
		return strings.TrimSpace(c.Query("token"))
	}

	return ""
}

func contextValue(c *gin.Context, key string) string {
	value, exists := c.Get(key)
	if !exists {
		return ""
	}
	str, ok := value.(string)
	if !ok {
		return ""
	}
	return str
}
