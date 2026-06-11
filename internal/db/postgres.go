package db

import (
	"context"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Connect() (*pgxpool.Pool, error) {
	return pgxpool.New(
		context.Background(),
		os.Getenv("DATABASE_URL"),
	)
}
