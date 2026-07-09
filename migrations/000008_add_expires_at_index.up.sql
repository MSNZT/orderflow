CREATE INDEX idx_orders_pending_expires_at 
ON orders (expires_at) WHERE status = 'pending';