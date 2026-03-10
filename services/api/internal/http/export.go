package httpapi

import (
	"github.com/fenco/trademate/services/api/internal/auth"
	"github.com/gin-gonic/gin"
)

func AuthMiddlewareProxy(tokenService *auth.Service) gin.HandlerFunc {
	return authMiddleware(tokenService)
}
