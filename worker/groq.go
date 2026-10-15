package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ──────────────────────────────────────────────
// Groq API types (OpenAI-compatible format)
// ──────────────────────────────────────────────

// Content is a string, or a list of text/image parts for vision models.
type groqMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type groqRequest struct {
	Model           string            `json:"model"`
	Messages        []groqMessage     `json:"messages"`
	MaxTokens       int               `json:"max_tokens"`
	Temperature     float64           `json:"temperature"`
	ReasoningEffort string            `json:"reasoning_effort,omitempty"`
	ResponseFormat  map[string]string `json:"response_format,omitempty"`
}

// defaultGroqModel must be a model Groq still serves (llama-3.1-8b-instant was retired).
// Override with GROQ_MODEL; list options with GET https://api.groq.com/openai/v1/models.
const defaultGroqModel = "openai/gpt-oss-20b"

func groqModel() string {
	if m := strings.TrimSpace(os.Getenv("GROQ_MODEL")); m != "" {
		return m
	}
	return defaultGroqModel
}

// llmDisabled is set when every provider reports its model doesn't exist; every further
// call would fail the same way, so the worker switches to the offline model.
var (
	llmDisabled     atomic.Bool
	llmDisabledOnce sync.Once
)

type groqResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// groqPersonaDecision is a persona's reaction as independent 0–10 intent scores rather
// than one chosen action: asked to pick a single action, the model collapses onto the
// safe "like" and almost never shares, so no post can cascade. The action is sampled
// from these scores (see decideAction).
type groqPersonaDecision struct {
	Relevant *int   `json:"relevant"` // 0-10 how much the post is about something they care about
	Watch    int    `json:"watch"`    // 0-100 % of the video watched
	Like     int    `json:"like"`     // 0-10 how likely they'd like it
	Comment  int    `json:"comment"`  // 0-10 how likely they'd comment
	Share    int    `json:"share"`    // 0-10 how likely they'd share / send it to someone
	Why      string `json:"why"`
}

// ──────────────────────────────────────────────
// Multi-Key Load Balancer & HTTP client
// ──────────────────────────────────────────────

var (
	httpClient = &http.Client{Timeout: 15 * time.Second}
	groqKeys   []string
	keysOnce   sync.Once
)

// loadGroqAPIKeys discovers all configured Groq API keys:
// GROQ_API_KEY, GROQ_API_KEY_1, GROQ_API_KEY_2, ... and GROQ_API_KEYS (comma separated).
func loadGroqAPIKeys() []string {
	keysOnce.Do(func() {
		groqKeys = discoverKeys("GROQ")
		if len(groqKeys) > 0 {
			log.Printf("[groq] Loaded %d Groq API key(s) for alternate rotation & 429 failover", len(groqKeys))
		}
	})
	return groqKeys
}

// hasLLMKeys reports whether any chat endpoint (Groq or Gemini) is usable.
func hasLLMKeys() bool {
	return len(chatEndpoints()) > 0 && !llmDisabled.Load()
}

// llmEndpoint is one API key on one OpenAI-compatible provider, with the model to use there.
type llmEndpoint struct {
	provider  string // "groq", "gemini" or "anthropic"
	baseURL   string
	key       string
	model     string
	reasoning string // reasoning_effort for this model ("" to omit)
}

// Gemini's OpenAI-compatible API. It rejects reasoning_effort "none"; "low" keeps a
// persona reply around 170 tokens.
var geminiBaseURL = "https://generativelanguage.googleapis.com/v1beta/openai"

// defaultGeminiModel is a fast, cheap Flash-Lite model; override with GEMINI_MODEL
// (list options with GET {geminiBaseURL}/models).
const defaultGeminiModel = "gemini-3.5-flash-lite"

var (
	geminiKeys     []string
	geminiKeysOnce sync.Once
	groqCounter    uint64 // round-robin position for Groq-only calls (video)
	chatCounter    uint64 // round-robin position across all chat endpoints
	// providers whose model turned out not to exist; they leave the rotation until restart
	disabledProviders sync.Map
)

// loadGeminiAPIKeys discovers GEMINI_API_KEY, GEMINI_API_KEY_1..10 and GEMINI_API_KEYS.
func loadGeminiAPIKeys() []string {
	geminiKeysOnce.Do(func() {
		geminiKeys = discoverKeys("GEMINI")
		if len(geminiKeys) > 0 {
			log.Printf("[gemini] Loaded %d Gemini API key(s) into the LLM rotation", len(geminiKeys))
		}
	})
	return geminiKeys
}

