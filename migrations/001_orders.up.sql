CREATE TABLE orders (
    id UUID PRIMARY KEY,
    customer_id VARCHAR(100) NOT NULL,
    total_minor BIGINT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT orders_customer_id_chk
        CHECK (btrim(customer_id) <> ''),
    CONSTRAINT orders_total_minor_chk
        CHECK (total_minor > 0),
    CONSTRAINT orders_status_chk
        CHECK (status IN ('pending', 'processed', 'failed'))
);

CREATE TABLE order_items (
    order_id UUID NOT NULL,
    product_id VARCHAR(100) NOT NULL,
    quantity BIGINT NOT NULL,
    unit_price_minor BIGINT NOT NULL,

    CONSTRAINT order_items_pk
        PRIMARY KEY (order_id, product_id),
    CONSTRAINT order_items_order_fk
        FOREIGN KEY (order_id)
        REFERENCES orders (id)
        ON DELETE CASCADE,
    CONSTRAINT order_items_product_chk
        CHECK (btrim(product_id) <> ''),
    CONSTRAINT order_items_quantity_chk
        CHECK (quantity BETWEEN 1 AND 1000000),
    CONSTRAINT order_items_price_chk
        CHECK (unit_price_minor > 0)
);

CREATE INDEX orders_customer_created_idx
    ON orders (customer_id, created_at DESC);

CREATE INDEX orders_status_created_idx
    ON orders (status, created_at DESC);
