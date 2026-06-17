package main

import (
	"log"
	"stripe-invoice-go/internal/store"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file loaded:", err)
	}

	pool, err := store.Connect()
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	st := store.New(pool)
	_ = st

	log.Println("Connected to database")
}