// discoverKeys reads PREFIX_API_KEY, PREFIX_API_KEY_1..10 and comma-separated
// PREFIX_API_KEYS, without duplicates.
func discoverKeys(prefix string) []string {
	var found []string
	add := func(k string) {
		if k = strings.TrimSpace(k); k != "" {
			found = append(found, k)
		}
	}
	add(os.Getenv(prefix + "_API_KEY"))
	for i := 1; i <= 10; i++ {
		add(os.Getenv(fmt.Sprintf("%s_API_KEY_%d", prefix, i)))
	}
	for _, k := range strings.Split(os.Getenv(prefix+"_API_KEYS"), ",") {
		add(k)
	}
	out, seen := []string{}, map[string]bool{}
	for _, k := range found {
		if !seen[k] {
			seen[k] = true
			out = append(out, k)
		}
	}
	return out
}

func groqEndpoints() []llmEndpoint {
	model := groqModel()
	reasoning := ""
	if strings.HasPrefix(model, "openai/gpt-oss") || strings.HasPrefix(model, "qwen/") {
		reasoning = "low"
	}
	eps := []llmEndpoint{}
	for _, k := range loadGroqAPIKeys() {
		eps = append(eps, llmEndpoint{provider: "groq", baseURL: groqBaseURL, key: k, model: model, reasoning: reasoning})
	}
	return eps
}

// chatEndpoints is every key of every provider still in the rotation, interleaved by
// provider (groq, gemini, anthropic, groq, …) so consecutive calls spread across
// providers' quotas.
func chatEndpoints() []llmEndpoint {
	enabled := func(provider string) bool {
		_, off := disabledProviders.Load(provider)
		return !off
	}
	var lists [][]llmEndpoint
	if enabled("groq") {
		lists = append(lists, groqEndpoints())
	}
	if enabled("gemini") {
		model := envOr("GEMINI_MODEL", defaultGeminiModel)
		var eps []llmEndpoint
		for _, k := range loadGeminiAPIKeys() {
			eps = append(eps, llmEndpoint{provider: "gemini", baseURL: geminiBaseURL, key: k, model: model, reasoning: "low"})
		}
		lists = append(lists, eps)
	}
	if enabled("anthropic") {
		var eps []llmEndpoint
		for _, k := range loadAnthropicAPIKeys() {
			eps = append(eps, llmEndpoint{provider: "anthropic", key: k, model: anthropicModel()})
		}
		lists = append(lists, eps)
	}
	out := []llmEndpoint{}
	for i := 0; ; i++ {
		added := false
		for _, l := range lists {
			if i < len(l) {
				out = append(out, l[i])
				added = true
			}
		}
		if !added {
			return out
		}
	}
}

// callCounts tallies successful chat calls per key ("groq #2"), so each job can log how
// the load spread across the rotation.
var callCounts sync.Map // label → *uint64

func countCall(ep llmEndpoint) {
	keys := loadGroqAPIKeys()
	switch ep.provider {
	case "gemini":
		keys = loadGeminiAPIKeys()
	case "anthropic":
		keys = loadAnthropicAPIKeys()
	}
	label := ep.provider
	for i, k := range keys {
		if k == ep.key {
			label = fmt.Sprintf("%s #%d", ep.provider, i+1)
		}
	}
	n, _ := callCounts.LoadOrStore(label, new(uint64))
	atomic.AddUint64(n.(*uint64), 1)
}

// llmCallCounts is a snapshot of callCounts.
func llmCallCounts() map[string]uint64 {
	out := map[string]uint64{}
	callCounts.Range(func(k, v any) bool {
		out[k.(string)] = atomic.LoadUint64(v.(*uint64))
		return true
	})
	return out
}

// disableProvider takes a provider whose model doesn't exist out of the rotation; when
// none are left, persona decisions switch to the offline model until restart.
func disableProvider(provider string, err error) {
	if _, already := disabledProviders.LoadOrStore(provider, true); already {
		return
	}
	hint := "GROQ_MODEL (see https://api.groq.com/openai/v1/models)"
	switch provider {
	case "gemini":
		hint = "GEMINI_MODEL (see " + geminiBaseURL + "/models)"
	case "anthropic":
		hint = "ANTHROPIC_API_KEY, ANTHROPIC_WORKSPACE_ID (for organization-level keys) and ANTHROPIC_MODEL (e.g. " + defaultAnthropicModel + ")"
	}
	log.Printf("[llm] ERROR: %s can't be used (%v). Check %s — leaving %s out of the rotation until restart.", provider, err, hint, provider)
	if len(chatEndpoints()) == 0 {
		llmDisabledOnce.Do(func() {
			llmDisabled.Store(true)
			log.Printf("[llm] No LLM provider left — using the offline model until restart.")
		})
	}
}

