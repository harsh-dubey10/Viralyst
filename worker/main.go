// main.go
// Entry point for the Go worker service.
// Loads config, connects to Redis, starts the worker loop.

package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

func main() {
	// Load .env file (local dev). In Docker/production use real env vars instead.
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found — using system environment variables")
	}

	// Normalize Redis address: strip redis:// scheme if present so go-redis
	// Addr field gets a plain "host:port" string regardless of what .env contains.
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "localhost:6379"
	}
	redisAddr := redisURL
	if strings.HasPrefix(redisAddr, "redis://") {
		redisAddr = strings.TrimPrefix(redisAddr, "redis://")
	}
	if strings.HasPrefix(redisAddr, "rediss://") {
		redisAddr = strings.TrimPrefix(redisAddr, "rediss://")
	}

	concurrencyStr := os.Getenv("WORKER_CONCURRENCY")
	concurrency := 10 // default concurrency
	if concurrencyStr != "" {
		if c, err := strconv.Atoi(concurrencyStr); err == nil && c > 0 {
			concurrency = c
		}
	}

	log.Printf("Starting Viralyst worker — Redis: %s, Concurrency: %d", redisAddr, concurrency)

	// Create worker and start loop
	worker := NewWorker(redisAddr, concurrency)
	ctx := context.Background()
	worker.Start(ctx)
}
