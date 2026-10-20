package routes

import (
	"eventflow/cmd/container"
	healthrouter "eventflow/internal/health/presentation/router"

	"github.com/gin-gonic/gin"
)

// ConfigureModule registra únicamente los módulos disponibles. Orders queda
// fuera hasta que se implemente su dominio y sus casos de uso.
func ConfigureModule(router *gin.Engine, c *container.Container) {
	healthrouter.RegisterRoutes(router, c.HealthHandler)
}
