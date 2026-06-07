CREATE TABLE user_sessions (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    refresh_token_hash TEXT NOT NULL,
    user_agent TEXT,
    ip_address INET,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_used_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,

    CONSTRAINT user_sessions_refresh_token_hash_unique UNIQUE (refresh_token_hash),
    CONSTRAINT user_sessions_expires_at_check CHECK (expires_at > created_at),
    CONSTRAINT user_sessions_revoked_at_check CHECK (revoked_at IS NULL OR revoked_at >= created_at)
);

CREATE INDEX idx_user_sessions_user_id ON user_sessions(user_id);
CREATE INDEX idx_user_sessions_expires_at ON user_sessions(expires_at);