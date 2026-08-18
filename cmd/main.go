package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"stripe-invoice-go/internal/cli"
	"stripe-invoice-go/internal/store"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	// Initialize context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle SIGINT (Ctrl+C) for graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt)
		<-sigChan
		cancel()
	}()

	// Create a channel to receive the connection instance
	poolChan := make(chan *pgxpool.Pool)

	// Start the run goroutine to initialize the connection instance
	go run(ctx, poolChan)

	// Wait for the connection instance from the run goroutine
	pool := <-poolChan
	defer pool.Close()

	queries := store.New(pool)

	// Init cobra CLI
	rootCmd := cli.NewRootCmd(queries)
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		slog.Error("command failed", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, c chan<- *pgxpool.Pool) {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file loaded:", err)
		os.Exit(1)
	}

	dbConnectionString := os.Getenv("DATABASE_URL")

	if dbConnectionString == "" {
		log.Println("no database connection string provided")
		os.Exit(1)
	}

	pgxPool, err := pgxpool.New(ctx, dbConnectionString)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}

	c <- pgxPool
}