// The persona model was told "never be polite" and that most people skip most posts;
// it then dismissed even posts squarely in a persona's interests as "generic" (a real
// 3.4M-view hit scored 3/100, below a bland pasta post). Fans do engage with decent
// posts in their niche, so the prompt says so while still demanding honest skips.
const groqSystemPrompt = "You simulate one real social media user scrolling their feed. Be honest, not polite: you skip what isn't for you. But you're a real fan of your interests: when a post is about one of them and isn't bad, you watch and often like it, the way fans actually do. Respond with one JSON object only."

// groqBaseURL is a variable so tests can point it at a local server.
var groqBaseURL = "https://api.groq.com/openai/v1"

// doGroqRequest sends one request with a specific key and returns the response body.
// Returns (body, statusCode, retryAfter, error); retryAfter is set on 429s.
func doGroqRequest(client *http.Client, req *http.Request, apiKey string) ([]byte, int, time.Duration, error) {
	req.Header.Set("Authorization", "Bearer "+apiKey)
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		var retryAfter time.Duration
		if secs, err := strconv.ParseFloat(resp.Header.Get("Retry-After"), 64); err == nil {
			retryAfter = time.Duration(secs * float64(time.Second))
		}
		return nil, resp.StatusCode, retryAfter, fmt.Errorf("groq status %d: %s", resp.StatusCode, string(b))
	}
	if err != nil {
		return nil, resp.StatusCode, 0, fmt.Errorf("read: %w", err)
	}
	return b, http.StatusOK, 0, nil
}

// callGroqChatSingleKey executes one chat completion on Groq with a specific key.
// Returns (content, statusCode, retryAfter, error).
func callGroqChatSingleKey(client *http.Client, payload groqRequest, apiKey string) (string, int, time.Duration, error) {
	return callChatSingle(client, groqBaseURL, payload, apiKey)
}

// callChatSingle executes one chat completion on any OpenAI-compatible endpoint.
func callChatSingle(client *http.Client, baseURL string, payload groqRequest, apiKey string) (string, int, time.Duration, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return "", 0, 0, fmt.Errorf("marshal: %w", err)
	}

	req, err := http.NewRequest("POST", baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", 0, 0, fmt.Errorf("request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	b, statusCode, retryAfter, err := doGroqRequest(client, req, apiKey)
	if err != nil {
		return "", statusCode, retryAfter, err
	}

	var gr groqResponse
	if err := json.Unmarshal(b, &gr); err != nil {
		return "", statusCode, 0, fmt.Errorf("decode: %w", err)
	}
	if len(gr.Choices) == 0 {
		return "", statusCode, 0, fmt.Errorf("empty choices in reply")
	}
	return gr.Choices[0].Message.Content, http.StatusOK, 0, nil
}

// Rate-limit handling: when every key is rate limited, wait as long as Groq asks
// (Retry-After) and try again, up to maxRateLimitWait in total per call.
const (
	maxRateLimitWait    = 90 * time.Second
	defaultRetryAfter   = 2 * time.Second
	maxSingleRetrySleep = 20 * time.Second
)

// withGroqKeys runs one request through the Groq keys only (video transcription and
// vision use Groq-specific models). See withEndpoints.
func withGroqKeys(do func(apiKey string) (int, time.Duration, error)) (int, error) {
	eps := groqEndpoints()
	if len(eps) == 0 {
		return 0, fmt.Errorf("no GROQ_API_KEY configured")
	}
	status, _, err := withEndpoints(eps, &groqCounter, func(ep llmEndpoint) (int, time.Duration, error) { return do(ep.key) })
	return status, err
}

// keyCooldowns holds, per API key, when it may be used again after a 429 (Groq's daily
// token limit can lock a key for minutes; retrying it first on every call wastes a round
// trip each time).
var keyCooldowns sync.Map // key → time.Time

func coolDown(key string, retryAfter time.Duration) {
	if retryAfter <= 0 {
		retryAfter = defaultRetryAfter
	}
	keyCooldowns.Store(key, time.Now().Add(retryAfter))
}

func coolingDown(key string) bool {
	until, ok := keyCooldowns.Load(key)
	return ok && time.Now().Before(until.(time.Time))
}

