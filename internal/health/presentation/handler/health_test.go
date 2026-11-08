package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"eventflow/internal/health/application/dto"
	"eventflow/internal/health/application/usecase"

	"github.com/gin-gonic/gin"
)

type healthRepositoryStub struct{ err error }

func (s healthRepositoryStub) Ping(context.Context) error { return s.err }

func TestHealthHTTP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tt := range []struct {
		name, database string
		err            error
	}{
		{"connected", "connected", nil},
		{"disconnected", "disconnected", errors.New("secret database password")},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.GET("/healthz", NewHealthHandler(usecase.NewHealthUseCase(healthRepositoryStub{err: tt.err})).Health)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/healthz", nil))
			if w.Code != http.StatusOK {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
			var body dto.HealthResponse
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body.Database != tt.database || body.Service == "" || body.Timestamp.IsZero() {
				t.Errorf("body=%+v", body)
			}
			if strings.Contains(w.Body.String(), "secret database password") {
				t.Error("respuesta expone error interno")
			}
		})
	}
}
