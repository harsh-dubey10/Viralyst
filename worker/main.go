// main.go
// Entry point for the Go worker service.
// Loads config, connects to Redis, starts the worker loop.

package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/joho/godotenv"
)

func main() {
	evalPath := flag.String("eval", "", "score posts with known outcomes from this CSV and exit (see eval.go)")
	offline := flag.Bool("offline", false, "with -eval: use the offline model even if Groq keys are set")
	flag.Parse()

	// Load .env file (local dev). In Docker/production use real env vars instead.
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found — using system environment variables")
	}

	// REDIS_URL may be a full URL (redis://[:pass@]host:port/db) or a plain "host:port".
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "localhost:6379"
	}

	concurrencyStr := os.Getenv("WORKER_CONCURRENCY")
	concurrency := 10 // default concurrency
	if concurrencyStr != "" {
		if c, err := strconv.Atoi(concurrencyStr); err == nil && c > 0 {
			concurrency = c
		}
	}

	if *evalPath != "" {
		if *offline {
			llmDisabled.Store(true)
		}
		if err := runEval(*evalPath, &Worker{concurrency: concurrency}, os.Stdout); err != nil {
			log.Fatalf("eval: %v", err)
		}
		return
	}

	log.Printf("Starting Viralyst worker — Redis: %s, Concurrency: %d", redisURL, concurrency)

	worker, err := NewWorker(redisURL, concurrency)
	if err != nil {
		log.Fatalf("Invalid REDIS_URL %q: %v", redisURL, err)
	}

	// Stop cleanly on Ctrl+C / docker stop: finish the current job, then exit.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	worker.Start(ctx)
	log.Println("Worker stopped")
}