// withEndpoints runs one request through the endpoints round-robin (counter advances once
// per call), failing over on errors, and backs off and retries when all are rate limited.
// A 404 (unknown model) skips that provider's other keys for this call — they would fail
// the same way — and is reported in notFound so the caller can stop using the provider.
// It returns the last status code seen.
func withEndpoints(eps []llmEndpoint, counter *uint64, do func(ep llmEndpoint) (int, time.Duration, error)) (int, map[string]error, error) {
	startIdx := int(atomic.AddUint64(counter, 1)-1) % len(eps)
	notFound := map[string]error{}
	var lastErr error
	lastStatus := 0
	waited := time.Duration(0)

	for {
		rateLimited, wait := false, time.Duration(0)
		// Keys cooling down after a 429 are skipped while any other key is available.
		anyReady := false
		for _, ep := range eps {
			if notFound[ep.provider] == nil && !coolingDown(ep.key) {
				anyReady = true
			}
		}
		for attempt := 0; attempt < len(eps); attempt++ {
			ep := eps[(startIdx+attempt)%len(eps)]
			if notFound[ep.provider] != nil || (anyReady && coolingDown(ep.key)) {
				continue
			}
			statusCode, retryAfter, err := do(ep)
			if err == nil {
				return statusCode, notFound, nil
			}
			lastErr, lastStatus = err, statusCode

			switch statusCode {
			case http.StatusTooManyRequests:
				rateLimited = true
				coolDown(ep.key, retryAfter)
				if retryAfter > wait {
					wait = retryAfter
				}
			case http.StatusNotFound:
				notFound[ep.provider] = err
			default:
				if len(eps) > 1 {
					log.Printf("[llm] %s key error (%v). Trying next key...", ep.provider, err)
				}
			}
		}

		if !rateLimited || waited >= maxRateLimitWait {
			return lastStatus, notFound, fmt.Errorf("all %d LLM key(s) failed: %w", len(eps), lastErr)
		}
		if wait <= 0 {
			wait = defaultRetryAfter
		}
		if wait > maxSingleRetrySleep {
			wait = maxSingleRetrySleep
		}
		time.Sleep(wait)
		waited += wait
	}
}

// callGroq runs one persona-decision chat completion.
func callGroq(prompt string) (string, error) {
	model := groqModel()
	payload := groqRequest{
		Model: model,
		Messages: []groqMessage{
			{Role: "system", Content: groqSystemPrompt},
			{Role: "user", Content: prompt},
		},
		// Groq counts max_tokens against the tokens-per-minute quota, so keep it tight;
		// low-effort reasoning plus the JSON reply uses well under this.
		MaxTokens:      220,
		Temperature:    0.85,
		ResponseFormat: map[string]string{"type": "json_object"},
	}
	if strings.HasPrefix(model, "openai/gpt-oss") || strings.HasPrefix(model, "qwen/") {
		payload.ReasoningEffort = "low"
	}

	return callGroqChat(payload)
}

// callGroqChat runs one chat completion (persona decisions, topic reading, hook rating)
// on the next endpoint in the Groq + Gemini rotation, with that provider's model. If a
// provider reports its model doesn't exist, it leaves the rotation; with none left the
// worker switches to the offline model.
func callGroqChat(payload groqRequest) (string, error) {
	eps := chatEndpoints()
	if len(eps) == 0 {
		return "", fmt.Errorf("no LLM API key configured")
	}
	var content string
	_, notFound, err := withEndpoints(eps, &chatCounter, func(ep llmEndpoint) (int, time.Duration, error) {
		p := payload
		p.Model, p.ReasoningEffort = ep.model, ep.reasoning
		var c string
		var statusCode int
		var retryAfter time.Duration
		var err error
		if ep.provider == "anthropic" {
			c, statusCode, retryAfter, err = callAnthropicSingle(p, ep)
		} else {
			c, statusCode, retryAfter, err = callChatSingle(httpClient, ep.baseURL, p, ep.key)
		}
		content = c
		if err == nil {
			countCall(ep)
		}
		return statusCode, retryAfter, err
	})
	for provider, perr := range notFound {
		disableProvider(provider, perr)
	}
	return content, err
}

// buildPrompt describes the persona and the post. It deliberately does not say whether
// the persona is "in the target audience" — that would nudge the model to engage.
// The persona judges the post only from who they are.
// rest, if non-empty, summarises what happens after the opening: the opening decides
// whether the persona keeps watching, the whole video whether they engage.
func buildPrompt(persona PersonaProfile, job SimulationJob, rest string) string {
	restText := ""
	if rest != "" {
		restText = fmt.Sprintf("\n\nIf you keep watching, the rest of the video: %s\n(Your watch %% depends on the opening; like / comment / share depend on the whole video, if you'd stay for it.)", rest)
	}
	return fmt.Sprintf(
		`You are %s, %d, %s, from %s, %d followers. Interests: %s. Personality: %s.

A post appears in your feed:
"%s"%s

React honestly as yourself. A post only needs to be about ONE of your interests to be for you — it doesn't have to cover them all.
- relevant: 0-10, how much this post is about something you care about (0 = nothing to do with your interests, 5 = loosely related, 10 = exactly your thing). Judge the topic, not how good the post is.
- watch: %% of the video you'd watch.
  0-20 = you swipe away in the first 3 seconds (not for you, or the opening loses you instantly).
  25-50 = it's for you and held you a while, then you moved on.
  50-80 = solid, you watched most of it. 80-100 = gripping, you watched it all.
- like / comment / share: 0-10 each, how likely you'd do it (0 = never, 5 = maybe, 10 = certainly). Score each independently.
- Fans like decent posts in their niche fairly often (5-7); save 8+ for posts they really enjoyed.
- People share posts that are useful, funny, relatable or surprising to send to a friend, a group chat or their followers — not only big accounts. Share 6+ if someone in your life should see it.
- Lurkers rarely comment; skeptics and critics rarely share but may comment to push back.
- If it's not about any of your interests: watch 0-20 and all three 0-2.

JSON: {"relevant":<0-10>,"watch":<0-100>,"like":<0-10>,"comment":<0-10>,"share":<0-10>,"why":"<one honest sentence in your voice>"}`,
		persona.Name, persona.Age, persona.Gender, persona.Location, persona.Followers,
		strings.Join(persona.Interests, ", "), persona.Behavior, getContentText(job), restText,
	)
}

