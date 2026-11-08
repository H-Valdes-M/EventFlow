CREATE TABLE outbox_events (
    message_id UUID PRIMARY KEY,
    event_id UUID NOT NULL UNIQUE,
    event_type TEXT NOT NULL CHECK (char_length(event_type) BETWEEN 1 AND 100),
    order_id UUID NOT NULL REFERENCES orders (id),
    payload JSONB NOT NULL,
    available_at TIMESTAMPTZ NOT NULL,
    published_at TIMESTAMPTZ,
    publish_attempts INTEGER NOT NULL DEFAULT 0 CHECK (publish_attempts >= 0),
    last_publish_error TEXT,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX outbox_events_pending_idx
    ON outbox_events (available_at)
    WHERE published_at IS NULL;

CREATE INDEX outbox_events_order_id_idx
    ON outbox_events (order_id);

CREATE TABLE event_processing (
    consumer_name TEXT NOT NULL,
    event_id UUID NOT NULL,
    order_id UUID NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('completed', 'failed')),
    attempts INTEGER NOT NULL CHECK (attempts BETWEEN 1 AND 4),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (consumer_name, event_id)
);

CREATE TABLE notifications (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    order_id UUID NOT NULL REFERENCES orders (id),
    notification_type TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (order_id, notification_type)
);

CREATE TABLE failed_events (
    consumer_name TEXT NOT NULL,
    event_id UUID NOT NULL,
    order_id UUID NOT NULL,
    reason_sanitized TEXT NOT NULL,
    failed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (consumer_name, event_id)
);

CREATE TABLE rejected_deliveries (
    consumer_name TEXT NOT NULL,
    payload_digest CHAR(64) NOT NULL,
    reason_sanitized TEXT NOT NULL,
    rejected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (consumer_name, payload_digest)
);
