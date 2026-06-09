// worker.go
// This is the heart of the distributed system.
// It pulls jobs from Redis and processes them using goroutines.
//
// KEY CONCEPT — Goroutines + WaitGroup:
// A goroutine is a lightweight thread managed by Go runtime.
// sync.WaitGroup lets us fire N goroutines and wait for ALL to finish.
//
// Without goroutines: 100 personas × 2s = 200s total
// With goroutines:    100 personas run simultaneously = ~2s total

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// Worker holds the Redis client and config
type Worker struct {
	redis       *redis.Client
	concurrency int  // max goroutines per simulation
}

// NewWorker creates a worker connected to Redis
func NewWorker(redisAddr string, concurrency int) *Worker {
	rdb := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})
	return &Worker{redis: rdb, concurrency: concurrency}
}

// Start begins the worker loop — runs forever, processing jobs.
// This is the "always listening" distributed worker pattern.
func (w *Worker) Start(ctx context.Context) {
	log.Println("Worker started — listening on simulation_queue")

	for {
		// BLPop = Blocking Left Pop
		// Waits up to 5 seconds for a job, then loops
		// This is how workers "listen" without burning CPU
		result, err := w.redis.BLPop(ctx, 5*time.Second, "simulation_queue").Result()
		if err != nil {
			if err == redis.Nil {
				// No job available — keep waiting
				continue
			}
			log.Printf("Redis error: %v", err)
			continue
		}

		// result[0] = queue name, result[1] = job JSON
		jobJSON := result[1]

		// Parse the job
		var job SimulationJob
		if err := json.Unmarshal([]byte(jobJSON), &job); err != nil {
			log.Printf("Failed to parse job: %v", err)
			continue
		}

		log.Printf("Processing job %s — %d personas for '%s'",
			job.JobID, job.Population, job.Demographic)

		// Update job status to "running"
		w.redis.HSet(ctx, fmt.Sprintf("job:%s", job.JobID), "status", "running")

		// Process the simulation
		startTime := time.Now()
		result2, err := w.ProcessSimulation(ctx, job)
		elapsed := time.Since(startTime)

		if err != nil {
			log.Printf("Simulation failed: %v", err)
			w.redis.HSet(ctx, fmt.Sprintf("job:%s", job.JobID), "status", "failed")
			continue
		}

		// Serialize result to JSON
		resultJSON, err := json.Marshal(result2)
		if err != nil {
			log.Printf("Failed to serialize result: %v", err)
			continue
		}

		// Write result to Redis — frontend polls for this
		w.redis.Set(ctx, fmt.Sprintf("result:%s", job.JobID), resultJSON, 1*time.Hour)

		// Update job status to "complete"
		w.redis.HSet(ctx, fmt.Sprintf("job:%s", job.JobID), "status", "complete")

		log.Printf("Job %s complete in %v", job.JobID, elapsed)
	}
}

// ProcessSimulation runs the full simulation using goroutines.
// THIS IS THE DISTRIBUTED SYSTEMS CORE OF YOUR THESIS.
func (w *Worker) ProcessSimulation(ctx context.Context, job SimulationJob) (*SimulationResult, error) {
	// Generate persona profiles
	personas := GeneratePersonas(job.Population, 0.55)

	// results slice — pre-allocated for thread safety
	// We allocate by index so goroutines don't conflict
	results := make([]PersonaResult, job.Population)

	// WaitGroup — tracks how many goroutines are still running
	// Add(n) = expect n goroutines
	// Done() = one goroutine finished
	// Wait() = block until all Done() calls received
	var wg sync.WaitGroup

	// Semaphore — limits concurrent goroutines to avoid rate limiting
	// Think of it as a "ticket system" — only N goroutines run at once
	sem := make(chan struct{}, w.concurrency)

	// Fire one goroutine per persona — THIS is the parallelism
	for i, persona := range personas {
		wg.Add(1) // tell WaitGroup to expect one more goroutine

		// Launch goroutine
		// 'go' keyword = run this function concurrently
		go func(idx int, p PersonaProfile) {
			defer wg.Done() // mark this goroutine as done when it returns

			// Acquire semaphore slot — blocks if concurrency limit reached
			sem <- struct{}{}
			defer func() { <-sem }() // release slot when done

			// Call Claude API for this persona
			results[idx] = EvaluatePersona(p, job)

		}(i, persona) // pass i and persona as arguments to avoid closure bugs
	}

	// Block here until ALL goroutines finish
	// This is the synchronization point
	wg.Wait()

	// Aggregate results into final simulation output
	return w.aggregateResults(job.JobID, results), nil
}

// aggregateResults computes scores and verdict from all persona results
func (w *Worker) aggregateResults(jobID string, results []PersonaResult) *SimulationResult {
	breakdown := ActionBreakdown{}
	inTargetCount := 0
	shownCount := 0

	for _, r := range results {
		shownCount++
		if r.InTarget {
			inTargetCount++
		}
		switch r.Action {
		case "like":
			breakdown.Likes++
		case "share":
			breakdown.Shares++
		case "comment":
			breakdown.Comments++
		default:
			breakdown.Scrolls++
		}
	}

	// Virality score — weighted by action type
	// Share=3pts, Comment=2pts, Like=1pt (mirrors Instagram algo weights)
	maxScore := shownCount * 3
	rawScore := breakdown.Shares*3 + breakdown.Comments*2 + breakdown.Likes
	virality := 0
	if maxScore > 0 {
		virality = (rawScore * 100) / maxScore
	}

	algoScore := virality * 12 / 10 // 1.2x multiplier
	engagement := 0
	if shownCount > 0 {
		engagement = ((shownCount - breakdown.Scrolls) * 100) / shownCount
	}

	inPct := 0
	if shownCount > 0 {
		inPct = (inTargetCount * 100) / shownCount
	}
	shareRate := 0
	if shownCount > 0 {
		shareRate = (breakdown.Shares * 100) / shownCount
	}

	// Compute verdict
	head, sub, verdict := computeVerdict(shareRate, inPct)

	return &SimulationResult{
		JobID:      jobID,
		Status:     "complete",
		Personas:   results,
		Virality:   virality,
		AlgoScore:  algoScore,
		Engagement: engagement,
		Breakdown:  breakdown,
		Verdict: VerdictData{
			Head:      head,
			Sub:       sub,
			Verdict:   verdict,
			InPct:     inPct,
			ShareRate: shareRate,
		},
	}
}

func computeVerdict(shareRate, inPct int) (string, string, string) {
	if shareRate > 12 && inPct > 60 {
		return "Viral hit.", "Strong in-demo spread with meaningful crossover. This content has legs.", "Breakout potential"
	} else if shareRate > 6 && inPct > 50 {
		return "Niche hit.", "Spreads well inside the target demo but never crossed over.", "Strong in-demo, no breakout"
	} else if inPct > 65 {
		return "Echo chamber.", "High in-target engagement but no crossover.", "Targeted, not viral"
	}
	return "Weak signal.", "Low engagement across both cohorts. Hook needs rework.", "Needs rework"
}

// randInt is a helper for random integers
func randInt(min, max int) int {
	return min + rand.Intn(max-min)
}