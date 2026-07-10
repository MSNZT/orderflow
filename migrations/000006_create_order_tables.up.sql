CREATE TABLE orders (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    status TEXT NOT NULL DEFAULT 'pending',
    total_price_cents BIGINT NOT NULL,
    currency TEXT NOT NULL,

    expires_at TIMESTAMPTZ NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT chk_orders_total_price_cents CHECK (total_price_cents >= 0),
    CONSTRAINT chk_orders_status CHECK (status IN ('pending', 'paid', 'canceled', 'expired')),
    CONSTRAINT chk_orders_currency CHECK (currency <> '')
);

CREATE INDEX idx_orders_user_id ON orders(user_id);
CREATE INDEX idx_orders_status ON orders(status);
CREATE INDEX idx_orders_created_at ON orders(created_at);

CREATE TABLE order_items (
    id UUID PRIMARY KEY,
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id),
    product_name TEXT NOT NULL,
    unit_price_cents BIGINT NOT NULL,
    currency TEXT NOT NULL,
    quantity INT NOT NULL,
    line_total_price_cents BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT chk_order_item_unit_price_cents CHECK (unit_price_cents > 0),
    CONSTRAINT chk_order_item_quantity CHECK (quantity > 0),
    CONSTRAINT chk_order_item_line_total_price_cents CHECK (line_total_price_cents > 0),
    CONSTRAINT chk_order_items_line_total_price_matches CHECK (line_total_price_cents = unit_price_cents * quantity),
    CONSTRAINT chk_order_item_currency CHECK (currency <> ''),
    CONSTRAINT chk_order_item_product_name CHECK (product_name <> ''),

    UNIQUE (order_id, product_id)
);

CREATE INDEX idx_order_items_product_id ON order_items(product_id);

