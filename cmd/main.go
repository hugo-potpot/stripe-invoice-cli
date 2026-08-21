package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"stripe-invoice-go/internal/archive"
	"stripe-invoice-go/internal/cli"
	"stripe-invoice-go/internal/config"
	"stripe-invoice-go/internal/mailer"
	"stripe-invoice-go/internal/store"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

var cfg config.Config

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

	archiver := archive.NewFileArchiver(cfg.ArchiveDir)
	smtpMailer := mailer.NewSMTPMailer(
		cfg.SMTP.Host,
		cfg.SMTP.Port,
		cfg.SMTP.Username,
		cfg.SMTP.Password,
		cfg.SMTP.From,
		cfg.SMTP.To,
	)

	// Init cobra CLI
	rootCmd := cli.NewRootCmd(queries, archiver, smtpMailer)
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

	var err error
	cfg, err = config.Load()
	if err != nil {
		slog.Error("load config failed", "error", err)
		os.Exit(1)
	}

	pgxPool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}

	c <- pgxPool
}
