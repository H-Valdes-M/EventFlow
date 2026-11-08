package consumer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"eventflow/internal/order/domain/entity"
	"eventflow/pkg/identifier"
	"eventflow/pkg/pubsub"
)

const Name = "order-created"

// Store persiste el resultado antes de confirmar la entrega.
type Store interface {
	Process(context.Context, entity.OrderCreatedEvent, int) error
	Fail(context.Context, entity.OrderCreatedEvent, int) error
	Reject(context.Context, string) error
}

type Handler struct{ store Store }

func New(store Store) *Handler { return &Handler{store: store} }

func (h *Handler) Handle(ctx context.Context, delivery pubsub.Delivery) (pubsub.Disposition, error) {
	var event entity.OrderCreatedEvent
	err := json.Unmarshal(delivery.Payload, &event)
	if err != nil || delivery.Type != entity.OrderCreatedEventType || event.EventType != entity.OrderCreatedEventType ||
		!identifier.IsCanonicalUUID(event.EventID) || !identifier.IsCanonicalUUID(event.OrderID) || event.OccurredAt.IsZero() || delivery.EventID != event.EventID {
		digest := sha256.Sum256(delivery.Payload)
		if err := h.store.Reject(ctx, hex.EncodeToString(digest[:])); err != nil {
			return pubsub.KeepUnconfirmed, err
		}
		return pubsub.NackDrop, nil
	}

	for attempt := 1; attempt <= 4; attempt++ {
		if err := h.store.Process(ctx, event, attempt); err == nil {
			return pubsub.Ack, nil
		} else if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return pubsub.KeepUnconfirmed, err
		}
		if attempt < 4 {
			timer := time.NewTimer(time.Second * time.Duration(1<<(attempt-1)))
			select {
			case <-ctx.Done():
				timer.Stop()
				return pubsub.KeepUnconfirmed, ctx.Err()
			case <-timer.C:
			}
		}
	}
	if err := h.store.Fail(ctx, event, 4); err != nil {
		return pubsub.KeepUnconfirmed, err
	}
	return pubsub.Ack, nil
}
