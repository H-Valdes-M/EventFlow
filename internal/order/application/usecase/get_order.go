package usecase

import (
	"context"
	"errors"

	"eventflow/internal/order/application/dto"
	"eventflow/internal/order/domain/entity"
	"eventflow/internal/order/domain/repository"
	"eventflow/pkg/apperrors"
)

// GetOrderUseCase coordina la lectura de un pedido sin modificarlo.
type GetOrderUseCase struct {
	repository repository.OrderRepository
}

func NewGetOrderUseCase(repository repository.OrderRepository) *GetOrderUseCase {
	return &GetOrderUseCase{repository: repository}
}

func (uc *GetOrderUseCase) Execute(ctx context.Context, id string) (dto.GetOrderOutput, error) {
	order, err := uc.repository.FindByID(ctx, id)
	if errors.Is(err, repository.ErrOrderNotFound) {
		return dto.GetOrderOutput{}, apperrors.New(apperrors.NotFound, "ORDER_NOT_FOUND", "El pedido no existe")
	}
	if err != nil {
		return dto.GetOrderOutput{}, err
	}

	items := make([]dto.GetOrderItemOutput, len(order.Items))
	for index, item := range order.Items {
		items[index] = dto.GetOrderItemOutput{
			ProductID:      item.ProductID,
			Quantity:       item.Quantity,
			UnitPriceMinor: item.UnitPriceMinor,
		}
	}
	return dto.GetOrderOutput{
		ID:         order.ID,
		CustomerID: order.CustomerID,
		Items:      items,
		TotalMinor: order.TotalMinor,
		Status:     string(order.Status),
		CreatedAt:  order.CreatedAt.UTC(),
		UpdatedAt:  order.UpdatedAt.UTC(),
	}, nil
}

// GetAllOrdersUseCase coordina la lectura de todos los pedidos.
type GetAllOrdersUseCase struct {
	repository repository.OrderRepository
}

func NewGetAllOrdersUseCase(repository repository.OrderRepository) *GetAllOrdersUseCase {
	return &GetAllOrdersUseCase{repository: repository}
}

// Execute obtiene todos los pedidos y los mapea al DTO de salida.
func (uc *GetAllOrdersUseCase) Execute(ctx context.Context) ([]dto.GetOrderOutput, error) {
	orders, err := uc.repository.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	outputs := make([]dto.GetOrderOutput, len(orders))
	for index, order := range orders {
		outputs[index] = mapOrderOutput(order)
	}
	return outputs, nil
}

func mapOrderOutput(order *entity.Order) dto.GetOrderOutput {
	items := make([]dto.GetOrderItemOutput, len(order.Items))
	for index, item := range order.Items {
		items[index] = dto.GetOrderItemOutput{
			ProductID:      item.ProductID,
			Quantity:       item.Quantity,
			UnitPriceMinor: item.UnitPriceMinor,
		}
	}
	return dto.GetOrderOutput{
		ID:         order.ID,
		CustomerID: order.CustomerID,
		Items:      items,
		TotalMinor: order.TotalMinor,
		Status:     string(order.Status),
		CreatedAt:  order.CreatedAt.UTC(),
		UpdatedAt:  order.UpdatedAt.UTC(),
	}
}
