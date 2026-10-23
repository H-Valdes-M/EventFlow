package postgres

import (
	"context"
	"errors"
	"time"

	"eventflow/internal/order/domain/entity"
	"eventflow/internal/order/domain/repository"
	"eventflow/pkg/apperrors"
	"eventflow/pkg/database"

	"gorm.io/gorm"
)

type OrderRepository struct {
	db *gorm.DB
}

func NewOrderRepository(provider database.Database) (*OrderRepository, error) {
	db, err := database.AsGORM(provider)
	if err != nil {
		return nil, err
	}

	return &OrderRepository{db: db}, nil
}

// Save guarda el pedido y sus artículos en una misma transacción.
func (r *OrderRepository) Save(ctx context.Context, order entity.Order) (*entity.Order, error) {
	model := orderModelFromEntity(order)
	if err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Omit("Items").Create(&model).Error; err != nil {
			return err
		}
		if len(model.Items) == 0 {
			return nil
		}
		return tx.Create(&model.Items).Error
	}); err != nil {
		return nil, persistenceError(err)
	}

	saved := order
	return &saved, nil
}

// FindByID reconstruye un pedido junto con sus artículos sin modificar la
// base de datos.
func (r *OrderRepository) FindByID(ctx context.Context, id string) (*entity.Order, error) {
	var model orderModel
	err := r.db.WithContext(ctx).Preload("Items", func(db *gorm.DB) *gorm.DB {
		return db.Order("product_id")
	}).First(&model, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, repository.ErrOrderNotFound
	}
	if err != nil {
		return nil, persistenceError(err)
	}
	return model.toEntity(), nil
}

type orderModel struct {
	ID         string             `gorm:"column:id;primaryKey"`
	CustomerID string             `gorm:"column:customer_id"`
	TotalMinor int64              `gorm:"column:total_minor"`
	Status     entity.OrderStatus `gorm:"column:status"`
	CreatedAt  time.Time          `gorm:"column:created_at"`
	UpdatedAt  time.Time          `gorm:"column:updated_at"`
	Items      []orderItemModel   `gorm:"foreignKey:OrderID;references:ID"`
}

func (orderModel) TableName() string { return "orders" }

type orderItemModel struct {
	OrderID        string `gorm:"column:order_id;primaryKey"`
	ProductID      string `gorm:"column:product_id;primaryKey"`
	Quantity       int64  `gorm:"column:quantity"`
	UnitPriceMinor int64  `gorm:"column:unit_price_minor"`
}

func (orderItemModel) TableName() string { return "order_items" }

func orderModelFromEntity(order entity.Order) orderModel {
	items := make([]orderItemModel, len(order.Items))
	for index, item := range order.Items {
		items[index] = orderItemModel{OrderID: order.ID, ProductID: item.ProductID, Quantity: item.Quantity, UnitPriceMinor: item.UnitPriceMinor}
	}
	return orderModel{ID: order.ID, CustomerID: order.CustomerID, TotalMinor: order.TotalMinor, Status: order.Status, CreatedAt: order.CreatedAt, UpdatedAt: order.UpdatedAt, Items: items}
}

func (model orderModel) toEntity() *entity.Order {
	items := make([]entity.OrderItem, len(model.Items))
	for index, item := range model.Items {
		items[index] = entity.OrderItem{ProductID: item.ProductID, Quantity: item.Quantity, UnitPriceMinor: item.UnitPriceMinor}
	}
	return &entity.Order{ID: model.ID, CustomerID: model.CustomerID, TotalMinor: model.TotalMinor, Status: model.Status, CreatedAt: model.CreatedAt, UpdatedAt: model.UpdatedAt, Items: items}
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
