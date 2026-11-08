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
