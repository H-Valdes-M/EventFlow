package routes

import (
	"eventflow/cmd/container"
	healthrouter "eventflow/internal/health/presentation/router"
	orderrouter "eventflow/internal/order/presentation/router"

	"github.com/gin-gonic/gin"
)

// ConfigureModule registra los módulos disponibles de la API.
func ConfigureModule(router *gin.Engine, c *container.Container) {
	healthrouter.RegisterRoutes(router, c.HealthHandler)
	orderrouter.Register(router, c.OrderHandler)
}
