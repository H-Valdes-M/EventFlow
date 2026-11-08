CREATE TABLE order_idempotency (
    key_hash CHAR(64) PRIMARY KEY,
    request_hash CHAR(64) NOT NULL,
    -- La clave se reserva antes de insertar el pedido, en la misma transacción.
    order_id UUID NOT NULL REFERENCES orders(id) DEFERRABLE INITIALLY DEFERRED,
    response_status TEXT NOT NULL,
    response_total_minor BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
