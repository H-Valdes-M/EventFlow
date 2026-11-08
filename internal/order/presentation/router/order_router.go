package router

import (
	"eventflow/internal/order/presentation/handler"

	"github.com/gin-gonic/gin"
)

// Register publica las rutas disponibles del módulo de pedidos.
func Register(routes gin.IRoutes, orderHandler *handler.OrderHandler) {
	routes.POST("/orders", orderHandler.Create)
	routes.GET("/orders", orderHandler.GetAll)
	routes.GET("/orders/:id", orderHandler.GetByID)
}
