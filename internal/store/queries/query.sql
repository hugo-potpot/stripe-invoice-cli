-- name: GetAccount :one
SELECT * FROM accounts
WHERE id = $1 LIMIT 1;

-- name: ListAccounts :many
SELECT * FROM accounts;


-- name: CreateMerchant :one
INSERT INTO merchants (
  account_id, token, name, identify
) VALUES (
    @account_id, @token, @name, @identify
)
RETURNING *;

-- name: ListMerchantsByAccount :many
SELECT * FROM merchants
WHERE account_id = $1;