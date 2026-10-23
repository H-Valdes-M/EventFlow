package container

import (
	"eventflow/internal/health/application/usecase"
	healthinfra "eventflow/internal/health/infrastructure"
	"eventflow/internal/health/presentation/handler"
	orderusecase "eventflow/internal/order/application/usecase"
	orderinfra "eventflow/internal/order/infrastructure/postgres"
	orderhandler "eventflow/internal/order/presentation/handler"
	"eventflow/pkg/database"
	"eventflow/pkg/logger"

	"go.uber.org/zap"
)

// Container centraliza las dependencias de los módulos disponibles.
type Container struct {
	DB            database.Database
	HealthHandler *handler.HealthHandler
	OrderHandler  *orderhandler.OrderHandler
	Log           *zap.Logger
}

func NewContainer(db database.Database, log *zap.Logger) (*Container, error) {
	healthRepo := healthinfra.NewPostgresHealthRepository(db)
	healthUseCase := usecase.NewHealthUseCase(healthRepo)
	orderRepo, err := orderinfra.NewOrderRepository(db)
	if err != nil {
		return nil, err
	}
	orderUseCase := orderusecase.NewCreateOrderUseCase(orderRepo)
	getOrderUseCase := orderusecase.NewGetOrderUseCase(orderRepo)

	if log == nil {
		log = logger.NewNop()
	}
	return &Container{
		DB:            db,
		HealthHandler: handler.NewHealthHandler(healthUseCase),
		OrderHandler:  orderhandler.NewOrderHandler(orderUseCase, getOrderUseCase),
		Log:           log,
	}, nil
}
