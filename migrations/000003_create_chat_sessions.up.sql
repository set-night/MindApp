CREATE TABLE chat_sessions (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    model       TEXT NOT NULL,
    temperature NUMERIC(3,1) NOT NULL DEFAULT 0.7,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_chat_sessions_user_id ON chat_sessions(user_id);
CREATE INDEX idx_chat_sessions_created_at ON chat_sessions(created_at);

ALTER TABLE users ADD CONSTRAINT fk_users_active_session
    FOREIGN KEY (active_session_id) REFERENCES chat_sessions(id) ON DELETE SET NULL;

CREATE TABLE session_messages (
    id         BIGSERIAL PRIMARY KEY,
    session_id BIGINT NOT NULL REFERENCES chat_sessions(id) ON DELETE CASCADE,
    role       TEXT NOT NULL CHECK (role IN ('user','assistant','system')),
    text       TEXT NOT NULL DEFAULT '',
    images     TEXT[] NOT NULL DEFAULT '{}',
    is_system  BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_session_messages_session_id ON session_messages(session_id);

CREATE TABLE message_files (
    id         BIGSERIAL PRIMARY KEY,
    message_id BIGINT NOT NULL REFERENCES session_messages(id) ON DELETE CASCADE,
    file_type  TEXT NOT NULL CHECK (file_type IN ('image','video','audio','document')),
    url        TEXT NOT NULL,
    name       TEXT NOT NULL DEFAULT ''
);

CREATE INDEX idx_message_files_message_id ON message_files(message_id);
