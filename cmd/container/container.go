package container

import (
	"eventflow/internal/health/application/usecase"
	healthinfra "eventflow/internal/health/infrastructure"
	"eventflow/internal/health/presentation/handler"
	"eventflow/pkg/database"
	"eventflow/pkg/logger"

	"go.uber.org/zap"
)

// Container centraliza las dependencias de los módulos actualmente disponibles.
// Orders se incorporará aquí cuando se implemente ese módulo.
type Container struct {
	DB            database.Database
	HealthHandler *handler.HealthHandler
	Log           *zap.Logger
}

func NewContainer(db database.Database, log *zap.Logger) (*Container, error) {
	healthRepo := healthinfra.NewPostgresHealthRepository(db)
	healthUseCase := usecase.NewHealthUseCase(healthRepo)

	if log == nil {
		log = logger.NewNop()
	}
	return &Container{
		DB:            db,
		HealthHandler: handler.NewHealthHandler(healthUseCase),
		Log:           log,
	}, nil
}
