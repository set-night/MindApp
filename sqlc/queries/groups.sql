-- name: GetGroupByTelegramID :one
SELECT * FROM groups WHERE telegram_id = $1;

-- name: GetGroupByID :one
SELECT * FROM groups WHERE id = $1;

-- name: CreateGroup :one
INSERT INTO groups (telegram_id, group_username, group_name)
VALUES ($1, $2, $3)
RETURNING *;

-- name: UpdateGroupInfo :exec
UPDATE groups SET group_username = $2, group_name = $3, updated_at = NOW() WHERE id = $1;

-- name: UpdateGroupLastInteraction :exec
UPDATE groups SET last_interaction = NOW(), updated_at = NOW() WHERE id = $1;

-- name: UpdateGroupBalance :one
UPDATE groups SET balance = balance + $2, updated_at = NOW() WHERE id = $1
RETURNING balance;

-- name: UpdateGroupBalanceWithCheck :one
UPDATE groups SET balance = balance + $2, updated_at = NOW()
WHERE id = $1 AND balance + $2 >= 0
RETURNING balance;

-- name: SetGroupSelectedModel :exec
UPDATE groups SET selected_model = $2, updated_at = NOW() WHERE id = $1;

-- name: ToggleGroupShowCost :exec
UPDATE groups SET show_cost = NOT show_cost, updated_at = NOW() WHERE id = $1;

-- name: ToggleGroupContextEnabled :exec
UPDATE groups SET context_enabled = NOT context_enabled, updated_at = NOW() WHERE id = $1;

-- name: SetGroupThreadID :exec
UPDATE groups SET thread_id = $2, updated_at = NOW() WHERE id = $1;

-- name: SetGroupPremiumUntil :exec
UPDATE groups SET premium_until = $2, updated_at = NOW() WHERE id = $1;

-- name: GetGroupForUpdate :one
SELECT * FROM groups WHERE id = $1 FOR UPDATE;

-- name: AddGroupContextMessage :exec
INSERT INTO group_context_messages (group_id, role, text) VALUES ($1, $2, $3);

-- name: GetGroupContextMessages :many
SELECT * FROM group_context_messages WHERE group_id = $1 ORDER BY created_at ASC;

-- name: DeleteGroupContextMessages :exec
DELETE FROM group_context_messages WHERE group_id = $1;

-- name: CountGroupContextMessages :one
SELECT COUNT(*) FROM group_context_messages WHERE group_id = $1;

-- name: DeleteOldestGroupContextMessages :exec
DELETE FROM group_context_messages
WHERE id IN (
    SELECT gcm.id FROM group_context_messages gcm
    WHERE gcm.group_id = $1
    ORDER BY gcm.created_at ASC
    LIMIT $2
);