// ──────────────────────────────────────────────
// EvaluatePersona — try Groq, fall back to heuristic
// ──────────────────────────────────────────────

func EvaluatePersona(persona PersonaProfile, sc *simContext, r *rand.Rand) PersonaResult {
	if !persona.Shown {
		return PersonaResult{
			PersonaID: persona.ID,
			Name:      persona.Name,
			Age:       persona.Age,
			Location:  persona.Location,
			Followers: persona.Followers,
			Gender:    persona.Gender,
			Behavior:  persona.Behavior,
			Interests: persona.Interests,
			InTarget:  persona.InTarget,
			Shown:     false,
			Action:    "never",
			Watch:     0,
			Why:       "",
			Round:     0,
		}
	}

	if hasLLMKeys() {
		d, err := askPersona(persona, sc)
		if err == nil {
			return assignByIntent(sc, []llmItem{{persona: persona, d: d, seed: r.Int63()}})[0]
		}
		log.Printf("[llm] Persona %d fallback — %v", persona.ID, err)
	}

	return fallbackResult(persona, sc, r)
}

// askPersona asks the LLM how this persona reacts to the post.
func askPersona(persona PersonaProfile, sc *simContext) (groqPersonaDecision, error) {
	raw, err := callGroq(buildPrompt(persona, sc.job, sc.video.Rest))
	if err != nil {
		return groqPersonaDecision{}, err
	}
	// Extract JSON substring robustly (model may wrap in markdown)
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start == -1 || end == -1 || end <= start {
		return groqPersonaDecision{}, fmt.Errorf("no JSON in response: %q", raw)
	}
	var d groqPersonaDecision
	if err := json.Unmarshal([]byte(raw[start:end+1]), &d); err != nil {
		return groqPersonaDecision{}, fmt.Errorf("parse decision: %w", err)
	}
	d.Watch = clampInt(d.Watch, 0, 100)
	d.Like = clampInt(d.Like, 0, 10)
	d.Comment = clampInt(d.Comment, 0, 10)
	d.Share = clampInt(d.Share, 0, 10)
	if d.Relevant != nil {
		v := clampInt(*d.Relevant, 0, 10)
		d.Relevant = &v
	}
	d.Why = strings.TrimSpace(d.Why)
	return d, nil
}

// llmItem is one persona the LLM answered for, with its RNG seed.
type llmItem struct {
	persona PersonaProfile
	d       groqPersonaDecision
	seed    int64
}

// Outcome categories: the four actions, with "skip" split into a quick swipe and
// watching without reacting (they mean different things for the hook).
const (
	catShare   = "share"
	catComment = "comment"
	catLike    = "like"
	catSwipe   = "swipe"
	catWatched = "watched"
)

