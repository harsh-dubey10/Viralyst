package main

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type Worker struct {
	redis       *redis.Client
	concurrency int
}

func NewWorker(redisAddr string, concurrency int) *Worker {
	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	return &Worker{redis: rdb, concurrency: concurrency}
}

func (w *Worker) Start(ctx context.Context) {
	log.Println("Worker started — listening on simulation_queue")
	for {
		result, err := w.redis.BLPop(ctx, 5*time.Second, "simulation_queue").Result()
		if err != nil {
			if err == redis.Nil {
				continue
			}
			log.Printf("Redis error: %v", err)
			continue
		}

		jobJSON := result[1]
		var job SimulationJob
		if err := json.Unmarshal([]byte(jobJSON), &job); err != nil {
			log.Printf("Failed to parse job: %v", err)
			continue
		}

		log.Printf("Processing job %s — %d personas for '%s'", job.JobID, job.Population, job.Demographic)
		w.redis.HSet(ctx, fmt.Sprintf("job:%s", job.JobID), "status", "running")

		startTime := time.Now()
		result2, err := w.ProcessSimulation(ctx, job)
		elapsed := time.Since(startTime)

		if err != nil {
			log.Printf("Simulation failed: %v", err)
			w.redis.HSet(ctx, fmt.Sprintf("job:%s", job.JobID), "status", "failed")
			continue
		}

		resultJSON, err := json.Marshal(result2)
		if err != nil {
			log.Printf("Failed to serialize result: %v", err)
			continue
		}

		w.redis.Set(ctx, fmt.Sprintf("result:%s", job.JobID), resultJSON, 1*time.Hour)
		w.redis.HSet(ctx, fmt.Sprintf("job:%s", job.JobID), "status", "complete")
		log.Printf("Job %s complete in %v", job.JobID, elapsed)
	}
}

func getJobSeed(job SimulationJob) int64 {
	h := fnv.New64a()
	h.Write([]byte(job.Demographic))
	h.Write([]byte(job.ContentDescription))
	h.Write([]byte(job.VideoFilename))
	h.Write([]byte(fmt.Sprintf("%d_%d", job.Population, job.Boost)))
	// Mix in the job ID (UUID) so every simulation run is unique even with
	// identical inputs, providing genuine stochastic variation.
	h.Write([]byte(job.JobID))
	return int64(h.Sum64())
}

func (w *Worker) ProcessSimulation(ctx context.Context, job SimulationJob) (*SimulationResult, error) {
	seed := getJobSeed(job)
	r := rand.New(rand.NewSource(seed))

	personas := GeneratePersonas(job.Population, 0.55, r)

	// Round 1: Determine initial seed population based on Boost
	seedCount := job.Boost * 4
	if seedCount < 6 {
		seedCount = 6
	}
	if seedCount > job.Population/2 {
		seedCount = job.Population / 2
	}

	inTargetIndices := []int{}
	outTargetIndices := []int{}
	for i, p := range personas {
		if p.InTarget {
			inTargetIndices = append(inTargetIndices, i)
		} else {
			outTargetIndices = append(outTargetIndices, i)
		}
	}

	seedIndices := make(map[int]bool)
	targetSeedsNeeded := seedCount * 3 / 4
	for i := 0; i < targetSeedsNeeded && i < len(inTargetIndices); i++ {
		seedIndices[inTargetIndices[i]] = true
	}
	for i := 0; len(seedIndices) < seedCount && i < len(outTargetIndices); i++ {
		seedIndices[outTargetIndices[i]] = true
	}

	for idx := range personas {
		if seedIndices[idx] {
			personas[idx].Shown = true
		}
	}

	results := make([]PersonaResult, job.Population)
	var wg sync.WaitGroup
	sem := make(chan struct{}, w.concurrency)

	for i, persona := range personas {
		wg.Add(1)
		go func(idx int, p PersonaProfile) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			personaSeed := seed + int64(p.ID*1337)
			pr := rand.New(rand.NewSource(personaSeed))
			res := EvaluatePersona(p, job, pr)
			if p.Shown {
				res.Round = 1
			}
			results[idx] = res
		}(i, persona)
	}

	wg.Wait()

	results, edges := buildSpreadGraph(personas, results, job, r)
	result := w.aggregateResults(job.JobID, results)
	result.Edges = edges
	return result, nil
}

