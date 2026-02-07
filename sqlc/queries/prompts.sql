-- name: GetOfficialPrompts :many
SELECT * FROM prompts WHERE is_official = true ORDER BY created_at DESC;

-- name: GetPromptByID :one
SELECT * FROM prompts WHERE id = $1;

-- name: GetPromptsByOwner :many
SELECT * FROM prompts WHERE owner_id = $1 ORDER BY created_at DESC;

-- name: CreatePrompt :one
INSERT INTO prompts (title, description, prompt_text, is_official, owner_id, price)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;
