// main.go
// Entry point for the Go worker service.
// Loads config, connects to Redis, starts the worker loop.

package main

import (
	"context"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

func main() {
	// Load .env file — reads ANTHROPIC_API_KEY, REDIS_URL etc.
	// In Docker this will come from environment variables instead
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found — using system environment variables")
	}

	// Read config from environment
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "localhost:6379" // default
	}

	concurrencyStr := os.Getenv("WORKER_CONCURRENCY")
	concurrency := 10 // default — max 10 Claude API calls at once
	if concurrencyStr != "" {
		if c, err := strconv.Atoi(concurrencyStr); err == nil {
			concurrency = c
		}
	}

	log.Printf("Starting Viralyst worker — Redis: %s, Concurrency: %d", redisURL, concurrency)

	// Create worker
	worker := NewWorker(redisURL, concurrency)

	// Create context — used for clean shutdown
	ctx := context.Background()

	// Start the worker loop — runs forever
	worker.Start(ctx)
}