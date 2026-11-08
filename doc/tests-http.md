# Tests HTTP

Los tests de handlers usan `net/http/httptest` y dependencias simuladas. No
requieren un servidor HTTP, PostgreSQL ni RabbitMQ en ejecución.

```powershell
go test ./internal/health/presentation/handler ./internal/order/presentation/handler
```

Para ejecutar todos los tests del proyecto:

```powershell
go test ./...
```

Actualmente `/healthz` responde `200` tanto cuando la base de datos está
conectada como cuando no lo está; el campo `database` indica `connected` o
`disconnected`. Los tests reflejan ese comportamiento existente.
