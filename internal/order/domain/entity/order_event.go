package entity

import "time"

const OrderCreatedEventType = "order.created"

// OrderCreatedEvent representa el evento emitido después de crear un pedido.
type OrderCreatedEvent struct {
	EventID    string    `json:"event_id"`
	EventType  string    `json:"event_type"`
	OccurredAt time.Time `json:"occurred_at"`
	OrderID    string    `json:"order_id"`
}
