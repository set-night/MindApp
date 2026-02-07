CREATE TABLE pay_tasks (
    id            BIGSERIAL PRIMARY KEY,
    title         TEXT NOT NULL,
    telegram_link TEXT NOT NULL,
    channel_id    TEXT NOT NULL,
    reward        NUMERIC(20,10) NOT NULL,
    time_limit    TIMESTAMPTZ,
    max_people    INTEGER
);

CREATE TABLE pay_task_completions (
    id           BIGSERIAL PRIMARY KEY,
    task_id      BIGINT NOT NULL REFERENCES pay_tasks(id),
    user_id      BIGINT NOT NULL REFERENCES users(id),
    completed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(task_id, user_id)
);
