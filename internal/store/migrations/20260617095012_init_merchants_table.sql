-- +goose Up
CREATE TABLE IF NOT EXISTS merchants (
    id SERIAL PRIMARY KEY,
    account_id INTEGER REFERENCES accounts(id) NOT NULL,
    token VARCHAR(50) NOT NULL,
    name VARCHAR(120) NOT NULL,
    identify VARCHAR(20)
);

-- +goose Down
DROP TABLE IF EXISTS merchants;