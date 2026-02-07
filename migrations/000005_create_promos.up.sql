CREATE TABLE promos (
    id         BIGSERIAL PRIMARY KEY,
    code       TEXT NOT NULL UNIQUE,
    amount     NUMERIC(20,10) NOT NULL CHECK (amount >= 0),
    comment    TEXT NOT NULL DEFAULT '',
    max_uses   INTEGER NOT NULL DEFAULT 1 CHECK (max_uses >= 1),
    created_by BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE promo_activations (
    id           BIGSERIAL PRIMARY KEY,
    promo_id     BIGINT NOT NULL REFERENCES promos(id),
    user_id      BIGINT NOT NULL REFERENCES users(id),
    activated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(promo_id, user_id)
);
