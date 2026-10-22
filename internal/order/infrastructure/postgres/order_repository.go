package postgres

import (
	"context"

	"eventflow/internal/order/domain/entity"
	"eventflow/internal/order/domain/repository"
	"eventflow/pkg/apperrors"
	"eventflow/pkg/database"

	"github.com/jackc/pgx/v5/pgxpool"
)

type OrderRepository struct {
	pool *pgxpool.Pool
}

func NewOrderRepository(provider database.Database) (*OrderRepository, error) {
	pool, err := database.AsPostgresPool(provider)
	if err != nil {
		return nil, err
	}

	return &OrderRepository{pool: pool}, nil
}

// Save guarda el pedido y sus artículos en una misma transacción.
func (r *OrderRepository) Save(ctx context.Context, order entity.Order) (*entity.Order, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, persistenceError(err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO orders (
			id, customer_id, total_minor, status, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6)
	`,
		order.ID,
		order.CustomerID,
		order.TotalMinor,
		order.Status,
		order.CreatedAt,
		order.UpdatedAt,
	)
	if err != nil {
		return nil, persistenceError(err)
	}

	for _, item := range order.Items {
		_, err = tx.Exec(ctx, `
			INSERT INTO order_items (
				order_id, product_id, quantity, unit_price_minor
			)
			VALUES ($1, $2, $3, $4)
		`,
			order.ID,
			item.ProductID,
			item.Quantity,
			item.UnitPriceMinor,
		)
		if err != nil {
			return nil, persistenceError(err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, persistenceError(err)
	}

	saved := order
	return &saved, nil
}

func persistenceError(cause error) error {
	return apperrors.Wrap(
		cause,
		apperrors.InternalError,
		"ORDER_PERSISTENCE",
		"No se pudo completar la operacion del pedido",
	)
}

// Comprueba al compilar que implementa la interfaz.
var _ repository.OrderRepository = (*OrderRepository)(nil)
