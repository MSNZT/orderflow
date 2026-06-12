CREATE TABLE products (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    price_cents BIGINT NOT NULL,
    currency TEXT NOT NULL DEFAULT 'RUB',
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT chk_price_cents CHECK (price_cents > 0),
    CONSTRAINT chk_currency CHECK (currency <> '')
);

