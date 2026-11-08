package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"eventflow/internal/order/application/dto"
	"eventflow/internal/order/application/usecase"
	"eventflow/internal/order/domain/entity"
	"eventflow/internal/order/domain/repository"
	"eventflow/pkg/apperrors"
	"eventflow/pkg/httputil"

	"github.com/gin-gonic/gin"
)

const testOrderID = "b5b7d67a-bfc5-483a-a80d-7de80cab748e"
const validOrderBody = `{"customer_id":"customer-1","items":[{"product_id":"product-1","quantity":2,"unit_price_minor":500}]}`
const privateError = "secret password and SQL details"

type createStub struct {
	input  dto.CreateOrderInput
	output dto.CreateOrderOutput
	err    error
	calls  int
}

func (s *createStub) Execute(_ context.Context, input dto.CreateOrderInput) (dto.CreateOrderOutput, error) {
	s.calls++
	s.input = input
	return s.output, s.err
}

type getStub struct {
	id     string
	output dto.GetOrderOutput
	err    error
	calls  int
}

func (s *getStub) Execute(_ context.Context, id string) (dto.GetOrderOutput, error) {
	s.calls++
	s.id = id
	return s.output, s.err
}

type getAllStub struct {
	output []dto.GetOrderOutput
	err    error
}

func (s *getAllStub) Execute(context.Context) ([]dto.GetOrderOutput, error) { return s.output, s.err }

type orderRepositoryStub struct{ calls int }

func (s *orderRepositoryStub) SaveIdempotent(_ context.Context, order entity.Order, _ entity.OrderCreatedEvent, _, _ string) (entity.Order, error) {
	s.calls++
	return order, nil
}

func orderRouter(create CreateOrderExecutor, get GetOrderExecutor, all GetAllOrdersExecutor) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewOrderHandler(create, get, all)
	r.POST("/orders", h.Create)
	r.GET("/orders", h.GetAll)
	r.GET("/orders/:id", h.GetByID)
	return r
}

func request(router http.Handler, method, path, body, key string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func assertHTTPError(t *testing.T, w *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if w.Code != status {
		t.Fatalf("status=%d want=%d body=%s", w.Code, status, w.Body.String())
	}
	var result httputil.ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Error.Code != code {
		t.Errorf("code=%q want=%q", result.Error.Code, code)
	}
	if strings.Contains(w.Body.String(), privateError) {
		t.Error("la respuesta expone el error interno")
	}
}

func TestCreateOrderHTTP(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		stub := &createStub{output: dto.CreateOrderOutput{ID: testOrderID, Status: "pending", TotalMinor: 1000}}
		w := request(orderRouter(stub, nil, nil), http.MethodPost, "/orders", validOrderBody, "key-1")
		if w.Code != http.StatusCreated {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
		if got := w.Header().Get("Location"); got != "/orders/"+testOrderID {
			t.Errorf("Location=%q", got)
		}
		var body createOrderResponse
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body.ID != testOrderID || body.Status != "pending" || body.TotalMinor != 1000 {
			t.Errorf("body=%+v", body)
		}
		if stub.calls != 1 || stub.input.IdempotencyKey != "key-1" || stub.input.CustomerID != "customer-1" || len(stub.input.Items) != 1 || stub.input.Items[0].Quantity != 2 {
			t.Errorf("input=%+v calls=%d", stub.input, stub.calls)
		}
	})
	for _, tt := range []struct {
		name, body, key, code string
		status                int
		err                   error
		calls                 int
	}{
		{"missing key", validOrderBody, "", "INVALID_IDEMPOTENCY_KEY", 400, nil, 0},
		{"invalid key", validOrderBody, "bad key", "INVALID_IDEMPOTENCY_KEY", 400, nil, 0},
		{"malformed JSON", `{`, "key-1", "INVALID_JSON", 400, nil, 0},
		{"unknown field", `{"unknown":1}`, "key-1", "INVALID_JSON", 400, nil, 0},
		{"conflict", validOrderBody, "key-1", "IDEMPOTENCY_CONFLICT", 409, apperrors.New(apperrors.Conflict, "IDEMPOTENCY_CONFLICT", "conflicto"), 1},
		{"dependency error", validOrderBody, "key-1", "INTERNAL_ERROR", 500, errors.New(privateError), 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			stub := &createStub{err: tt.err}
			w := request(orderRouter(stub, nil, nil), http.MethodPost, "/orders", tt.body, tt.key)
			assertHTTPError(t, w, tt.status, tt.code)
			if stub.calls != tt.calls {
				t.Errorf("calls=%d want=%d", stub.calls, tt.calls)
			}
		})
	}
}

