package httpapi

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func respond[T any](c *gin.Context, status int, data T) {
	c.JSON(status, gin.H{
		"code":       "OK",
		"message":    "success",
		"request_id": requestID(),
		"data":       data,
	})
}

func respondError(c *gin.Context, status int, message string) {
	respondErrorCode(c, status, http.StatusText(status), message)
}

func respondErrorCode(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, gin.H{
		"code":       code,
		"message":    message,
		"request_id": requestID(),
	})
}

func requestID() string {
	return time.Now().UTC().Format("20060102150405.000000")
}
