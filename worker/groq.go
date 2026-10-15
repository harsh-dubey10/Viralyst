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

// llmDisabled is set when Groq reports the model doesn't exist; every further call would
// fail the same way, so the worker switches to the offline model instead of retrying.
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
	keyCounter uint64
	keysOnce   sync.Once
)

// loadGroqAPIKeys discovers all configured Groq API keys:
// GROQ_API_KEY, GROQ_API_KEY_1, GROQ_API_KEY_2, ... and GROQ_API_KEYS (comma separated).
func loadGroqAPIKeys() []string {
	keysOnce.Do(func() {
		var discovered []string

		// 1. Single default key
		if k := strings.TrimSpace(os.Getenv("GROQ_API_KEY")); k != "" {
			discovered = append(discovered, k)
		}

		// 2. Numbered keys GROQ_API_KEY_1 ... GROQ_API_KEY_10
		for i := 1; i <= 10; i++ {
			envName := fmt.Sprintf("GROQ_API_KEY_%d", i)
			if k := strings.TrimSpace(os.Getenv(envName)); k != "" {
				discovered = append(discovered, k)
			}
		}

		// 3. Comma-separated list
		if kList := os.Getenv("GROQ_API_KEYS"); kList != "" {
			for _, k := range strings.Split(kList, ",") {
				if k = strings.TrimSpace(k); k != "" {
					discovered = append(discovered, k)
				}
			}
		}

		// Deduplicate
		seen := make(map[string]bool)
		for _, k := range discovered {
			if !seen[k] {
				seen[k] = true
				groqKeys = append(groqKeys, k)
			}
		}

		if len(groqKeys) > 0 {
			log.Printf("[groq] Loaded %d Groq API key(s) for alternate rotation & 429 failover", len(groqKeys))
		}
	})
	return groqKeys
}

