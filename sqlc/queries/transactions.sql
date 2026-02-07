-- name: CreateTransaction :one
INSERT INTO transactions (user_id, group_id, amount, tx_type, description)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetUserTransactions :many
SELECT * FROM transactions WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3;

-- name: GetGroupTransactions :many
SELECT * FROM transactions WHERE group_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3;
