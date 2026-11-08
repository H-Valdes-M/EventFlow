package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"eventflow/pkg/apperrors"

	"github.com/gin-gonic/gin"
)

func TestErrorHandlerMiddlewareMapsUnexpectedPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(ErrorHandlerMiddleware(nil))
	router.GET("/panic", func(*gin.Context) {
		panic("unexpected failure")
	})

	request := httptest.NewRequest(http.MethodGet, "/panic", nil)
	request.Header.Set("X-Request-ID", "request-123")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, response.Code)
	}
	if response.Header().Get("Content-Type") != "application/json; charset=utf-8" {
		t.Fatalf("expected JSON content type, got %q", response.Header().Get("Content-Type"))
	}
	if response.Body.String() == "" {
		t.Fatal("expected error response body")
	}
	if !strings.Contains(response.Body.String(), `"request_id":"request-123"`) {
		t.Fatalf("expected request ID in response, got %s", response.Body.String())
	}
}

func TestErrorHandlerMiddlewareMapsTypedPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(ErrorHandlerMiddleware(nil))
	router.GET("/panic", func(*gin.Context) {
		panic(apperrors.New(apperrors.NotFound, "ORDER_NOT_FOUND", "Pedido no encontrado"))
	})

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/panic", nil))

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, response.Code)
	}
	if !strings.Contains(response.Body.String(), `"code":"ORDER_NOT_FOUND"`) {
		t.Fatalf("expected mapped error code, got %s", response.Body.String())
	}
}
