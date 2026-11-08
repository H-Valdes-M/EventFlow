package repository

import (
	"context"
	"errors"

	"eventflow/internal/order/domain/entity"
)

// ErrOrderNotFound distingue la ausencia de un pedido de un fallo operativo.
var ErrOrderNotFound = errors.New("pedido no encontrado")

var ErrIdempotencyConflict = errors.New("clave de idempotencia utilizada con otro pedido")

type IdempotentOrderRepository interface {
	SaveIdempotent(context.Context, entity.Order, entity.OrderCreatedEvent, string, string) (entity.Order, error)
}

// OrderRepository define la persistencia necesaria por los casos de uso.
type OrderRepository interface {
	Save(context.Context, entity.Order) error
	SaveWithEvent(context.Context, entity.Order, entity.OrderCreatedEvent) error
	FindByID(context.Context, string) (*entity.Order, error)
	FindAll(context.Context) ([]*entity.Order, error)
}
