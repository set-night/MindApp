-- name: CheckAndIncrementRateLimit :one
INSERT INTO rate_limits (chat_id, request_count, window_start)
VALUES ($1, 1, NOW())
ON CONFLICT (chat_id)
DO UPDATE SET
    request_count = CASE
        WHEN rate_limits.window_start < NOW() - INTERVAL '1 minute'
        THEN 1
        ELSE rate_limits.request_count + 1
    END,
    window_start = CASE
        WHEN rate_limits.window_start < NOW() - INTERVAL '1 minute'
        THEN NOW()
        ELSE rate_limits.window_start
    END
RETURNING request_count;

-- name: TrySetActiveRequest :one
INSERT INTO active_requests (chat_id, started_at)
VALUES ($1, NOW())
ON CONFLICT (chat_id) DO NOTHING
RETURNING chat_id;

-- name: RemoveActiveRequest :exec
DELETE FROM active_requests WHERE chat_id = $1;

-- name: CleanupStaleRequests :exec
DELETE FROM active_requests WHERE started_at < NOW() - INTERVAL '3 minutes';
