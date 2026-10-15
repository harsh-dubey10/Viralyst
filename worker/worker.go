package main

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"log"
	"math"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type Worker struct {
	redis       *redis.Client
	concurrency int
}

func NewWorker(redisURL string, concurrency int) (*Worker, error) {
	opts := &redis.Options{Addr: redisURL}
	if strings.Contains(redisURL, "://") {
		parsed, err := redis.ParseURL(redisURL)
		if err != nil {
			return nil, err
		}
		opts = parsed
	}
	return &Worker{redis: redis.NewClient(opts), concurrency: concurrency}, nil
}

const resultTTL = 1 * time.Hour

func (w *Worker) Start(ctx context.Context) {
	log.Println("Worker started — listening on simulation_queue")
	backoff := time.Second
	for ctx.Err() == nil {
		// The API LPUSHes, so popping from the right makes the queue first-in first-out.
		result, err := w.redis.BRPop(ctx, 5*time.Second, "simulation_queue").Result()
		if err != nil {
			if err == redis.Nil || ctx.Err() != nil {
				continue
			}
			// Redis is down or unreachable: back off instead of spinning.
			log.Printf("Redis error: %v (retrying in %v)", err, backoff)
			select {
			case <-ctx.Done():
			case <-time.After(backoff):
			}
			if backoff < 30*time.Second {
				backoff *= 2
			}
			continue
		}
		backoff = time.Second

		var job SimulationJob
		if err := json.Unmarshal([]byte(result[1]), &job); err != nil {
			log.Printf("Failed to parse job: %v", err)
			continue
		}
		w.handleJob(ctx, job)
	}
}

// setStatus records a job's status and keeps its hash alive as long as its result, so a
// slow job can't outlive its status key (or recreate it without a TTL).
func (w *Worker) setStatus(ctx context.Context, jobID, status string) {
	key := fmt.Sprintf("job:%s", jobID)
	pipe := w.redis.TxPipeline()
	pipe.HSet(ctx, key, "status", status)
	pipe.Expire(ctx, key, resultTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		log.Printf("Failed to set job %s status=%s: %v", jobID, status, err)
	}
}

// handleJob runs one simulation and records its outcome. A panic inside the
// simulation marks the job failed rather than crashing the whole worker.
func (w *Worker) handleJob(ctx context.Context, job SimulationJob) {
	// A job that has started is finished and recorded even if shutdown begins meanwhile;
	// with the cancelled ctx the result write would fail and leave it "running".
	ctx = context.WithoutCancel(ctx)
	defer func() {
		if rec := recover(); rec != nil {
			log.Printf("Job %s panicked: %v", job.JobID, rec)
			w.setStatus(ctx, job.JobID, "failed")
		}
	}()

	if job.Population < 1 || job.Boost < 1 {
		log.Printf("Job %s rejected: invalid population=%d boost=%d", job.JobID, job.Population, job.Boost)
		w.setStatus(ctx, job.JobID, "failed")
		return
	}

	log.Printf("Processing job %s — %d personas for '%s'", job.JobID, job.Population, job.Demographic)
	w.setStatus(ctx, job.JobID, "running")

	startTime := time.Now()
	simResult := w.ProcessSimulation(job)

	resultJSON, err := json.Marshal(simResult)
	if err != nil {
		log.Printf("Failed to serialize result for job %s: %v", job.JobID, err)
		w.setStatus(ctx, job.JobID, "failed")
		return
	}

	if err := w.redis.Set(ctx, fmt.Sprintf("result:%s", job.JobID), resultJSON, resultTTL).Err(); err != nil {
		log.Printf("Failed to store result for job %s: %v", job.JobID, err)
		w.setStatus(ctx, job.JobID, "failed")
		return
	}
	w.setStatus(ctx, job.JobID, "complete")
	log.Printf("Job %s complete in %v", job.JobID, time.Since(startTime))
}

func getJobSeed(job SimulationJob) int64 {
	h := fnv.New64a()
	h.Write([]byte(job.Demographic))
	h.Write([]byte(job.Niche))
	h.Write([]byte(job.ContentDescription))
	h.Write([]byte(job.VideoFilename))
	h.Write([]byte(fmt.Sprintf("%d_%d", job.Population, job.Boost)))
	// Mix in the job ID (UUID) so every simulation run is unique even with
	// identical inputs, providing genuine stochastic variation.
	h.Write([]byte(job.JobID))
	return int64(h.Sum64())
}