// assignByIntent turns a batch of LLM persona answers into outcomes: the calibrated
// model decides HOW MANY viewers share / comment / like / swipe, the personas' own
// answers decide WHO.
//
// Single LLM personas are poor at absolute levels (depending on the prompt they had a
// real 3.4M-view hit flop, or 57% of viewers "sharing" a so-so post), so using their
// answers directly swings every result to an extreme. Sampling each persona through
// reaction() instead — relevance from the persona, hook strength from the comparative
// rating — gives calibrated totals but ignored what most personas said (only 16% ended
// up doing what they said). Here each persona's calibrated draw only adds to the counts;
// the slots then go to the personas whose own scores fit them best, so the people who
// said they'd share are the sharers and keep their own words.
func assignByIntent(sc *simContext, items []llmItem) []PersonaResult {
	n := len(items)
	rngs := make([]*rand.Rand, n)
	counts := map[string]int{}
	for i, it := range items {
		rngs[i] = rand.New(rand.NewSource(it.seed))
		action, watch := reaction(it.persona, sc.content, llmRelevance(it.d), llmTilt(it.d), rngs[i])
		counts[category(action, watch)]++
	}

	// Fill the slots in order of how much they move the cascade, each from the personas
	// still unassigned, ranked by their own stated intent (stable: ties keep batch order).
	cat := make([]string, n)
	remaining := make([]int, n)
	for i := range remaining {
		remaining[i] = i
	}
	// A slot goes first to personas who said exactly that (so a "like" slot doesn't go
	// to a would-be sharer while someone who said "like" is left over), then by score.
	said := make([]string, n)
	for i, it := range items {
		said[i] = llmImpliedAction(it.d)
		if said[i] == "skip" {
			said[i] = catWatched
			if it.d.Watch < swipeWatch {
				said[i] = catSwipe
			}
		}
	}
	take := func(c string, score func(d groqPersonaDecision) float64) {
		sort.SliceStable(remaining, func(a, b int) bool {
			ia, ib := remaining[a], remaining[b]
			if (said[ia] == c) != (said[ib] == c) {
				return said[ia] == c
			}
			return score(items[ia].d) > score(items[ib].d)
		})
		k := minInt(counts[c], len(remaining))
		for _, i := range remaining[:k] {
			cat[i] = c
		}
		remaining = remaining[k:]
	}
	take(catShare, func(d groqPersonaDecision) float64 {
		return 100*float64(d.Share) + 10*float64(d.Like) + float64(d.Watch)/10
	})
	take(catComment, func(d groqPersonaDecision) float64 { return 100*float64(d.Comment) + float64(d.Watch)/10 })
	take(catLike, func(d groqPersonaDecision) float64 { return 100*float64(d.Like) + float64(d.Watch)/10 })
	take(catSwipe, func(d groqPersonaDecision) float64 { return -float64(d.Watch) - llmRelevance(d) })
	for _, i := range remaining {
		cat[i] = catWatched
	}

	out := make([]PersonaResult, n)
	for i, it := range items {
		p, d, r := it.persona, it.d, rngs[i]
		action := cat[i]
		if action == catSwipe || action == catWatched {
			action = "skip"
		}
		// Keep the persona's own watch % and words when the slot matches what they said.
		watch := d.Watch
		if (cat[i] == catSwipe) != (d.Watch < swipeWatch) {
			watch = categoryWatch(cat[i], p, sc.content, llmRelevance(d), r)
		}
		why := d.Why
		if action != llmImpliedAction(d) || why == "" {
			why = whyFor(action, watch, p, sc.content, r)
		}
		out[i] = PersonaResult{
			PersonaID: p.ID,
			Name:      p.Name,
			Age:       p.Age,
			Location:  p.Location,
			Shown:     true,
			Followers: p.Followers,
			Gender:    p.Gender,
			Behavior:  p.Behavior,
			Interests: p.Interests,
			InTarget:  p.InTarget,
			Action:    action,
			Watch:     clampInt(watch, 0, 100),
			Why:       why,
			Round:     1,
			llm:       true,
		}
	}
	return out
}

func category(action string, watch int) string {
	if action == "skip" {
		if watch < swipeWatch {
			return catSwipe
		}
		return catWatched
	}
	return action
}

// categoryWatch is a watch % typical of an outcome category, as reaction() draws it.
func categoryWatch(cat string, p PersonaProfile, c ContentProfile, rel float64, r *rand.Rand) int {
	v := 0.55*rel + 0.45*c.HookScore
	baseWatch := 30 + int(40*v) + int(behaviorModifiers[p.Behavior][4])
	switch cat {
	case catShare:
		return baseWatch + 25 + r.Intn(20)
	case catComment:
		return baseWatch + 10 + r.Intn(20)
	case catLike:
		return baseWatch + r.Intn(20)
	case catSwipe:
		return 2 + r.Intn(18)
	}
	return swipeWatch + r.Intn(30)
}

// Below this watch % the persona swiped away before anything could land.
const swipeWatch = 25

// llmRelevance is the persona's 0-10 "is this for me" as 0..1. Older replies without it
// fall back to how long they said they'd watch.
func llmRelevance(d groqPersonaDecision) float64 {
	if d.Relevant != nil {
		return clampF(float64(*d.Relevant)/10, 0, 1)
	}
	return clampF(float64(d.Watch-10)/50, 0, 1)
}

// llmTilt scales share/comment/like odds by the persona's own scores (5 = neutral):
// a keen persona is up to ~1.5-1.9× likelier than the base rate, an unmoved one ~0.4-0.5×.
func llmTilt(d groqPersonaDecision) [3]float64 {
	return [3]float64{
		0.4 + 1.5*float64(d.Share)/10,
		0.5 + float64(d.Comment)/10,
		0.5 + float64(d.Like)/10,
	}
}

