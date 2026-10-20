package usecase

import (
	"context"
	"testing"
)

type mockHealthRepository struct{}

func (m *mockHealthRepository) Ping(ctx context.Context) error {
	return nil
}

func TestHealthUseCase(t *testing.T) {
	mockRepo := &mockHealthRepository{}
	// Patrón
	// 1) Arrancamos creando el use case
	uc := NewHealthUseCase(mockRepo)

	// 2) Ejecutamos el metodo
	res, err := uc.Execute(context.Background())

	// 3) Validamos resultados
	if err != nil {
		t.Fatalf("Execute() retuned unexpected error: %v", err)
	}

	if res == nil {
		t.Fatalf("Exceute retuned nil response")
	}

	if res.Service != "microservice" {
		t.Errorf("Execute().Service = %q, want %q", res.Service, "microservice")
	}

	if res.Timestamp.IsZero() {
		t.Errorf("Execute().Timestamp is zero, expected a valid timestamp")
	}

	if res.Database != "connected" {
		t.Errorf("Execute().Database = %q, want %q", res.Database, "connected")
	}
}

// Ejemplo cuando el use case tenga un repositorio:
// func TestHealthUseCase_Execute_WithDB(t *testing.T) {
//     mockRepo := new(MockHealthRepository)
//     mockRepo.On("Check").Return(nil)
//     uc := NewHealthUseCase(mockRepo)
//     res, err := uc.Execute()
//     assert.NoError(t, err)
//     assert.NotNil(t, res)
// }
