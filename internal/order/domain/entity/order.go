package entity

import (
	"eventflow/pkg/apperrors"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

const maxOrderItems = 100 // máximo de items que puede tener un pedido.

type OrderStatus string // representa los estados de negocio de un pedido.

const OrderPending OrderStatus = "pending"

type Order struct {
	ID         string
	CustomerID string
	Items      []OrderItem
	TotalMinor int64
	Status     OrderStatus
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func NewOrder(id, customerID string, items []OrderItem, now time.Time) (Order, error) {
	customerID = strings.TrimSpace(customerID)
	if length := utf8.RuneCountInString(customerID); length == 0 || length > maxIdentifier {
		return Order{}, orderValidationError("customer_id", "El identificador del cliente debe tener entre 1 y 100 caracteres")
	}
	if len(items) == 0 || len(items) > maxOrderItems {
		return Order{}, orderValidationError("items", "El pedido debe tener entre 1 y 100 items")
	}

	normalizedItems := make([]OrderItem, 0, len(items))
	productIDs := make(map[string]struct{})
	for _, item := range items {
		validatedItem, err := ValidateItem(item)
		if err != nil {
			return Order{}, err
		}
		if _, exists := productIDs[validatedItem.ProductID]; exists {
			return Order{}, orderValidationError("items", "No se permiten productos duplicados en el pedido")
		}
		productIDs[validatedItem.ProductID] = struct{}{}
		normalizedItems = append(normalizedItems, validatedItem)
	}
	sort.Slice(normalizedItems, func(i, j int) bool {
		return normalizedItems[i].ProductID < normalizedItems[j].ProductID
	})

	total, err := CalculateTotal(normalizedItems)
	if err != nil {
		return Order{}, err
	}
	now = now.UTC()

	return Order{
		ID:         id,
		CustomerID: customerID,
		Items:      normalizedItems,
		TotalMinor: total,
		Status:     OrderPending,
		CreatedAt:  now,
		UpdatedAt:  now,
	}, nil
}

func orderValidationError(field, message string) *apperrors.Error {
	err := apperrors.New(apperrors.ValidationError, "ORDER_VALIDATION", message)
	err.Details = map[string]any{"field": field}
	return err
}