// simContext carries the parsed inputs every persona evaluation needs.
type simContext struct {
	job       SimulationJob
	audience  AudienceProfile
	content   ContentProfile
	video     VideoInsight
	topicSrc  string // "llm" or "keywords"
	hookSrc   string // "llm" or "keywords"
	nicheName string // the niche as given, trimmed
}

// newSimContext parses the job's inputs. Text extracted from an uploaded video is
// appended to the hook description, so both engines react to what the video shows.
// The niche, if given, always leads the audience's interests; a niche the lexicon
// doesn't cover becomes a custom topic. If readTopics is set (an LLM is available) it
// reads the audience and hook topics and rates the hook; otherwise, or if it fails,
// topics come from keyword matching and the hook from keyword rules.
func newSimContext(job SimulationJob, video VideoInsight, readTopics topicReader) *simContext {
	hasText := strings.TrimSpace(job.ContentDescription) != ""
	// Topics and hook strength come from what the viewer is told; the personas also
	// get the scene description.
	scored := job
	scored.ContentDescription = strings.TrimSpace(job.ContentDescription + "\n" + video.HookText())
	if hook := video.Hook(); hook != "" {
		job.ContentDescription = strings.TrimSpace(job.ContentDescription + "\n" + hook)
	}

	nicheTopics, custom := resolveNiche(job.Niche)
	var extra []topic
	var extraNames []string
	if custom != nil {
		extra, extraNames = []topic{*custom}, []string{custom.Name}
	}
	audienceDesc := strings.TrimSpace(strings.Join(strings.Fields(job.Niche+" — "+job.Demographic), " "))
	audienceDesc = strings.Trim(audienceDesc, " —")

	audience := ParseAudience(job.Demographic)
	audience.Topics = detectTopics(job.Demographic, extra...)
	contentTopics, topicSrc, hookSrc := []string(nil), "keywords", "keywords"
	var rating *HookRating
	if readTopics != nil {
		if t, err := readTopics(job.Demographic, scored.ContentDescription, audienceDesc, extraNames); err != nil {
			log.Printf("Job %s: LLM topic reading failed, using keywords: %v", job.JobID, err)
		} else {
			audience.Topics, contentTopics, topicSrc, rating = t.Audience, t.Content, "llm", t.Hook
		}
	}
	audience.Topics = mergeTopics(nicheTopics, audience.Topics)
	audience.Core = nicheTopics
	if custom != nil && contentTopics != nil && anyPhrase(normalize(scored.ContentDescription), custom.Keywords) {
		// The LLM may not tag a post with a niche it has never seen; the niche's own
		// words are a reliable signal, so a hook that names it counts as on-niche.
		contentTopics = mergeTopics([]string{custom.Name}, contentTopics)
	}

	content := AnalyzeContent(scored, contentTopics, extra...)
	if len(content.Topics) == 0 && len(nicheTopics) > 0 && content.HasText {
		// A creator's post is about their niche unless it says otherwise ("3 opening
		// traps" on a chess channel needn't say "chess"). Without this, every such hook
		// read as topic-less and was shown to no one who cares about it.
		content.Topics = nicheTopics
		content.Issues = removeString(content.Issues, noTopicIssue)
	}
	if rating != nil && content.HasText {
		// The LLM judged this hook for this audience; its fixes replace the generic
		// keyword-rule advice, but issues about missing topics or text stay. The score is
		// blended with the keyword rules: two independent judges are steadier than one
		// (temperature-0 ratings still bunch around "typical" for very different hooks).
		content.HookScore = clampF(llmHookWeight*float64(rating.Score)/100+(1-llmHookWeight)*content.HookScore, 0.05, 0.95)
		content.Signals = rating.Strengths
		content.Issues = rating.Fixes
		if len(content.Topics) == 0 {
			content.Issues = append(content.Issues, noTopicIssue)
		}
		hookSrc = "llm"
	}
	sc := &simContext{job: job, audience: audience, content: content, video: video, topicSrc: topicSrc, hookSrc: hookSrc, nicheName: strings.Join(strings.Fields(job.Niche), " ")}
	if job.VideoFilename != "" && video.Note != "" {
		var fallback string
		switch {
		case video.Hook() != "":
			fallback = "results use the parts that were extracted."
		case hasText:
			fallback = "results use your text description only."
		default:
			fallback = "add a text description of the first 3 seconds so personas have something to react to."
		}
		sc.content.Issues = append(sc.content.Issues, fmt.Sprintf("The video couldn't be fully analysed (%s) — %s", video.Note, fallback))
	}
	return sc
}

