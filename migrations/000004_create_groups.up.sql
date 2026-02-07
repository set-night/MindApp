CREATE TABLE groups (
    id              BIGSERIAL PRIMARY KEY,
    telegram_id     BIGINT NOT NULL UNIQUE,
    balance         NUMERIC(20,10) NOT NULL DEFAULT 0,
    group_username  TEXT NOT NULL DEFAULT '',
    group_name      TEXT NOT NULL DEFAULT '',
    premium_until   TIMESTAMPTZ,
    last_interaction TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    thread_id       INTEGER,
    selected_model  TEXT NOT NULL DEFAULT 'z-ai/glm-4.5-air:free',
    show_cost       BOOLEAN NOT NULL DEFAULT FALSE,
    context_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_groups_telegram_id ON groups(telegram_id);

ALTER TABLE transactions ADD CONSTRAINT fk_transactions_group
    FOREIGN KEY (group_id) REFERENCES groups(id);

CREATE TABLE group_context_messages (
    id       BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    role     TEXT NOT NULL CHECK (role IN ('user','assistant','system')),
    text     TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_group_context_group_id ON group_context_messages(group_id);
