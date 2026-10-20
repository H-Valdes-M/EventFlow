package router

import (
	"eventflow/internal/health/presentation/handler"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.Engine, h *handler.HealthHandler) {
	router.GET("/healthz", h.Health)
}