// llmHookWeight is the LLM hook rating's share of the hook score; keyword rules get the rest.
const llmHookWeight = 0.6

// evalTask is one persona evaluation: which result slot to fill and the RNG seed to use.
type evalTask struct {
	persona PersonaProfile
	result  *PersonaResult
	seed    int64
	round   int
}

// evaluateParallel runs persona evaluations (LLM calls) concurrently, bounded by the
// worker's concurrency limit. Seeds are drawn up front so output stays reproducible.
func (w *Worker) evaluateParallel(sc *simContext, tasks []evalTask) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, w.concurrency)
	for _, t := range tasks {
		wg.Add(1)
		sem <- struct{}{}
		go func(t evalTask) {
			defer wg.Done()
			defer func() { <-sem }()
			defer func() {
				// handleJob's recover can't see this goroutine; without this a panic in
				// one persona would kill the worker and strand the job.
				if rec := recover(); rec != nil {
					log.Printf("Persona %d evaluation panicked: %v", t.persona.ID, rec)
					*t.result = fallbackResult(t.persona, sc, rand.New(rand.NewSource(t.seed)))
					t.result.Shown, t.result.Round = true, t.round
				}
			}()
			res := EvaluatePersona(t.persona, sc, rand.New(rand.NewSource(t.seed)))
			if res.Action == "never" {
				res.Action = "skip"
			}
			res.Shown = true
			res.Round = t.round
			*t.result = res
		}(t)
	}
	wg.Wait()
}

// In-target share of the generated population, and of the round-1 seed audience.
const (
	inTargetRatio     = 0.5
	seedInTargetRatio = 0.75
)

// offlineReplicates is how many independent runs the offline engine simulates to
// measure how stable its verdict is (each takes ~2ms).
const offlineReplicates = 25

// ProcessSimulation runs the simulation. With an LLM configured it runs once. Offline,
// it runs several replicates with different seeds, reports the most common verdict
// with its agreement rate, and returns the first replicate that produced it.
func (w *Worker) ProcessSimulation(job SimulationJob) *SimulationResult {
	seed := getJobSeed(job)
	var readTopics topicReader
	if hasGroqAPIKeys() {
		readTopics = readTopicsWithLLM
	}
	sc := newSimContext(job, AnalyzeVideo(job.VideoFilename), readTopics)

	if hasGroqAPIKeys() {
		res := w.simulateOnce(sc, seed)
		res.Analysis.Confidence = -1
		res.Analysis.VerdictDistribution = map[string]int{}
		return res
	}

	runs := make([]*SimulationResult, offlineReplicates)
	dist := map[string]int{}
	for i := range runs {
		runs[i] = w.simulateOnce(sc, seed+int64(i)*7919)
		dist[runs[i].Verdict.Head]++
	}
	modal := runs[0].Verdict.Head
	for _, res := range runs { // first-seen order breaks ties deterministically
		if dist[res.Verdict.Head] > dist[modal] {
			modal = res.Verdict.Head
		}
	}
	// Report the typical run, not the luckiest or unluckiest: of the runs with the modal
	// verdict, the one whose virality is closest to their median.
	modalRuns := []*SimulationResult{}
	for _, res := range runs {
		if res.Verdict.Head == modal {
			modalRuns = append(modalRuns, res)
		}
	}
	scores := make([]int, len(modalRuns))
	for i, res := range modalRuns {
		scores[i] = res.Virality
	}
	sort.Ints(scores)
	median := scores[len(scores)/2]
	best := modalRuns[0]
	for _, res := range modalRuns[1:] {
		if absInt(res.Virality-median) < absInt(best.Virality-median) {
			best = res
		}
	}
	best.Analysis.VerdictDistribution = dist
	best.Analysis.Replicates = offlineReplicates
	best.Analysis.Confidence = dist[modal] * 100 / offlineReplicates
	return best
}