// llmImpliedAction is what the persona's scores say they'd most likely do.
func llmImpliedAction(d groqPersonaDecision) string {
	switch {
	case d.Watch < swipeWatch:
		return "skip"
	case d.Share >= 6 && d.Share >= d.Like-2:
		return "share"
	case d.Comment >= 6 && d.Comment >= d.Like:
		return "comment"
	case d.Like >= 5:
		return "like"
	}
	return "skip"
}

// ──────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────

func getContentText(job SimulationJob) string {
	if job.ContentDescription != "" {
		return job.ContentDescription
	}
	if job.VideoFilename != "" {
		return "[A video post with no description]"
	}
	return "[No content provided]"
}

// behaviorModifiers maps each behavior type to (shareΔ, commentΔ, likeΔ, skipΔ, watchΔ).
var behaviorModifiers = map[string][5]float64{
	"sharer":                  {+0.08, +0.01, +0.02, -0.05, +10},
	"influencer":              {+0.07, +0.03, +0.03, -0.06, +12},
	"content creator":         {+0.06, +0.03, +0.02, -0.04, +8},
	"commenter":               {-0.01, +0.12, +0.02, -0.04, +5},
	"enthusiast":              {+0.02, +0.03, +0.08, -0.04, +7},
	"trend setter":            {+0.05, +0.02, +0.05, -0.04, +6},
	"trend follower":          {+0.01, +0.01, +0.06, -0.02, +4},
	"early adopter":           {+0.03, +0.02, +0.06, -0.03, +5},
	"networker":               {+0.04, +0.04, +0.03, -0.03, +4},
	"thought leader":          {+0.06, +0.05, +0.02, -0.04, +8},
	"mentor":                  {+0.02, +0.06, +0.04, -0.03, +5},
	"risk taker":              {+0.04, +0.02, +0.03, -0.03, +3},
	"analytical thinker":      {-0.01, +0.06, +0.04, -0.02, +6},
	"creative problem solver": {+0.02, +0.04, +0.05, -0.03, +4},
	"pragmatist":              {+0.00, +0.02, +0.04, -0.01, +2},
	"observer":                {-0.01, +0.01, +0.03, +0.00, +2},
	"critic":                  {-0.02, +0.08, -0.01, +0.02, +3},
	"lurker":                  {-0.04, -0.04, -0.02, +0.12, -8},
	"skeptic":                 {-0.03, +0.05, -0.02, +0.06, -5},
}

// followerShareDelta: bigger accounts amplify good content more readily.
func followerShareDelta(followers int) float64 {
	switch {
	case followers >= 5000:
		return 0.06
	case followers >= 1500:
		return 0.03
	default:
		return 0
	}
}

// fallbackResult is the offline decision model: reaction() with keyword relevance and no
// per-persona tilt, plus a rationale written from the outcome.
func fallbackResult(p PersonaProfile, sc *simContext, r *rand.Rand) PersonaResult {
	action, watch := reaction(p, sc.content, relevance(p, sc.content), [3]float64{1, 1, 1}, r)
	return PersonaResult{
		PersonaID: p.ID,
		Name:      p.Name,
		Age:       p.Age,
		Location:  p.Location,
		Shown:     p.Shown,
		Followers: p.Followers,
		Gender:    p.Gender,
		Behavior:  p.Behavior,
		Interests: p.Interests,
		InTarget:  p.InTarget,
		Action:    action,
		Watch:     clampInt(watch, 0, 100),
		Why:       whyFor(action, watch, p, sc.content, r),
		Round:     1,
	}
}

// reaction samples what a persona does, shared by both engines. It is driven by two
// signals: how relevant the content is to the persona (0..1) and how strong the hook is.
// Behavior archetype and follower count only nudge the odds; tilt scales the share,
// comment and like odds (1 = neutral).
//
//  1. Attention (did they stop scrolling in the first ~3s?)
//     = 0.03 + relevance·(0.15 + 0.75·hook) + 0.12·hook − behavior skip bias
//     A great hook matters most to people who already care about the topic;
//     it barely moves someone with no interest in it.
//  2. If they stopped, value v = 0.55·relevance + 0.45·hook sets the odds of
//     share / comment / like; otherwise they watched a bit and moved on.
func reaction(p PersonaProfile, c ContentProfile, rel float64, tilt [3]float64, r *rand.Rand) (action string, watch int) {
	hook := c.HookScore
	bMod := behaviorModifiers[p.Behavior]

	attention := clampF(0.03+rel*(0.15+0.75*hook)+0.12*hook-bMod[3], 0.02, 0.92)
	if r.Float64() >= attention {
		return "skip", 2 + r.Intn(18)
	}

	v := 0.55*rel + 0.45*hook
	pShare := clampF((0.02+0.24*v*rel+bMod[0]+followerShareDelta(p.Followers)*v)*tilt[0], 0, 0.6)
	debate := 0.0
	if c.Debate {
		debate = 0.08
	}
	pComment := clampF((0.03+0.12*v+debate+bMod[1])*tilt[1], 0, 0.5)
	pLike := clampF((0.15+0.40*v+bMod[2])*tilt[2], 0, 0.8)
	if total := pShare + pComment + pLike; total > 0.95 {
		pShare, pComment, pLike = pShare*0.95/total, pComment*0.95/total, pLike*0.95/total
	}

	baseWatch := 30 + int(40*v) + int(bMod[4])
	switch roll := r.Float64(); {
	case roll < pShare:
		return "share", baseWatch + 25 + r.Intn(20)
	case roll < pShare+pComment:
		return "comment", baseWatch + 10 + r.Intn(20)
	case roll < pShare+pComment+pLike:
		return "like", baseWatch + r.Intn(20)
	}
	return "skip", swipeWatch + r.Intn(30)
}

