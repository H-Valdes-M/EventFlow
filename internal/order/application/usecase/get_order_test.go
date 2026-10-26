package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"eventflow/internal/order/domain/entity"
	"eventflow/internal/order/domain/repository"
	"eventflow/pkg/apperrors"
)

type getOrderRepositoryMock struct {
	order  *entity.Order
	orders []*entity.Order
	err    error
	id     string
}

func (m *getOrderRepositoryMock) Save(context.Context, entity.Order) error {
	return nil
}

func (m *getOrderRepositoryMock) FindByID(_ context.Context, id string) (*entity.Order, error) {
	m.id = id
	return m.order, m.err
}

func (m *getOrderRepositoryMock) FindAll(context.Context) ([]*entity.Order, error) {
	return m.orders, m.err
}

func TestGetOrderUseCaseExecuteMapsOrder(t *testing.T) {
	createdAt := time.Date(2026, time.January, 10, 12, 0, 0, 0, time.FixedZone("test", -3*60*60))
	updatedAt := createdAt.Add(time.Hour)
	repo := &getOrderRepositoryMock{order: &entity.Order{
		ID:         "b5b7d67a-bfc5-483a-a80d-7de80cab748e",
		CustomerID: "customer-123",
		Items: []entity.OrderItem{
			{ProductID: "product-001", Quantity: 2, UnitPriceMinor: 1500},
		},
		TotalMinor: 3000,
		Status:     entity.OrderPending,
		CreatedAt:  createdAt,
		UpdatedAt:  updatedAt,
	}}
	uc := NewGetOrderUseCase(repo)

	output, err := uc.Execute(context.Background(), repo.order.ID)

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if repo.id != repo.order.ID {
		t.Fatalf("FindByID recibió %q, se esperaba %q", repo.id, repo.order.ID)
	}
	if output.ID != repo.order.ID || output.CustomerID != repo.order.CustomerID {
		t.Fatal("el DTO no conserva los datos principales del pedido")
	}
	if output.Status != string(entity.OrderPending) || output.TotalMinor != 3000 {
		t.Fatalf("DTO status/total = %q/%d, valores incorrectos", output.Status, output.TotalMinor)
	}
	if len(output.Items) != 1 || output.Items[0].ProductID != "product-001" || output.Items[0].Quantity != 2 {
		t.Fatal("el DTO no mapea correctamente los artículos")
	}
	if !output.CreatedAt.Equal(createdAt.UTC()) || !output.UpdatedAt.Equal(updatedAt.UTC()) {
		t.Fatal("las fechas del DTO deben estar normalizadas a UTC")
	}
}

func TestGetOrderUseCaseExecuteMapsNotFound(t *testing.T) {
	repo := &getOrderRepositoryMock{err: repository.ErrOrderNotFound}
	uc := NewGetOrderUseCase(repo)

	_, err := uc.Execute(context.Background(), "order-inexistente")

	if err == nil || !errors.Is(err, apperrors.ErrNotFound) {
		t.Fatalf("Execute() error = %v, se esperaba un error not found", err)
	}
	var appErr *apperrors.Error
	if !errors.As(err, &appErr) || appErr.Code != "ORDER_NOT_FOUND" {
		t.Fatalf("error = %v, se esperaba código ORDER_NOT_FOUND", err)
	}
}

func TestGetOrderUseCaseExecutePropagatesRepositoryError(t *testing.T) {
	wantErr := errors.New("fallo de lectura")
	repo := &getOrderRepositoryMock{err: wantErr}
	uc := NewGetOrderUseCase(repo)

	_, err := uc.Execute(context.Background(), "order-1")

	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, se esperaba %v", err, wantErr)
	}
}

func TestGetAllOrdersUseCaseExecuteMapsOrders(t *testing.T) {
	repo := &getOrderRepositoryMock{orders: []*entity.Order{
		{
			ID:         "order-1",
			CustomerID: "customer-1",
			Status:     entity.OrderPending,
			Items:      []entity.OrderItem{{ProductID: "product-1", Quantity: 1, UnitPriceMinor: 100}},
			TotalMinor: 100,
		},
		{ID: "order-2", CustomerID: "customer-2", Status: entity.OrderPending, TotalMinor: 200},
	}}
	uc := NewGetAllOrdersUseCase(repo)

	outputs, err := uc.Execute(context.Background())

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(outputs) != 2 {
		t.Fatalf("cantidad de pedidos = %d, se esperaban 2", len(outputs))
	}
	if outputs[0].ID != "order-1" || len(outputs[0].Items) != 1 {
		t.Fatal("el primer pedido no fue mapeado correctamente")
	}
	if outputs[1].ID != "order-2" || outputs[1].TotalMinor != 200 {
		t.Fatal("el segundo pedido no fue mapeado correctamente")
	}
}
