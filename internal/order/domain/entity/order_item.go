package entity

import (
	"eventflow/pkg/apperrors"
	"strings"
	"unicode/utf8"
)

const (
	maxIdentifier   = 100
	maxItemQuantity = int64(1_000_000)
)

// OrderItem representa una linea de pedido. UnitPriceMinor equivale a pesos (CLP)
type OrderItem struct {
	ProductID      string
	Quantity       int64
	UnitPriceMinor int64
}

func ValidateItem(item OrderItem) (OrderItem, error) {
	item.ProductID = strings.TrimSpace(item.ProductID)
	if lenght := utf8.RuneCountInString(item.ProductID); lenght == 0 || lenght > maxIdentifier {
		return OrderItem{}, validationError("product_id", "El identificador del producto debe tener entre 1 y 100 caracteres")
	}
	if item.Quantity < 1 || item.Quantity > maxItemQuantity {
		return OrderItem{}, validationError("quantity", "La cantidad debe estar entre 1 y 1,000,000")
	}
	if item.UnitPriceMinor <= 0 {
		return OrderItem{}, validationError("unit_price_minor", "El precio unitario debe ser mayor que 0")
	}
	return item, nil
}

func validationError(field, message string) *apperrors.Error {
	err := apperrors.New(apperrors.ValidationError, "ORDER_ITEM_VALIDATION", message)
	err.Details = map[string]interface{}{
		"field": field,
	}
	return err
}
