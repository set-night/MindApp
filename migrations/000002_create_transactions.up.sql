CREATE TABLE transactions (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT REFERENCES users(id),
    group_id    BIGINT,
    amount      NUMERIC(20,10) NOT NULL,
    tx_type     TEXT NOT NULL CHECK (tx_type IN ('debit','credit')),
    description TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_transactions_user_id ON transactions(user_id);
CREATE INDEX idx_transactions_group_id ON transactions(group_id);
CREATE INDEX idx_transactions_created_at ON transactions(created_at);
