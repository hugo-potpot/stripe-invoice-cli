-- +goose Up
CREATE TABLE IF NOT EXISTS accounts (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL,
    cookie VARCHAR(255) NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS accounts;
