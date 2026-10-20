package handler

import (
	"eventflow/internal/health/application/dto"
	"eventflow/internal/health/application/usecase"

	"net/http"

	"github.com/gin-gonic/gin"
)

type HealthHandler struct {
	uc *usecase.HealthUseCase
}

// Constructor del Controller
func NewHealthHandler(uc *usecase.HealthUseCase) *HealthHandler {
	return &HealthHandler{
		uc: uc,
	}
}

// @Summary Health Check
// @Description Returns the health status of the service
// @Tags Health
// @Produce json
// @Success 200 {object} dto.HealthResponse
// @Failure 503 {object} dto.ErrorResponse
// @Router /healthz [get]
func (h *HealthHandler) Health(c *gin.Context) {
	res, err := h.uc.Execute(c.Request.Context())

	if err != nil {
		errorResponse := dto.NewErrorResponse("Service unhealthy")
		c.JSON(http.StatusServiceUnavailable, errorResponse)
		return
	}

	c.JSON(http.StatusOK, res)
}
