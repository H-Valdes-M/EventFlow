package middleware

import (
	"eventflow/pkg/apperrors"
	"eventflow/pkg/httputil"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ErrorHandlerMiddleware maneja errores globalmente
func ErrorHandlerMiddleware(log *zap.Logger) gin.HandlerFunc {
	if log == nil {
		log = zap.NewNop()
	}

	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		var err error

		switch e := recovered.(type) {
		case error:
			err = e
		default:
			err = apperrors.New(
				apperrors.InternalError,
				"PANIC_001",
				"Ocurrio un error interno",
			)
		}

		requestID := c.GetHeader("X-Request-ID")
		statusCode, errorResponse := httputil.MapErrorToHTTP(err, requestID, time.Now())

		if statusCode >= 500 {
			log.Error("Server error",
				zap.String("error", err.Error()),
				zap.String("path", c.Request.URL.Path),
				zap.String("method", c.Request.Method),
				zap.Int("status", statusCode),
			)
		} else {
			log.Warn("Client error",
				zap.String("error", err.Error()),
				zap.String("path", c.Request.URL.Path),
				zap.String("method", c.Request.Method),
				zap.Int("status", statusCode),
			)
		}

		c.Abort()
		c.JSON(statusCode, errorResponse)
	})
}
