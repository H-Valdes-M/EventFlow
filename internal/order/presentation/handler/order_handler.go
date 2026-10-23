package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"regexp"
	"time"

	"eventflow/internal/order/application/dto"
	"eventflow/pkg/apperrors"
	"eventflow/pkg/httputil"

	"github.com/gin-gonic/gin"
)

const maxRequestBodyBytes = 1 << 20

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)

// CreateOrderExecutor es el contrato que necesita el handler para crear un
// pedido. Mantenerlo como interfaz facilita probar HTTP sin PostgreSQL.
type CreateOrderExecutor interface {
	Execute(context.Context, dto.CreateOrderInput) (dto.CreateOrderOutput, error)
}

// GetOrderExecutor es el contrato que necesita el handler para consultar un pedido.
type GetOrderExecutor interface {
	Execute(context.Context, string) (dto.GetOrderOutput, error)
}

// OrderHandler adapta las peticiones HTTP del módulo de pedidos.
type OrderHandler struct {
	create CreateOrderExecutor
	get    GetOrderExecutor
}

func NewOrderHandler(create CreateOrderExecutor, get GetOrderExecutor) *OrderHandler {
	return &OrderHandler{create: create, get: get}
}

// Create crea un pedido validado por el caso de uso.
//
// @Summary Crear pedido
// @Tags orders
// @Accept json
// @Produce json
// @Param request body createOrderRequest true "Pedido"
// @Success 201 {object} createOrderResponse
// @Failure 400,413,415,422,500 {object} httputil.ErrorResponse
// @Router /orders [post]
func (h *OrderHandler) Create(c *gin.Context) {
	input, err := decodeCreateRequest(c)
	if err != nil {
		writeError(c, err)
		return
	}

	output, err := h.create.Execute(c.Request.Context(), input)
	if err != nil {
		writeError(c, err)
		return
	}

	c.Set("order_id", output.ID)
	c.Header("Location", "/orders/"+output.ID)
	c.JSON(http.StatusCreated, createOrderResponse{
		ID:         output.ID,
		Status:     output.Status,
		TotalMinor: output.TotalMinor,
	})
}

// GetByID consulta un pedido y sus artículos.
//
// @Summary Consultar pedido
// @Tags orders
// @Produce json
// @Param id path string true "UUID del pedido"
// @Success 200 {object} getOrderResponse
// @Failure 400,404,500 {object} httputil.ErrorResponse
// @Router /orders/{id} [get]
func (h *OrderHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	if !uuidPattern.MatchString(id) {
		writeError(c, apperrors.New(apperrors.BadRequest, "INVALID_ORDER_ID", "El identificador del pedido no es valido"))
		return
	}

	output, err := h.get.Execute(c.Request.Context(), id)
	if err != nil {
		writeError(c, err)
		return
	}

	c.Set("order_id", output.ID)
	items := make([]orderItemResponse, len(output.Items))
	for index, item := range output.Items {
		items[index] = orderItemResponse{
			ProductID:      item.ProductID,
			Quantity:       item.Quantity,
			UnitPriceMinor: item.UnitPriceMinor,
		}
	}
	c.JSON(http.StatusOK, getOrderResponse{
		ID:         output.ID,
		CustomerID: output.CustomerID,
		Items:      items,
		TotalMinor: output.TotalMinor,
		Status:     output.Status,
		CreatedAt:  output.CreatedAt,
		UpdatedAt:  output.UpdatedAt,
	})
}

type createOrderRequest struct {
	CustomerID string             `json:"customer_id"`
	Items      []orderItemRequest `json:"items"`
}

type orderItemRequest struct {
	ProductID      string `json:"product_id"`
	Quantity       int64  `json:"quantity"`
	UnitPriceMinor int64  `json:"unit_price_minor"`
}

type createOrderResponse struct {
	ID         string `json:"id"`
	Status     string `json:"status"`
	TotalMinor int64  `json:"total_minor"`
}

type getOrderResponse struct {
	ID         string              `json:"id"`
	CustomerID string              `json:"customer_id"`
	Items      []orderItemResponse `json:"items"`
	TotalMinor int64               `json:"total_minor"`
	Status     string              `json:"status"`
	CreatedAt  time.Time           `json:"created_at"`
	UpdatedAt  time.Time           `json:"updated_at"`
}

type orderItemResponse struct {
	ProductID      string `json:"product_id"`
	Quantity       int64  `json:"quantity"`
	UnitPriceMinor int64  `json:"unit_price_minor"`
}

func decodeCreateRequest(c *gin.Context) (dto.CreateOrderInput, error) {
	mediaType, _, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return dto.CreateOrderInput{}, apperrors.New(
			apperrors.UnsupportedMediaType,
			"UNSUPPORTED_MEDIA_TYPE",
			"El contenido debe ser application/json",
		)
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxRequestBodyBytes)
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()

	var request createOrderRequest
	if err := decoder.Decode(&request); err != nil {
		return dto.CreateOrderInput{}, decodeError(err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return dto.CreateOrderInput{}, apperrors.New(
			apperrors.BadRequest,
			"INVALID_JSON",
			"El cuerpo debe contener un solo objeto JSON",
		)
	}

	items := make([]dto.CreateOrderItemInput, len(request.Items))
	for index, item := range request.Items {
		items[index] = dto.CreateOrderItemInput{
			ProductID:      item.ProductID,
			Quantity:       item.Quantity,
			UnitPriceMinor: item.UnitPriceMinor,
		}
	}
	return dto.CreateOrderInput{CustomerID: request.CustomerID, Items: items}, nil
}

func decodeError(err error) error {
	var maxBytesError *http.MaxBytesError
	if errors.As(err, &maxBytesError) {
		return apperrors.New(
			apperrors.PayloadTooLarge,
			"PAYLOAD_TOO_LARGE",
			"El cuerpo supera el limite permitido",
		)
	}
	return apperrors.New(apperrors.BadRequest, "INVALID_JSON", "El cuerpo JSON no es valido")
}

func writeError(c *gin.Context, err error) {
	status, response := httputil.MapErrorToHTTP(err, c.GetHeader("X-Request-ID"), time.Now().UTC())
	c.AbortWithStatusJSON(status, response)
}
