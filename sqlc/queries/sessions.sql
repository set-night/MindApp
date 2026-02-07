-- name: GetSessionByID :one
SELECT * FROM chat_sessions WHERE id = $1;

-- name: GetSessionsByUserID :many
SELECT * FROM chat_sessions WHERE user_id = $1 ORDER BY updated_at DESC LIMIT $2 OFFSET $3;

-- name: CountSessionsByUserID :one
SELECT COUNT(*) FROM chat_sessions WHERE user_id = $1;

-- name: CreateSession :one
INSERT INTO chat_sessions (user_id, model, temperature)
VALUES ($1, $2, $3)
RETURNING *;

-- name: UpdateSessionModel :exec
UPDATE chat_sessions SET model = $2, updated_at = NOW() WHERE id = $1;

-- name: UpdateSessionTimestamp :exec
UPDATE chat_sessions SET updated_at = NOW() WHERE id = $1;

-- name: DeleteSession :exec
DELETE FROM chat_sessions WHERE id = $1;

-- name: DeleteAllUserSessions :exec
DELETE FROM chat_sessions WHERE user_id = $1;

-- name: DeleteOldestUserSessions :exec
DELETE FROM chat_sessions
WHERE id IN (
    SELECT cs.id FROM chat_sessions cs
    WHERE cs.user_id = $1
    ORDER BY cs.updated_at ASC
    LIMIT $2
);

-- name: AddSessionMessage :one
INSERT INTO session_messages (session_id, role, text, images, is_system)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetSessionMessages :many
SELECT * FROM session_messages WHERE session_id = $1 ORDER BY created_at ASC;

-- name: CountSessionMessages :one
SELECT COUNT(*) FROM session_messages WHERE session_id = $1;

-- name: GetFirstSessionMessage :one
SELECT * FROM session_messages WHERE session_id = $1 ORDER BY created_at ASC LIMIT 1;

-- name: AddMessageFile :exec
INSERT INTO message_files (message_id, file_type, url, name) VALUES ($1, $2, $3, $4);

-- name: GetMessageFiles :many
SELECT * FROM message_files WHERE message_id = $1;
