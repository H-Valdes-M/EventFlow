package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"eventflow/internal/order/domain/entity"
	"eventflow/internal/order/domain/repository"
	"eventflow/pkg/apperrors"
	"eventflow/pkg/database"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
func (r *OrderRepository) Save(ctx context.Context, order entity.Order) error {
	return r.save(ctx, order, nil)
}

// SaveWithEvent guarda el pedido, sus artículos y el evento outbox en una
// única transacción.
func (r *OrderRepository) SaveWithEvent(ctx context.Context, order entity.Order, event entity.OrderCreatedEvent) error {
	return r.save(ctx, order, &event)
}

func (r *OrderRepository) save(ctx context.Context, order entity.Order, event *entity.OrderCreatedEvent) error {
	if err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return saveOrder(tx, order, event)
	}); err != nil {
		return persistenceError(err)
	}
	return nil
}

// SaveIdempotent reserva la clave y persiste pedido y evento de forma atomica.
func (r *OrderRepository) SaveIdempotent(ctx context.Context, order entity.Order, event entity.OrderCreatedEvent, keyHash, requestHash string) (entity.Order, error) {
	resultOrder := order
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		entry := idempotencyModel{KeyHash: keyHash, RequestHash: requestHash, OrderID: order.ID, ResponseStatus: string(order.Status), ResponseTotalMinor: order.TotalMinor}
		result := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "key_hash"}}, DoNothing: true}).Create(&entry)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			if err := tx.First(&entry, "key_hash = ?", keyHash).Error; err != nil {
				return err
			}
			if entry.RequestHash != requestHash {
				return repository.ErrIdempotencyConflict
			}
			resultOrder = entity.Order{ID: entry.OrderID, Status: entity.OrderStatus(entry.ResponseStatus), TotalMinor: entry.ResponseTotalMinor}
			return nil
		}
		return saveOrder(tx, order, &event)
	})
	if errors.Is(err, repository.ErrIdempotencyConflict) {
		return entity.Order{}, err
	}
	if err != nil {
		return entity.Order{}, persistenceError(err)
	}
	return resultOrder, nil
}

type idempotencyModel struct {
	KeyHash            string `gorm:"column:key_hash;primaryKey"`
	RequestHash        string `gorm:"column:request_hash"`
	OrderID            string `gorm:"column:order_id"`
	ResponseStatus     string `gorm:"column:response_status"`
	ResponseTotalMinor int64  `gorm:"column:response_total_minor"`
}

func (idempotencyModel) TableName() string { return "order_idempotency" }

func saveOrder(tx *gorm.DB, order entity.Order, event *entity.OrderCreatedEvent) error {
	model := orderModelFromEntity(order)
	if err := tx.Omit("Items").Create(&model).Error; err != nil {
		return err
	}
	if len(model.Items) > 0 {
		if err := tx.Create(&model.Items).Error; err != nil {
			return err
		}
	}
	if event == nil {
		return nil
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return tx.Create(&outboxEventModel{
		MessageID:   event.EventID,
		EventID:     event.EventID,
		EventType:   event.EventType,
		OrderID:     event.OrderID,
		Payload:     payload,
		AvailableAt: event.OccurredAt,
		CreatedAt:   event.OccurredAt,
	}).Error
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

// FindAll reconstruye todos los pedidos junto con sus artículos, ordenados del
// más reciente al más antiguo.
func (r *OrderRepository) FindAll(ctx context.Context) ([]*entity.Order, error) {
	var models []orderModel
	err := r.db.WithContext(ctx).
		Preload("Items", func(db *gorm.DB) *gorm.DB {
			return db.Order("product_id")
		}).
		Order("created_at DESC").
		Find(&models).Error
	if err != nil {
		return nil, persistenceError(err)
	}

	orders := make([]*entity.Order, len(models))
	for index, model := range models {
		orders[index] = model.toEntity()
	}
	return orders, nil
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

type outboxEventModel struct {
	MessageID        string     `gorm:"column:message_id;primaryKey"`
	EventID          string     `gorm:"column:event_id;uniqueIndex"`
	EventType        string     `gorm:"column:event_type"`
	OrderID          string     `gorm:"column:order_id"`
	Payload          []byte     `gorm:"column:payload;type:jsonb"`
	AvailableAt      time.Time  `gorm:"column:available_at"`
	PublishedAt      *time.Time `gorm:"column:published_at"`
	PublishAttempts  int        `gorm:"column:publish_attempts"`
	LastPublishError string     `gorm:"column:last_publish_error"`
	CreatedAt        time.Time  `gorm:"column:created_at"`
}

func (outboxEventModel) TableName() string { return "outbox_events" }

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
