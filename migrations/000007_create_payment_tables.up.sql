CREATE TABLE payments (
    id UUID PRIMARY KEY,
    order_id UUID NOT NULL REFERENCES orders(id),
    provider_payment_id TEXT NULL UNIQUE,
    idempotency_key UUID NOT NULL UNIQUE,
    status TEXT NOT NULL,
    amount_cents BIGINT NOT NULL,
    currency TEXT NOT NULL,
    confirmation_url TEXT NULL,
    test BOOLEAN NULL,
    cancellation_party TEXT NULL,
    cancellation_reason TEXT NULL,
    provider_created_at TIMESTAMPTZ NULL,
    succeeded_at TIMESTAMPTZ NULL,
    canceled_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT chk_payments_amount_cents CHECK (amount_cents > 0),
    CONSTRAINT chk_payments_status CHECK (status IN (
        'creating', 'pending', 'waiting_for_capture', 'succeeded', 'canceled', 'failed'
    )),
    CONSTRAINT chk_payments_currency CHECK (currency <> '')
);

CREATE INDEX idx_payments_order_id_created_at ON payments(order_id, created_at DESC);
CREATE UNIQUE INDEX ux_payments_active_order 
    ON payments(order_id) 
    WHERE status IN ('creating', 'pending', 'waiting_for_capture');
CREATE UNIQUE INDEX ux_payments_succeeded_order ON payments(order_id) WHERE status = 'succeeded';