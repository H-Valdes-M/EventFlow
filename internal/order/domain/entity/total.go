package entity

import (
	"eventflow/pkg/apperrors"
	"math"
)

func CalculateTotal(items []OrderItem) (int64, error) {
	var total int64
	for _, item := range items {
		if item.Quantity < 1 || item.UnitPriceMinor <= 0 {
			return 0, totalError()
		}
		if item.Quantity > math.MaxInt64/item.UnitPriceMinor {
			return 0, totalError()
		}
		lineTotal := item.Quantity * item.UnitPriceMinor
		if total > math.MaxInt64-lineTotal {
			return 0, totalError()
		}
		total += lineTotal
	}
	return total, nil
}

func totalError() *apperrors.Error {
	err := apperrors.New(apperrors.ValidationError, "ORDER_TOTAL_VALIDATION", "El total del pedido no es correcto")
	err.Details = map[string]interface{}{
		"field": "items",
	}
	return err
}
