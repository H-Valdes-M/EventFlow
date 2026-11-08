# Idempotencia de creación de pedidos

`POST /orders` requiere una cabecera `Idempotency-Key` única por operación.
Debe tener entre 1 y 128 caracteres ASCII: letras, números, `.`, `_`, `~` o `-`.
Una clave ausente, repetida en varias cabeceras o inválida produce `400`.

```http
POST /orders HTTP/1.1
Content-Type: application/json
Idempotency-Key: pedido-cliente-123

{"customer_id":"cliente-1","items":[{"product_id":"producto-1","quantity":1,"unit_price_minor":1500}]}
```

La primera petición exitosa devuelve `201` con el pedido y la cabecera `Location`.
Una repetición con la misma clave y los mismos datos normalizados devuelve la
misma respuesta inicial, incluso si el estado del pedido cambió después. La
normalización elimina espacios alrededor de identificadores y ordena los artículos.
La misma clave con datos diferentes devuelve `409 Conflict`. Para una operación
nueva debe usarse otra clave. Las claves son globales, se almacenan como SHA-256
y actualmente no tienen vencimiento; no deben contener datos sensibles.

Aplicar la migración `004_order_idempotency` antes de iniciar la API. Para ejecutar
la prueba de integración, proporcionar una URL de una base PostgreSQL de pruebas
en la que el usuario tenga permiso para crear y eliminar esquemas:

```powershell
$env:TEST_DATABASE_URL = 'postgres://usuario:contraseña@localhost:5433/base_pruebas?sslmode=disable'
go test ./internal/order/infrastructure/postgres -run TestIdempotencyPostgres -v
```

La prueba crea un esquema temporal aislado y lo elimina al finalizar. Sin esa
variable, la prueba de integración se omite; las pruebas unitarias siguen activas.