// whyFor writes a first-person rationale for an outcome from the persona and content.
func whyFor(action string, watch int, p PersonaProfile, c ContentProfile, r *rand.Rand) string {
	matched := matchedTopics(p.Interests, c.Topics)
	switch action {
	case "share":
		return shareWhy(p, matched, r)
	case "comment":
		return commentWhy(p, c, matched, r)
	case "like":
		return likeWhy(p, matched, r)
	}
	if watch < swipeWatch {
		return skipWhy(p, c, r)
	}
	return watchedNoActionWhy(c, matched, r)
}

// ── Context-grounded rationale text ─────────────────────────────────────────

func pick(r *rand.Rand, options ...string) string {
	return options[r.Intn(len(options))]
}

func skipWhy(p PersonaProfile, c ContentProfile, r *rand.Rand) string {
	if len(c.Topics) > 0 && len(matchedTopics(p.Interests, c.Topics)) == 0 {
		return pick(r,
			fmt.Sprintf("I follow %s, not %s. Swiped past in a second.", p.Interests[0], c.Topics[0]),
			fmt.Sprintf("Not into %s — scrolled on.", c.Topics[0]),
		)
	}
	if c.HookScore < 0.35 {
		return pick(r,
			"Nothing in the first line told me why I should care. Gone.",
			"Felt generic — no number, no tension, no reason to stay.",
			"Looked like an ad. Thumb kept moving.",
		)
	}
	return pick(r,
		"Right topic, but I've seen this angle a hundred times. Skipped.",
		"Not in the mood for this right now — scrolled past.",
	)
}

// The rationale templates have a version with the persona's matching interest and one
// without, so a missing topic never leaves a hole ("about the this details").

func shareWhy(p PersonaProfile, matched []string, r *rand.Rand) string {
	if len(matched) == 0 {
		return pick(r,
			"Had to send this to a couple of friends.",
			fmt.Sprintf("Reposted it for my %d followers — too good not to.", p.Followers),
			"Dropped it straight into the group chat.",
		)
	}
	t := matched[0]
	return pick(r,
		fmt.Sprintf("This is exactly what my %s friends need to see. Sent it on.", t),
		fmt.Sprintf("Rare %s post with real substance — reposted for my %d followers.", t, p.Followers),
		fmt.Sprintf("Forwarded to two friends deep into %s.", t),
	)
}

func commentWhy(p PersonaProfile, c ContentProfile, matched []string, r *rand.Rand) string {
	about, take, details := "", "", ""
	if len(matched) > 0 {
		about, take, details = " about "+matched[0], matched[0]+" ", " about the "+matched[0]+" details"
	}
	if c.Debate {
		return pick(r,
			fmt.Sprintf("Strong claim%s — I left a comment pushing back.", about),
			"It asked a real question, so I answered in the comments.",
		)
	}
	if p.Behavior == "critic" || p.Behavior == "skeptic" || p.Behavior == "analytical thinker" {
		return fmt.Sprintf("Interesting %stake, but I want to see the data. Commented.", take)
	}
	return fmt.Sprintf("Asked a follow-up question%s in the comments.", details)
}

func likeWhy(p PersonaProfile, matched []string, r *rand.Rand) string {
	if len(matched) == 0 {
		return "Not my usual topic, but the hook was good enough for a like."
	}
	return pick(r,
		fmt.Sprintf("Solid %s content. Liked and moved on.", matched[0]),
		fmt.Sprintf("Useful for my %s interest — quick like.", matched[0]),
	)
}

func watchedNoActionWhy(c ContentProfile, matched []string, r *rand.Rand) string {
	if len(matched) > 0 {
		return pick(r,
			fmt.Sprintf("Watched most of it — relevant to %s, but didn't feel worth engaging.", matched[0]),
			"Hook got me, the payoff didn't. Watched, no reaction.",
		)
	}
	return "The opening caught my eye, but it's not really for me."
}