func TestCreateOrderHTTPBusinessValidation(t *testing.T) {
	for _, body := range []string{
		`{"customer_id":"","items":[{"product_id":"p","quantity":1,"unit_price_minor":1}]}`,
		`{"customer_id":"c","items":[]}`,
		`{"customer_id":"c","items":[{"product_id":"p","quantity":0,"unit_price_minor":1}]}`,
		`{"customer_id":"c","items":[{"product_id":"p","quantity":1,"unit_price_minor":0}]}`,
	} {
		t.Run(body, func(t *testing.T) {
			repo := &orderRepositoryStub{}
			w := request(orderRouter(usecase.NewCreateOrderUseCase(repo), nil, nil), http.MethodPost, "/orders", body, "key-1")
			if w.Code != http.StatusUnprocessableEntity {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
			if repo.calls != 0 {
				t.Fatal("no se debe guardar un pedido invalido")
			}
		})
	}
}

func sampleOrder() dto.GetOrderOutput {
	return dto.GetOrderOutput{ID: testOrderID, CustomerID: "customer-1", Status: "processed", TotalMinor: 1000,
		CreatedAt: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC), UpdatedAt: time.Date(2026, 1, 2, 3, 5, 5, 0, time.UTC),
		Items: []dto.GetOrderItemOutput{{ProductID: "product-1", Quantity: 2, UnitPriceMinor: 500}}}
}

func TestGetOrderHTTP(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		stub := &getStub{output: sampleOrder()}
		w := request(orderRouter(nil, stub, nil), http.MethodGet, "/orders/"+testOrderID, "", "")
		if w.Code != 200 {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
		var body getOrderResponse
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if stub.id != testOrderID || stub.calls != 1 || body.ID != testOrderID || body.CustomerID != "customer-1" || body.Status != "processed" || body.TotalMinor != 1000 || len(body.Items) != 1 || body.Items[0].Quantity != 2 || !body.CreatedAt.Equal(sampleOrder().CreatedAt) {
			t.Errorf("body=%+v stub=%+v", body, stub)
		}
	})
	for _, tt := range []struct {
		name, id, code string
		status         int
		err            error
		calls          int
	}{
		{"invalid id", "invalid", "INVALID_ORDER_ID", 400, nil, 0},
		{"not found", testOrderID, "ORDER_NOT_FOUND", 404, apperrors.New(apperrors.NotFound, "ORDER_NOT_FOUND", "No existe"), 1},
		{"dependency error", testOrderID, "INTERNAL_ERROR", 500, errors.New(privateError), 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			stub := &getStub{err: tt.err}
			w := request(orderRouter(nil, stub, nil), http.MethodGet, "/orders/"+tt.id, "", "")
			assertHTTPError(t, w, tt.status, tt.code)
			if stub.calls != tt.calls {
				t.Errorf("calls=%d want=%d", stub.calls, tt.calls)
			}
		})
	}
}

func TestGetAllOrdersHTTP(t *testing.T) {
	for _, tt := range []struct {
		name   string
		orders []dto.GetOrderOutput
		err    error
		count  int
		status int
	}{
		{"orders", []dto.GetOrderOutput{sampleOrder()}, nil, 1, 200},
		{"empty", nil, nil, 0, 200},
		{"dependency error", nil, errors.New(privateError), 0, 500},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := request(orderRouter(nil, nil, &getAllStub{output: tt.orders, err: tt.err}), http.MethodGet, "/orders", "", "")
			if tt.err != nil {
				assertHTTPError(t, w, tt.status, "INTERNAL_ERROR")
				return
			}
			if w.Code != tt.status {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
			var body []getOrderResponse
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body == nil || len(body) != tt.count {
				t.Fatalf("body=%s", w.Body.String())
			}
			if tt.count > 0 && (body[0].ID != testOrderID || len(body[0].Items) != 1 || body[0].Items[0].ProductID != "product-1") {
				t.Errorf("body=%+v", body)
			}
		})
	}
}

var _ repository.IdempotentOrderRepository = (*orderRepositoryStub)(nil)
