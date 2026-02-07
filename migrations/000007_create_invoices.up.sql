CREATE TABLE invoices (
    id                   BIGSERIAL PRIMARY KEY,
    user_telegram_id     BIGINT NOT NULL,
    amount               NUMERIC(20,10) NOT NULL,
    cryptomus_invoice_id TEXT NOT NULL UNIQUE,
    status               TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','paid','failed')),
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_invoices_user_telegram_id ON invoices(user_telegram_id);
CREATE INDEX idx_invoices_status ON invoices(status);
CREATE INDEX idx_invoices_created_at ON invoices(created_at);