func (w *Worker) aggregateResults(jobID string, results []PersonaResult) *SimulationResult {
	breakdown := ActionBreakdown{}
	inTargetCount := 0
	shownCount := 0

	for _, r := range results {
		if !r.Shown || r.Action == "never" {
			continue
		}
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

	maxScore := shownCount * 3
	rawScore := breakdown.Shares*3 + breakdown.Comments*2 + breakdown.Likes
	virality := 0
	if maxScore > 0 {
		virality = (rawScore * 100) / maxScore
	}

	algoScore := virality * 12 / 10
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

	skipRate := 0
	if shownCount > 0 {
		skipRate = (breakdown.Scrolls * 100) / shownCount
	}

	head, sub, verdict := computeVerdict(shareRate, inPct, skipRate)

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

func buildSpreadGraph(personas []PersonaProfile, results []PersonaResult, job SimulationJob, r *rand.Rand) ([]PersonaResult, []EdgeResult) {
	edges := []EdgeResult{}

	resultMap := make(map[int]*PersonaResult)
	for i := range results {
		resultMap[results[i].PersonaID] = &results[i]
	}

	reached := make(map[int]bool)
	for _, res := range results {
		if res.Shown {
			reached[res.PersonaID] = true
			if resultMap[res.PersonaID].Round == 0 {
				resultMap[res.PersonaID].Round = 1
			}
		}
	}

	// Round 2: Direct Shares (Solid Red Lines)
	sharers := []PersonaResult{}
	for _, res := range results {
		if res.Shown && res.Action == "share" {
			sharers = append(sharers, res)
		}
	}

	for _, sharer := range sharers {
		sharerPersona := personas[sharer.PersonaID-1]
		reach := sharerPersona.Followers / 1500
		if reach < 1 {
			reach = 1
		}
		if reach > 3 {
			reach = 3
		}

		targets := []int{}
		for _, p := range personas {
			if !reached[p.ID] {
				targets = append(targets, p.ID)
			}
		}

		r.Shuffle(len(targets), func(i, j int) {
			targets[i], targets[j] = targets[j], targets[i]
		})

		for i := 0; i < reach && i < len(targets); i++ {
			targetID := targets[i]
			edges = append(edges, EdgeResult{From: sharer.PersonaID, To: targetID, Type: "direct", Round: 2})
			reached[targetID] = true

			if res, ok := resultMap[targetID]; ok {
				res.Shown = true
				res.Round = 2
				targetPersona := personas[targetID-1]
				targetPersona.Shown = true
				pr := rand.New(rand.NewSource(r.Int63()))
				evalRes := EvaluatePersona(targetPersona, job, pr)
				res.Action = evalRes.Action
				res.Watch = evalRes.Watch
				res.Why = evalRes.Why
				res.Round = 2
				if res.Action == "never" {
					res.Action = "skip"
				}
			}
		}
	}

	// Round 3: Algorithmic Recommendation (Dotted Grey Lines)
	engagers := []PersonaResult{}
	for _, res := range results {
		if res.Shown && (res.Action == "like" || res.Action == "comment" || res.Action == "share") {
			engagers = append(engagers, res)
		}
	}

	for _, engager := range engagers {
		if r.Float64() > 0.60 {
			continue
		}
		engagerPersona := personas[engager.PersonaID-1]

		for _, p := range personas {
			if reached[p.ID] {
				continue
			}
			overlap := interestOverlap(engagerPersona.Interests, p.Interests)
			if overlap > 0 || r.Float64() < 0.15 {
				edges = append(edges, EdgeResult{From: engager.PersonaID, To: p.ID, Type: "algo", Round: 3})
				reached[p.ID] = true

				if res, ok := resultMap[p.ID]; ok {
					res.Shown = true
					res.Round = 3
					targetPersona := personas[p.ID-1]
					targetPersona.Shown = true
					pr := rand.New(rand.NewSource(r.Int63()))
					evalRes := EvaluatePersona(targetPersona, job, pr)
					res.Action = evalRes.Action
					res.Watch = evalRes.Watch
					res.Why = evalRes.Why
					res.Round = 3
					if res.Action == "never" {
						res.Action = "skip"
					}
				}
				break
			}
		}
	}

	// Unreached personas remain Grey Nodes (Shown = false, Action = "never", Round = 0)
	for i := range results {
		if !reached[results[i].PersonaID] {
			results[i].Shown = false
			results[i].Action = "never"
			results[i].Round = 0
			results[i].Watch = 0
			results[i].Why = "Not reached by network share or algorithm."
		}
	}

	return results, edges
}

func interestOverlap(a, b []string) int {
	count := 0
	for _, x := range a {
		for _, y := range b {
			if x == y {
				count++
			}
		}
	}
	return count
}

func computeVerdict(shareRate, inPct, skipRate int) (string, string, string) {
	if shareRate >= 9 && inPct >= 48 {
		return "Viral hit.", "Strong in-demo spread with organic cross-demo viral spread. High viral coefficient.", "Breakout potential"
	} else if shareRate >= 5 && inPct >= 40 {
		return "Niche hit.", "Deep engagement inside target demographic, but limited out-of-target crossover.", "Strong in-demo, no breakout"
	} else if inPct >= 55 && shareRate < 5 {
		return "Echo chamber.", "High in-target retention but fails to trigger algorithmic expansion.", "Targeted, not viral"
	} else if skipRate >= 65 {
		return "Weak hook.", "Over 65% of viewers scrolled past in the first 3 seconds. Opening hook needs rework.", "Needs rework"
	}
	return "Weak signal.", "Low engagement and share rate across both target and out-of-target cohorts.", "Needs rework"
}