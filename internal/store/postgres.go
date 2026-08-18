package store

import (
	"context"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	UniqueViolation = "23505"
)

func Connect() (*pgxpool.Pool, error) {
	return pgxpool.New(
		context.Background(),
		os.Getenv("DATABASE_URL"),
	)
}
