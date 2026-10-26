package usecase

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"eventflow/internal/order/application/dto"
	"eventflow/internal/order/domain/entity"
)

type createOrderRepositoryMock struct {
	savedOrder entity.Order
	saveErr    error
}

func (m *createOrderRepositoryMock) Save(_ context.Context, order entity.Order) error {
	m.savedOrder = order
	return m.saveErr
}

func (m *createOrderRepositoryMock) FindByID(context.Context, string) (*entity.Order, error) {
	return nil, errors.New("FindByID no esperado en esta prueba")
}

func (m *createOrderRepositoryMock) FindAll(context.Context) ([]*entity.Order, error) {
	return nil, errors.New("FindAll no esperado en esta prueba")
}

func TestCreateOrderUseCaseExecuteCreatesAndSavesPendingOrder(t *testing.T) {
	repo := &createOrderRepositoryMock{}
	uc := NewCreateOrderUseCase(repo)

	output, err := uc.Execute(context.Background(), dto.CreateOrderInput{
		CustomerID: " customer-123 ",
		Items: []dto.CreateOrderItemInput{
			{ProductID: "product-002", Quantity: 1, UnitPriceMinor: 500},
			{ProductID: "product-001", Quantity: 2, UnitPriceMinor: 1500},
		},
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`).MatchString(output.ID) {
		t.Fatalf("ID = %q, se esperaba UUID v4", output.ID)
	}
	if output.Status != string(entity.OrderPending) {
		t.Fatalf("Status = %q, se esperaba %q", output.Status, entity.OrderPending)
	}
	if output.TotalMinor != 3500 {
		t.Fatalf("TotalMinor = %d, se esperaba 3500", output.TotalMinor)
	}
	if repo.savedOrder.ID != output.ID {
		t.Fatalf("el pedido guardado tiene ID %q, se esperaba %q", repo.savedOrder.ID, output.ID)
	}
	if repo.savedOrder.CustomerID != "customer-123" {
		t.Fatalf("CustomerID = %q, se esperaba normalizado", repo.savedOrder.CustomerID)
	}
	if repo.savedOrder.Status != entity.OrderPending {
		t.Fatalf("estado guardado = %q, se esperaba pending", repo.savedOrder.Status)
	}
	if repo.savedOrder.CreatedAt.IsZero() || repo.savedOrder.UpdatedAt.IsZero() {
		t.Fatal("las fechas del pedido guardado no deben estar vacías")
	}
	if len(repo.savedOrder.Items) != 2 || repo.savedOrder.Items[0].ProductID != "product-001" {
		t.Fatal("los artículos deben guardarse normalizados y ordenados")
	}
}

func TestCreateOrderUseCaseExecuteReturnsValidationErrorWithoutSaving(t *testing.T) {
	repo := &createOrderRepositoryMock{}
	uc := NewCreateOrderUseCase(repo)

	_, err := uc.Execute(context.Background(), dto.CreateOrderInput{
		CustomerID: "customer-123",
		Items:      nil,
	})

	if err == nil {
		t.Fatal("Execute() debía devolver error de validación")
	}
	if repo.savedOrder.ID != "" {
		t.Fatal("no se debe guardar un pedido inválido")
	}
}

func TestCreateOrderUseCaseExecuteReturnsRepositoryError(t *testing.T) {
	wantErr := errors.New("base de datos no disponible")
	repo := &createOrderRepositoryMock{saveErr: wantErr}
	uc := NewCreateOrderUseCase(repo)

	_, err := uc.Execute(context.Background(), dto.CreateOrderInput{
		CustomerID: "customer-123",
		Items: []dto.CreateOrderItemInput{
			{ProductID: "product-001", Quantity: 1, UnitPriceMinor: 100},
		},
	})

	if err == nil || !errors.Is(err, wantErr) {
		t.Fatalf("Execute() error = %v, se esperaba una envoltura de %v", err, wantErr)
	}
}
