CREATE TABLE users (
    id                BIGSERIAL PRIMARY KEY,
    telegram_id       BIGINT NOT NULL UNIQUE,
    is_admin          BOOLEAN NOT NULL DEFAULT FALSE,
    first_name        TEXT NOT NULL DEFAULT '',
    username          TEXT NOT NULL DEFAULT '',
    balance           NUMERIC(20,10) NOT NULL DEFAULT 0,
    referral_code     TEXT NOT NULL UNIQUE,
    referral_balance  NUMERIC(20,10) NOT NULL DEFAULT 0,
    referred_by_id    BIGINT REFERENCES users(id),
    premium_until     TIMESTAMPTZ,
    active_session_id BIGINT,
    last_interaction  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    selected_model    TEXT NOT NULL DEFAULT 'z-ai/glm-4.5-air:free',
    favorite_models   TEXT[] NOT NULL DEFAULT '{}',
    temperature       NUMERIC(3,1) NOT NULL DEFAULT 1.0,
    show_cost         BOOLEAN NOT NULL DEFAULT FALSE,
    send_user_info    BOOLEAN NOT NULL DEFAULT TRUE,
    context_enabled   BOOLEAN NOT NULL DEFAULT TRUE,
    session_timeout_ms INTEGER NOT NULL DEFAULT 600000,
    last_skysmart     TIMESTAMPTZ NOT NULL DEFAULT '1970-01-01',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_telegram_id ON users(telegram_id);
CREATE INDEX idx_users_referral_code ON users(referral_code);