func (w *Worker) simulateOnce(sc *simContext, seed int64) *SimulationResult {
	job := sc.job
	r := rand.New(rand.NewSource(seed))

	personas := GeneratePersonas(job.Population, sc.audience, inTargetRatio, r)

	// Round 1: the platform shows the post to a seed audience, mostly in-target.
	// Boost 1–10 shows it to 12.5–35% of the population, so a bigger population means a
	// bigger (less noisy) seed sample rather than the same ~24 people.
	seedCount := clampInt(int(math.Round(float64(job.Population)*(0.10+0.025*float64(job.Boost)))), 6, job.Population/2)
	inTargetIdx, outTargetIdx := []int{}, []int{}
	for i, p := range personas {
		if p.InTarget {
			inTargetIdx = append(inTargetIdx, i)
		} else {
			outTargetIdx = append(outTargetIdx, i)
		}
	}
	seedIdx := map[int]bool{}
	for i := 0; i < int(float64(seedCount)*seedInTargetRatio) && i < len(inTargetIdx); i++ {
		seedIdx[inTargetIdx[i]] = true
	}
	for i := 0; len(seedIdx) < seedCount && i < len(outTargetIdx); i++ {
		seedIdx[outTargetIdx[i]] = true
	}

	results := make([]PersonaResult, job.Population)
	tasks := []evalTask{}
	for idx := range personas {
		if seedIdx[idx] {
			personas[idx].Shown = true
			tasks = append(tasks, evalTask{
				persona: personas[idx],
				result:  &results[idx],
				seed:    seed + int64(personas[idx].ID*1337),
				round:   1,
			})
		} else {
			results[idx] = EvaluatePersona(personas[idx], sc, nil)
		}
	}
	w.evaluateParallel(sc, tasks)

	edges, pushRate := w.buildSpreadGraph(sc, personas, results, r)
	result := aggregateResults(sc, results, len(seedIdx), pushRate)
	result.Edges = edges
	return result
}

// Algorithmic push: the platform recommends the post to unreached users in
// proportion to how well it performed with the people who saw it.
const (
	pushEngagementFloor = 0.08 // weighted engagement below this earns no push
	pushSlope           = 2.5
	maxPushRate         = 0.6 // at most 60% of unreached users get recommended it
)

// buildSpreadGraph runs rounds 2 (direct shares) and 3 (algorithmic push), filling
// results in place. It returns the edges and the push rate the algorithm granted.
func (w *Worker) buildSpreadGraph(sc *simContext, personas []PersonaProfile, results []PersonaResult, r *rand.Rand) ([]EdgeResult, float64) {
	edges := []EdgeResult{}
	reached := map[int]bool{}
	for _, res := range results {
		if res.Shown {
			reached[res.PersonaID] = true
		}
	}

	// queue marks a persona as reached in the given round and schedules its evaluation.
	queue := func(tasks []evalTask, targetID, round int) []evalTask {
		reached[targetID] = true
		target := personas[targetID-1]
		target.Shown = true
		return append(tasks, evalTask{persona: target, result: &results[targetID-1], seed: r.Int63(), round: round})
	}

	// Round 2: each sharer reaches 1–4 people (more with more followers), chosen by
	// homophily — people tend to share with others who have similar interests.
	round2 := []evalTask{}
	for _, sharer := range results {
		if !sharer.Shown || sharer.Action != "share" {
			continue
		}
		sp := personas[sharer.PersonaID-1]
		reach := clampInt(1+sp.Followers/2500, 1, 4)
		for k := 0; k < reach; k++ {
			candidates, weights := []int{}, []float64{}
			for _, p := range personas {
				if reached[p.ID] {
					continue
				}
				wt := 1 + 2*float64(interestOverlap(sp.Interests, p.Interests))
				if p.InTarget == sp.InTarget {
					wt++
				}
				candidates = append(candidates, p.ID)
				weights = append(weights, wt)
			}
			if len(candidates) == 0 {
				break
			}
			target := candidates[weightedPick(weights, r)]
			edges = append(edges, EdgeResult{From: sharer.PersonaID, To: target, Type: "direct", Round: 2})
			round2 = queue(round2, target, 2)
		}
	}
	w.evaluateParallel(sc, round2)

	// Round 3: push budget scales with weighted engagement so far; recipients are the
	// unreached users whose interests best match the content (plus some exploration).
	b, reachedCount := ActionBreakdown{}, 0
	engagers := []PersonaResult{}
	for _, res := range results {
		if !res.Shown {
			continue
		}
		reachedCount++
		switch res.Action {
		case "share":
			b.Shares++
		case "comment":
			b.Comments++
		case "like":
			b.Likes++
		default:
			continue
		}
		engagers = append(engagers, res)
	}
	pushRate := 0.0
	if reachedCount > 0 && len(engagers) > 0 {
		weighted := float64(3*b.Shares+2*b.Comments+b.Likes) / float64(3*reachedCount)
		pushRate = clampF((weighted-pushEngagementFloor)*pushSlope, 0, maxPushRate)
	}

	type candidate struct {
		id    int
		score float64
	}
	candidates := []candidate{}
	for _, p := range personas {
		if !reached[p.ID] {
			candidates = append(candidates, candidate{p.ID, relevance(p, sc.content) + 0.6*r.Float64()})
		}
	}
	sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].score > candidates[j].score })
	budget := int(math.Round(pushRate * float64(len(candidates))))

	round3 := []evalTask{}
	for _, c := range candidates[:budget] {
		target := personas[c.id-1]
		// Attribute the push (for the graph) to a random engager who shares an interest
		// with the recipient, or any engager if none does.
		similar := []int{}
		for _, e := range engagers {
			if interestOverlap(e.Interests, target.Interests) > 0 {
				similar = append(similar, e.PersonaID)
			}
		}
		from := engagers[r.Intn(len(engagers))].PersonaID
		if len(similar) > 0 {
			from = similar[r.Intn(len(similar))]
		}
		edges = append(edges, EdgeResult{From: from, To: c.id, Type: "algo", Round: 3})
		round3 = queue(round3, c.id, 3)
	}
	w.evaluateParallel(sc, round3)

	for i := range results {
		if !reached[results[i].PersonaID] {
			results[i].Shown = false
			results[i].Action = "never"
			results[i].Round = 0
			results[i].Watch = 0
			results[i].Why = "Not reached by network share or algorithm."
		}
	}
	return edges, pushRate
}

