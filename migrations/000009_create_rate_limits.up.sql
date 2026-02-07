CREATE TABLE rate_limits (
    chat_id       BIGINT PRIMARY KEY,
    request_count INTEGER NOT NULL DEFAULT 0,
    window_start  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE active_requests (
    chat_id    BIGINT PRIMARY KEY,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
