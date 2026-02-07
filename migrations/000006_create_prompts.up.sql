CREATE TABLE prompts (
    id          BIGSERIAL PRIMARY KEY,
    title       TEXT NOT NULL,
    description TEXT NOT NULL,
    prompt_text TEXT NOT NULL,
    is_official BOOLEAN NOT NULL DEFAULT FALSE,
    owner_id    BIGINT REFERENCES users(id),
    price       NUMERIC(20,10) NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_prompts_is_official ON prompts(is_official);
CREATE INDEX idx_prompts_owner_id ON prompts(owner_id);