func weightedPick(weights []float64, r *rand.Rand) int {
	total := 0.0
	for _, w := range weights {
		total += w
	}
	roll := r.Float64() * total
	for i, w := range weights {
		if roll < w {
			return i
		}
		roll -= w
	}
	return len(weights) - 1
}

func pct(n, d int) int {
	if d == 0 {
		return 0
	}
	return n * 100 / d
}

func aggregateResults(sc *simContext, results []PersonaResult, seedCount int, pushRate float64) *SimulationResult {
	breakdown := ActionBreakdown{}
	var reached, inReached, inEngaged, outReached, outEngaged, seedSkips, llmCount int
	var relevantSeeds, relevantSeedSkips int

	for _, r := range results {
		if !r.Shown {
			continue
		}
		reached++
		if r.llm {
			llmCount++
		}
		relevantSeed := r.Round == 1 && interestRelevance(r.Interests, sc.content) >= 0.5
		if relevantSeed {
			relevantSeeds++
		}
		engaged := true
		switch r.Action {
		case "like":
			breakdown.Likes++
		case "share":
			breakdown.Shares++
		case "comment":
			breakdown.Comments++
		default:
			breakdown.Scrolls++
			engaged = false
			// Only a quick swipe counts against the hook; someone who watched most of it
			// and moved on was hooked — the payoff, not the opening, lost them.
			if r.Round == 1 && r.Watch < swipeWatch {
				seedSkips++
				if relevantSeed {
					relevantSeedSkips++
				}
			}
		}
		if r.InTarget {
			inReached++
			if engaged {
				inEngaged++
			}
		} else {
			outReached++
			if engaged {
				outEngaged++
			}
		}
	}

	// Virality = 60% engagement quality (shares 3×, comments 2×, likes 1×; 0.4 counts
	// as excellent) + 40% how far it spread beyond the seed audience.
	quality := 0.0
	if reached > 0 {
		quality = float64(3*breakdown.Shares+2*breakdown.Comments+breakdown.Likes) / float64(3*reached)
	}
	spread := 0.0
	if pop := len(results); pop > seedCount {
		spread = clampF(float64(reached-seedCount)/float64(pop-seedCount), 0, 1)
	}
	virality := int(math.Round(100 * (0.6*math.Min(1, quality/0.4) + 0.4*spread)))

	amplification := 0.0
	if seedCount > 0 {
		amplification = math.Round(float64(reached)/float64(seedCount)*10) / 10
	}

	a := Analysis{
		AudienceTopics:      nonNil(sc.audience.Topics),
		AudienceAge:         sc.audience.AgeNote,
		AudienceRegions:     nonNil(sc.audience.Regions),
		ContentTopics:       nonNil(sc.content.Topics),
		AudienceFit:         AudienceFit(sc.audience, sc.content),
		HookScore:           int(math.Round(sc.content.HookScore * 100)),
		HookSignals:         nonNil(sc.content.Signals),
		HookSkipRate:        pct(seedSkips, seedCount),
		RelevantSkipRate:    -1,
		InTargetEngagement:  pct(inEngaged, inReached),
		OutTargetEngagement: pct(outEngaged, outReached),
		Amplification:       amplification,
		Engine:              "heuristic",
		VideoHook:           sc.video.Hook(),
		VideoRest:           sc.video.Rest,
		VideoNote:           sc.video.Note,
		TopicSource:         sc.topicSrc,
		HookSource:          sc.hookSrc,
		Niche:               sc.nicheName,
	}
	if relevantSeeds > 0 {
		a.RelevantSkipRate = pct(relevantSeedSkips, relevantSeeds)
	}
	a.Recommendations = nonNil(buildRecommendations(sc.audience, sc.content, a.AudienceFit))
	a.LLMShare = pct(llmCount, reached)
	switch {
	case llmCount == reached && reached > 0:
		a.Engine = "llm"
	case llmCount > 0:
		a.Engine = "mixed"
	}

	shareRate := pct(breakdown.Shares, reached)
	head, sub, verdict := computeVerdict(a, shareRate, virality)

	return &SimulationResult{
		JobID:      sc.job.JobID,
		Status:     "complete",
		Personas:   results,
		Virality:   virality,
		AlgoScore:  int(math.Round(100 * pushRate / maxPushRate)),
		Engagement: pct(reached-breakdown.Scrolls, reached),
		Breakdown:  breakdown,
		Verdict: VerdictData{
			Head:      head,
			Sub:       sub,
			Verdict:   verdict,
			InPct:     pct(inReached, reached),
			ShareRate: shareRate,
		},
		Analysis: a,
	}
}

