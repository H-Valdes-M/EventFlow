package postgres

import (
	"context"
	"errors"
	"time"

	"eventflow/pkg/database"
	"eventflow/pkg/pubsub"

	"gorm.io/gorm"
)

// OutboxRelay publica los eventos persistidos en PostgreSQL en el broker.
type OutboxRelay struct {
	db        *gorm.DB
	publisher pubsub.Publisher
}

func NewOutboxRelay(provider database.Database, publisher pubsub.Publisher) (*OutboxRelay, error) {
	db, err := database.AsGORM(provider)
	if err != nil {
		return nil, err
	}
	if publisher == nil {
		return nil, errors.New("publisher de outbox no configurado")
	}
	return &OutboxRelay{db: db, publisher: publisher}, nil
}

// Run publica eventos pendientes hasta que el contexto se cancele.
func (r *OutboxRelay) Run(ctx context.Context, pollInterval time.Duration, batchSize int, publishTimeout time.Duration) error {
	if r == nil || r.db == nil || r.publisher == nil {
		return errors.New("relay de outbox no inicializado")
	}
	if pollInterval <= 0 || batchSize < 1 || publishTimeout <= 0 {
		return errors.New("configuracion invalida del relay de outbox")
	}

	if err := r.publishBatch(ctx, batchSize, publishTimeout); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := r.publishBatch(ctx, batchSize, publishTimeout); err != nil && !errors.Is(err, context.Canceled) {
				return err
			}
		}
	}
}

func (r *OutboxRelay) publishBatch(ctx context.Context, batchSize int, publishTimeout time.Duration) error {
	now := time.Now().UTC()
	var events []outboxEventModel
	if err := r.db.WithContext(ctx).
		Where("published_at IS NULL AND available_at <= ?", now).
		Order("available_at ASC, created_at ASC").
		Limit(batchSize).
		Find(&events).Error; err != nil {
		return err
	}

	for _, event := range events {
		attempt := event.PublishAttempts + 1
		if err := r.db.WithContext(ctx).Model(&outboxEventModel{}).
			Where("message_id = ? AND published_at IS NULL", event.MessageID).
			Updates(map[string]any{"publish_attempts": attempt}).Error; err != nil {
			return err
		}

		publishCtx, cancel := context.WithTimeout(ctx, publishTimeout)
		err := r.publisher.PublishConfirmed(publishCtx, pubsub.Message{
			MessageID: event.MessageID,
			EventID:   event.EventID,
			Type:      event.EventType,
			Attempt:   attempt,
			Payload:   event.Payload,
		})
		cancel()
		if err != nil {
			nextAttempt := time.Now().UTC().Add(backoff(attempt))
			if updateErr := r.db.WithContext(ctx).Model(&outboxEventModel{}).
				Where("message_id = ?", event.MessageID).
				Updates(map[string]any{
					"available_at":       nextAttempt,
					"last_publish_error": "no se pudo publicar el evento",
				}).Error; updateErr != nil {
				return updateErr
			}
			continue
		}

		publishedAt := time.Now().UTC()
		if err := r.db.WithContext(ctx).Model(&outboxEventModel{}).
			Where("message_id = ?", event.MessageID).
			Updates(map[string]any{"published_at": publishedAt, "last_publish_error": nil}).Error; err != nil {
			return err
		}
	}
	return nil
}

func backoff(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	if attempt > 6 {
		attempt = 6
	}
	return time.Duration(1<<(attempt-1)) * time.Second
}
