-- name: GetUserByTelegramID :one
SELECT * FROM users WHERE telegram_id = $1;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByReferralCode :one
SELECT * FROM users WHERE referral_code = $1;

-- name: CreateUser :one
INSERT INTO users (telegram_id, first_name, username, referral_code, referred_by_id, is_admin)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: UpdateUserInfo :exec
UPDATE users SET first_name = $2, username = $3, updated_at = NOW() WHERE id = $1;

-- name: UpdateUserLastInteraction :exec
UPDATE users SET last_interaction = NOW(), updated_at = NOW() WHERE id = $1;

-- name: UpdateUserBalance :one
UPDATE users SET balance = balance + $2, updated_at = NOW() WHERE id = $1
RETURNING balance;

-- name: UpdateUserBalanceWithCheck :one
UPDATE users SET balance = balance + $2, updated_at = NOW()
WHERE id = $1 AND balance + $2 >= 0
RETURNING balance;

-- name: UpdateUserReferralBalance :exec
UPDATE users SET referral_balance = referral_balance + $2, updated_at = NOW() WHERE id = $1;

-- name: SetUserActiveSession :exec
UPDATE users SET active_session_id = $2, updated_at = NOW() WHERE id = $1;

-- name: SetUserSelectedModel :exec
UPDATE users SET selected_model = $2, updated_at = NOW() WHERE id = $1;

-- name: SetUserTemperature :exec
UPDATE users SET temperature = $2, updated_at = NOW() WHERE id = $1;

-- name: ToggleUserShowCost :exec
UPDATE users SET show_cost = NOT show_cost, updated_at = NOW() WHERE id = $1;

-- name: ToggleUserContextEnabled :exec
UPDATE users SET context_enabled = NOT context_enabled, updated_at = NOW() WHERE id = $1;

-- name: ToggleUserSendUserInfo :exec
UPDATE users SET send_user_info = NOT send_user_info, updated_at = NOW() WHERE id = $1;

-- name: SetUserSessionTimeout :exec
UPDATE users SET session_timeout_ms = $2, updated_at = NOW() WHERE id = $1;

-- name: SetUserPremiumUntil :exec
UPDATE users SET premium_until = $2, updated_at = NOW() WHERE id = $1;

-- name: SetUserFavoriteModels :exec
UPDATE users SET favorite_models = $2, updated_at = NOW() WHERE id = $1;

-- name: SetUserIsAdmin :exec
UPDATE users SET is_admin = $2, updated_at = NOW() WHERE id = $1;

-- name: UpdateUserLastSkysmart :exec
UPDATE users SET last_skysmart = NOW(), updated_at = NOW() WHERE id = $1;

-- name: GetUserForUpdate :one
SELECT * FROM users WHERE id = $1 FOR UPDATE;

-- name: CountTotalUsers :one
SELECT COUNT(*) FROM users;

-- name: CountUsersCreatedAfter :one
SELECT COUNT(*) FROM users WHERE created_at >= $1;

-- name: CountPremiumUsers :one
SELECT COUNT(*) FROM users WHERE premium_until IS NOT NULL AND premium_until > NOW();
