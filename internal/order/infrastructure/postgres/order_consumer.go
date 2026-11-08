package postgres

import (
	"context"
	"errors"

	"eventflow/internal/order/application/consumer"
	"eventflow/internal/order/domain/entity"
	"eventflow/pkg/database"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	processingCompleted   = "completed"
	processingFailed      = "failed"
	notificationProcessed = "order.processed"
	processingFailure     = "No se pudo procesar el pedido"
)

type OrderConsumerStore struct{ db *gorm.DB }

func NewOrderConsumerStore(provider database.Database) (*OrderConsumerStore, error) {
	db, err := database.AsGORM(provider)
	if err != nil {
		return nil, err
	}
	return &OrderConsumerStore{db: db}, nil
}

func (s *OrderConsumerStore) Process(ctx context.Context, event entity.OrderCreatedEvent, attempt int) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		inserted, err := insertEventProcessing(tx, event, processingCompleted, attempt)
		if err != nil {
			return err
		}
		if !inserted {
			return nil
		}
		result := tx.Model(&orderModel{}).
			Where("id = ? AND status IN ?", event.OrderID, []string{"pending", "processed"}).
			UpdateColumns(map[string]any{
				"status":     "processed",
				"updated_at": gorm.Expr("CASE WHEN status = ? THEN NOW() ELSE updated_at END", "pending"),
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errors.New("pedido inexistente o con estado no procesable")
		}
		return tx.Table("notifications").Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "order_id"}, {Name: "notification_type"}}, DoNothing: true,
		}).Create(map[string]any{"order_id": event.OrderID, "notification_type": notificationProcessed}).Error
	})
}

func (s *OrderConsumerStore) Fail(ctx context.Context, event entity.OrderCreatedEvent, attempts int) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		inserted, err := insertEventProcessing(tx, event, processingFailed, attempts)
		if err != nil {
			return err
		}
		if !inserted {
			return nil
		}
		return tx.Table("failed_events").Clauses(clause.OnConflict{DoNothing: true}).Create(map[string]any{
			"consumer_name": consumer.Name, "event_id": event.EventID,
			"order_id": event.OrderID, "reason_sanitized": processingFailure,
		}).Error
	})
}

func insertEventProcessing(tx *gorm.DB, event entity.OrderCreatedEvent, status string, attempts int) (bool, error) {
	result := tx.Table("event_processing").Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "consumer_name"}, {Name: "event_id"}}, DoNothing: true,
	}).Create(map[string]any{
		"consumer_name": consumer.Name, "event_id": event.EventID, "order_id": event.OrderID,
		"status": status, "attempts": attempts,
	})
	return result.RowsAffected == 1, result.Error
}

func (s *OrderConsumerStore) Reject(ctx context.Context, digest string) error {
	return s.db.WithContext(ctx).Table("rejected_deliveries").Clauses(clause.OnConflict{DoNothing: true}).Create(map[string]any{
		"consumer_name": consumer.Name, "payload_digest": digest, "reason_sanitized": "Evento invalido",
	}).Error
}
