package routes

import (
	"eventflow/cmd/container"

	"github.com/gin-gonic/gin"
)

// APIRouter coordina el registro de las rutas de los módulos disponibles.
type APIRouter struct {
	container *container.Container
}

func NewAPIRouter(cont *container.Container) *APIRouter {
	return &APIRouter{container: cont}
}

func (r *APIRouter) RegisterRoutes(router *gin.Engine) {
	ConfigureModule(router, r.container)
}