func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func interestOverlap(a, b []string) int {
	return len(matchedTopics(a, b))
}

// computeVerdict classifies the outcome from the measured cascade.
func computeVerdict(a Analysis, shareRate, virality int) (string, string, string) {
	// A hook is "weak" when people the topic is relevant to still scroll past.
	// If nobody relevant saw it, fall back to the overall seed skip rate.
	hookSkip := a.RelevantSkipRate
	if hookSkip < 0 {
		hookSkip = a.HookSkipRate
	}
	switch {
	case a.AudienceFit == 0 && len(a.ContentTopics) > 0 && a.InTargetEngagement < 25:
		return "Audience mismatch.", fmt.Sprintf("The hook is about %s, but your target audience cares about %s — only %d%% of them engaged.",
			strings.Join(a.ContentTopics, ", "), strings.Join(a.AudienceTopics, ", "), a.InTargetEngagement), "Wrong audience"
	case hookSkip >= 60:
		who := "viewers who care about this topic still"
		if a.RelevantSkipRate < 0 {
			who = "the first viewers" // the model couldn't tell who the topic is relevant to
		}
		return "Weak hook.", fmt.Sprintf("%d%% of %s scrolled past in the first 3 seconds. Rework the opening.", hookSkip, who), "Needs rework"
	case virality >= 60 && shareRate >= 10 && a.OutTargetEngagement >= 30:
		return "Viral hit.", fmt.Sprintf("Strong in-demo engagement (%d%%) that also broke out to other audiences (%d%%), reaching %.1f× the seed audience.", a.InTargetEngagement, a.OutTargetEngagement, a.Amplification), "Breakout potential"
	case a.InTargetEngagement >= 40 && shareRate >= 6:
		return "Niche hit.", fmt.Sprintf("Your target audience engaged (%d%%) and shared it, but crossover beyond the niche is limited (%d%%).", a.InTargetEngagement, a.OutTargetEngagement), "Strong in-demo, no breakout"
	case a.InTargetEngagement >= 35:
		return "Echo chamber.", fmt.Sprintf("The target audience liked it (%d%% engaged) but few shared it (%d%%), so it never escaped the seed group.", a.InTargetEngagement, shareRate), "Targeted, not viral"
	}
	return "Weak signal.", fmt.Sprintf("Low engagement from both the target audience (%d%%) and everyone else (%d%%).", a.InTargetEngagement, a.OutTargetEngagement), "Needs rework"
}
