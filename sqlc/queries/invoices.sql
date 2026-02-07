-- name: CreateInvoice :one
INSERT INTO invoices (user_telegram_id, amount, cryptomus_invoice_id)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetInvoiceByCryptomusID :one
SELECT * FROM invoices WHERE cryptomus_invoice_id = $1;

-- name: UpdateInvoiceStatus :exec
UPDATE invoices SET status = $2 WHERE id = $1;

-- name: GetPendingInvoiceByUser :one
SELECT * FROM invoices
WHERE user_telegram_id = $1 AND status = 'pending' AND created_at > NOW() - INTERVAL '30 minutes'
ORDER BY created_at DESC
LIMIT 1;

-- name: DeleteInvoice :exec
DELETE FROM invoices WHERE id = $1;
