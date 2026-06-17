-- name: GetAccount :one
SELECT * FROM accounts
WHERE id = $1 LIMIT 1;

-- name: CreateMerchant :one
INSERT INTO merchants (
  account_id, token, name, identify
) VALUES (
    @account_id, @token, @name, @identify
)
RETURNING *;