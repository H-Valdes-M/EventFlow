package usecase

import (
	"context"
	"eventflow/internal/health/application/dto"
	domain "eventflow/internal/health/domain/repository"
)

type HealthUseCase struct {
	repo domain.HealthRepository
}

// Constructor del Caso de Uso
func NewHealthUseCase(repo domain.HealthRepository) *HealthUseCase {
	return &HealthUseCase{repo: repo}
}

func (uc *HealthUseCase) Execute(ctx context.Context) (*dto.HealthResponse, error) {

	dbStatus := "connected"

	if err := uc.repo.Ping(ctx); err != nil {
		dbStatus = "disconnected"
	}

	// Aqui podriamos agregar otras validaciones:
	// - Verificar conexión a DB
	// - Verificación con servicios externos
	// Si alguno falla, se retorna error HTTP 503
	return dto.NewHealthResponse(dbStatus), nil
}