func hasGroqAPIKeys() bool {
	return len(loadGroqAPIKeys()) > 0 && !llmDisabled.Load()
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

// callGroqChatSingleKey executes one chat completion with a specific key.
// Returns (content, statusCode, retryAfter, error).
func callGroqChatSingleKey(client *http.Client, payload groqRequest, apiKey string) (string, int, time.Duration, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return "", 0, 0, fmt.Errorf("marshal: %w", err)
	}

	req, err := http.NewRequest("POST", groqBaseURL+"/chat/completions", bytes.NewReader(body))
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
		return "", statusCode, 0, fmt.Errorf("empty choices from groq")
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

// withGroqKeys runs one request through the configured keys (round-robin), failing over
// on errors, and backs off and retries when all keys are rate limited. A 404 (unknown
// model) returns at once, since every key would fail the same way. It returns the last
// status code seen so callers can react to it.
func withGroqKeys(do func(apiKey string) (int, time.Duration, error)) (int, error) {
	keys := loadGroqAPIKeys()
	if len(keys) == 0 {
		return 0, fmt.Errorf("no GROQ_API_KEY configured")
	}

	startIdx := int(atomic.AddUint64(&keyCounter, 1)-1) % len(keys)
	var lastErr error
	lastStatus := 0
	waited := time.Duration(0)

	for {
		rateLimited, wait := false, time.Duration(0)
		for attempt := 0; attempt < len(keys); attempt++ {
			currentIdx := (startIdx + attempt) % len(keys)
			statusCode, retryAfter, err := do(keys[currentIdx])
			if err == nil {
				return statusCode, nil
			}
			lastErr, lastStatus = err, statusCode

			switch statusCode {
			case http.StatusTooManyRequests:
				rateLimited = true
				if retryAfter > wait {
					wait = retryAfter
				}
			case http.StatusNotFound:
				return statusCode, err
			default:
				if len(keys) > 1 {
					log.Printf("[groq] Key #%d error (%v). Trying next key...", currentIdx+1, err)
				}
			}
		}

		if !rateLimited || waited >= maxRateLimitWait {
			return lastStatus, fmt.Errorf("all %d Groq API key(s) failed: %w", len(keys), lastErr)
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

// callGroqChat runs one chat completion with the persona model. If Groq reports the
// model doesn't exist, it switches the worker to the offline model.
func callGroqChat(payload groqRequest) (string, error) {
	var content string
	statusCode, err := withGroqKeys(func(apiKey string) (int, time.Duration, error) {
		c, statusCode, retryAfter, err := callGroqChatSingleKey(httpClient, payload, apiKey)
		content = c
		return statusCode, retryAfter, err
	})
	if statusCode == http.StatusNotFound {
		llmDisabledOnce.Do(func() {
			llmDisabled.Store(true)
			log.Printf("[groq] ERROR: model %q not available (%v). Set GROQ_MODEL to a model from "+
				"https://api.groq.com/openai/v1/models — using the offline model until restart.", payload.Model, err)
		})
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

	// Try Groq first with multi-key load balancing
	if hasGroqAPIKeys() {
		result, err := evaluateWithGroq(persona, sc, r)
		if err == nil {
			return result
		}
		log.Printf("[groq] Persona %d fallback — %v", persona.ID, err)
	}

	return fallbackResult(persona, sc, r)
}

// evaluateWithGroq asks the LLM how this persona reacts, then samples the action through
// the same calibrated reaction model as the offline engine. Single LLM personas judge
// relevance well but post quality poorly (depending on the prompt they either dismissed
// a real 3.4M-view hit or rated a bland pasta post a hit), so:
//   - relevance comes from the persona ("is this about something I care about?"),
//   - hook strength from the one comparative hook rating for the whole post,
//   - base rates from reaction(), shared with the offline engine,
//   - and the persona's own like/comment/share scores tilt its odds.
func evaluateWithGroq(persona PersonaProfile, sc *simContext, r *rand.Rand) (PersonaResult, error) {
	raw, err := callGroq(buildPrompt(persona, sc.job, sc.video.Rest))
	if err != nil {
		return PersonaResult{}, err
	}

	// Extract JSON substring robustly (model may wrap in markdown)
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start == -1 || end == -1 || end <= start {
		return PersonaResult{}, fmt.Errorf("no JSON in response: %q", raw)
	}
	var d groqPersonaDecision
	if err := json.Unmarshal([]byte(raw[start:end+1]), &d); err != nil {
		return PersonaResult{}, fmt.Errorf("parse decision: %w", err)
	}
	d.Watch = clampInt(d.Watch, 0, 100)
	d.Like = clampInt(d.Like, 0, 10)
	d.Comment = clampInt(d.Comment, 0, 10)
	d.Share = clampInt(d.Share, 0, 10)
	if r == nil {
		r = rand.New(rand.NewSource(int64(persona.ID)))
	}

	action, watch := reaction(persona, sc.content, llmRelevance(d), llmTilt(d), r)
	why := strings.TrimSpace(d.Why)
	if implied := llmImpliedAction(d); action == implied {
		// The sampled outcome matches what the persona said: keep their words and watch %.
		if (d.Watch < swipeWatch) == (watch < swipeWatch) {
			watch = d.Watch
		}
	} else {
		// The persona's sentence would contradict the outcome; describe the outcome instead.
		why = whyFor(action, watch, persona, sc.content, r)
	}
	if why == "" {
		why = whyFor(action, watch, persona, sc.content, r)
	}

	return PersonaResult{
		PersonaID: persona.ID,
		Name:      persona.Name,
		Age:       persona.Age,
		Location:  persona.Location,
		Shown:     true,
		Followers: persona.Followers,
		Gender:    persona.Gender,
		Behavior:  persona.Behavior,
		Interests: persona.Interests,
		InTarget:  persona.InTarget,
		Action:    action,
		Watch:     clampInt(watch, 0, 100),
		Why:       why,
		Round:     1,
		llm:       true,
	}, nil
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

func topicOr(matched []string, fallback string) string {
	if len(matched) > 0 {
		return matched[0]
	}
	return fallback
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

func shareWhy(p PersonaProfile, matched []string, r *rand.Rand) string {
	t := topicOr(matched, "my feed")
	return pick(r,
		fmt.Sprintf("This is exactly what my %s circle needs. Sent it on.", t),
		fmt.Sprintf("Rare %s post with real substance — reposted for my %d followers.", t, p.Followers),
		fmt.Sprintf("Forwarded to two friends deep into %s.", t),
	)
}

func commentWhy(p PersonaProfile, c ContentProfile, matched []string, r *rand.Rand) string {
	t := topicOr(matched, "this")
	if c.Debate {
		return pick(r,
			fmt.Sprintf("Strong claim about %s — I left a comment pushing back.", t),
			"It asked a real question, so I answered in the comments.",
		)
	}
	if p.Behavior == "critic" || p.Behavior == "skeptic" || p.Behavior == "analytical thinker" {
		return fmt.Sprintf("Interesting %s take, but I want to see the data. Commented.", t)
	}
	return fmt.Sprintf("Asked a follow-up about the %s details in the comments.", t)
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
