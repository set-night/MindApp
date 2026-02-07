-- name: GetPromoByCode :one
SELECT p.*, COUNT(pa.id)::int AS activation_count
FROM promos p
LEFT JOIN promo_activations pa ON pa.promo_id = p.id
WHERE UPPER(p.code) = UPPER($1)
GROUP BY p.id;

-- name: CreatePromo :one
INSERT INTO promos (code, amount, comment, max_uses, created_by)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: CheckPromoActivation :one
SELECT EXISTS(
    SELECT 1 FROM promo_activations WHERE promo_id = $1 AND user_id = $2
) AS activated;

-- name: CreatePromoActivation :exec
INSERT INTO promo_activations (promo_id, user_id) VALUES ($1, $2);

-- name: CountPromos :one
SELECT COUNT(*) FROM promos;

-- name: CountPromoActivations :one
SELECT COUNT(*) FROM promo_activations;
