package usecase

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"eventflow/internal/order/application/dto"
	"eventflow/internal/order/domain/entity"
	"eventflow/internal/order/domain/repository"
	"eventflow/pkg/apperrors"
)

// CreateOrderUseCase coordina la validación de dominio y la persistencia.
type CreateOrderUseCase struct {
	repo repository.IdempotentOrderRepository
}

// crear un nuevo caso de uso para crear pedidos
func NewCreateOrderUseCase(repo repository.IdempotentOrderRepository) *CreateOrderUseCase {
	return &CreateOrderUseCase{repo: repo}
}

func (uc *CreateOrderUseCase) Execute(ctx context.Context, input dto.CreateOrderInput) (dto.CreateOrderOutput, error) {
	id, err := NewUUID()
	if err != nil {
		return dto.CreateOrderOutput{}, apperrors.Wrap(
			err,
			apperrors.InternalError,
			"ORDER_ID_GENERATION",
			"No se pudo crear el pedido",
		)
	}

	items := make([]entity.OrderItem, len(input.Items))
	for i, item := range input.Items {
		items[i] = entity.OrderItem{
			ProductID:      item.ProductID,
			Quantity:       item.Quantity,
			UnitPriceMinor: item.UnitPriceMinor,
		}
	}
	order, err := entity.NewOrder(id, input.CustomerID, items, time.Now())
	if err != nil {
		return dto.CreateOrderOutput{}, err
	}

	eventID, err := NewUUID()
	if err != nil {
		return dto.CreateOrderOutput{}, apperrors.Wrap(
			err,
			apperrors.InternalError,
			"ORDER_EVENT_ID_GENERATION",
			"No se pudo crear el evento del pedido",
		)
	}
	event := entity.OrderCreatedEvent{
		EventID:    eventID,
		EventType:  entity.OrderCreatedEventType,
		OccurredAt: order.CreatedAt,
		OrderID:    order.ID,
	}
	canonical, err := json.Marshal(struct {
		CustomerID string
		Items      []entity.OrderItem
	}{order.CustomerID, order.Items})
	if err != nil {
		return dto.CreateOrderOutput{}, fmt.Errorf("codificar pedido: %w", err)
	}
	keyHash := sha256.Sum256([]byte(input.IdempotencyKey))
	requestHash := sha256.Sum256(canonical)
	order, err = uc.repo.SaveIdempotent(ctx, order, event, hex.EncodeToString(keyHash[:]), hex.EncodeToString(requestHash[:]))
	if errors.Is(err, repository.ErrIdempotencyConflict) {
		return dto.CreateOrderOutput{}, apperrors.New(apperrors.Conflict, "IDEMPOTENCY_CONFLICT", "La clave ya se utilizo con otro pedido")
	}
	if err != nil {
		return dto.CreateOrderOutput{}, fmt.Errorf("guardar pedido: %w", err)
	}

	return dto.CreateOrderOutput{
		ID:         order.ID,
		Status:     string(order.Status),
		TotalMinor: order.TotalMinor,
	}, nil
}

// NewUUID genera un UUID v4 compatible con la columna orders.id.
func NewUUID() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", err
	}
	value[6] = value[6]&0x0f | 0x40
	value[8] = value[8]&0x3f | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", value[0:4], value[4:6], value[6:8], value[8:10], value[10:16]), nil
}
