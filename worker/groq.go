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
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ──────────────────────────────────────────────
// Groq API types (OpenAI-compatible format)
// ──────────────────────────────────────────────

type groqMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type groqRequest struct {
	Model       string        `json:"model"`
	Messages    []groqMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens"`
	Temperature float64       `json:"temperature"`
}

type groqResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

type groqPersonaDecision struct {
	Action string `json:"action"` // "like" | "share" | "comment" | "skip"
	Watch  int    `json:"watch"`  // 0-100 % retention
	Why    string `json:"why"`
}

// ──────────────────────────────────────────────
// Multi-Key Load Balancer & HTTP client
// ──────────────────────────────────────────────

var (
	httpClient  = &http.Client{Timeout: 15 * time.Second}
	groqKeys    []string
	keyCounter  uint64
	keysOnce    sync.Once
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
	return len(loadGroqAPIKeys()) > 0
}

// callGroqSingleKey executes a single POST request to Groq API with a specific key.
// Returns (content, statusCode, error).
func callGroqSingleKey(prompt string, apiKey string) (string, int, error) {
	payload := groqRequest{
		Model: "llama-3.1-8b-instant",
		Messages: []groqMessage{
			{
				Role:    "system",
				Content: "You are a ruthless, realistic social media audience simulator. Real users doomscroll quickly with short attention spans and skip 75%-85% of content in under 3 seconds unless it genuinely matches their interests. Never be overly polite or agreeable. Respond with a single raw JSON object only, no markdown, no explanation.",
			},
			{Role: "user", Content: prompt},
		},
		MaxTokens:   140,
		Temperature: 0.85,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", 0, fmt.Errorf("marshal: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.groq.com/openai/v1/chat/completions", bytes.NewBuffer(body))
	if err != nil {
		return "", 0, fmt.Errorf("request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", resp.StatusCode, fmt.Errorf("groq status %d: %s", resp.StatusCode, string(b))
	}

	var gr groqResponse
	if err := json.NewDecoder(resp.Body).Decode(&gr); err != nil {
		return "", resp.StatusCode, fmt.Errorf("decode: %w", err)
	}
	if len(gr.Choices) == 0 {
		return "", resp.StatusCode, fmt.Errorf("empty choices from groq")
	}
	return gr.Choices[0].Message.Content, http.StatusOK, nil
}

// callGroq alternates between available keys using round-robin and fails over to
// other keys in the pool if one hits a 429 rate limit or error.
func callGroq(prompt string) (string, error) {
	keys := loadGroqAPIKeys()
	if len(keys) == 0 {
		return "", fmt.Errorf("no GROQ_API_KEY configured")
	}

	// Pick starting key via atomic round-robin counter
	startIdx := int(atomic.AddUint64(&keyCounter, 1)-1) % len(keys)
	var lastErr error

	// Try all keys in rotation if 429 / error occurs
	for attempt := 0; attempt < len(keys); attempt++ {
		currentIdx := (startIdx + attempt) % len(keys)
		key := keys[currentIdx]

		content, statusCode, err := callGroqSingleKey(prompt, key)
		if err == nil {
			return content, nil
		}

		lastErr = err
		// If hit 429 rate limit, log and try next key
		if statusCode == http.StatusTooManyRequests {
			log.Printf("[groq] Key #%d hit 429 rate limit. Failing over to next key...", currentIdx+1)
			continue
		}

		// On other status codes or network error, also attempt next key if available
		if len(keys) > 1 {
			log.Printf("[groq] Key #%d error (%v). Trying next key...", currentIdx+1, err)
		}
	}

	return "", fmt.Errorf("all %d Groq API key(s) failed: %w", len(keys), lastErr)
}

// buildPrompt creates a persona-evaluation prompt grounded in real social media skepticism.
func buildPrompt(persona PersonaProfile, job SimulationJob) string {
	content := getContentText(job)
	interests := strings.Join(persona.Interests, ", ")
	targetStatus := "in the target niche audience"
	if !persona.InTarget {
		targetStatus = "OUTSIDE the primary target demographic"
	}

	return fmt.Sprintf(
		`You are %s, age %d, %s from %s. Followers: %d.
Your specific interests: %s.
Your behavior profile: %s.
You are %s.

You just saw this social media post / video hook on your feed:
"%s"

Decide your authentic, split-second reaction as a real user scrolling through your feed:
- CRITICAL REALISM: Real users skip 75%%-85%% of posts in < 3 seconds. Do NOT be polite.
- If the content does NOT match your interests (%s), you immediately swipe past ("skip", watch: 5-25).
- If your behavior is "lurker", "skeptic", or "critic", you almost never share. "Lurkers" almost never comment.
- Only "share" (watch: 75-100) if it is genuinely extraordinary, high-signal, or urgent for your network.
- Only "like" (watch: 40-70) if the hook immediately grabbed you and delivered real value.
- Only "comment" (watch: 45-75) if it sparked an active question or debate.
- Otherwise "skip" (watch: 5-30).

Reply ONLY with this valid JSON (no code blocks, no other text):
{"action":"like"|"share"|"comment"|"skip","watch":<0-100 integer>,"why":"<1 sharp sentence in your own voice explaining your honest decision>"}`,
		persona.Name, persona.Age, persona.Gender, persona.Location, persona.Followers,
		interests, persona.Behavior, targetStatus, content, interests,
	)
}

// ──────────────────────────────────────────────
// EvaluatePersona — try Groq, fall back to heuristic
// ──────────────────────────────────────────────

func EvaluatePersona(persona PersonaProfile, job SimulationJob, r *rand.Rand) PersonaResult {
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
		result, err := evaluateWithGroq(persona, job, r)
		if err == nil {
			return result
		}
		log.Printf("[groq] Persona %d fallback — %v", persona.ID, err)
	}

	return fallbackResult(persona, job, r)
}

// evaluateWithGroq calls the LLM and parses its JSON reply.
func evaluateWithGroq(persona PersonaProfile, job SimulationJob, r *rand.Rand) (PersonaResult, error) {
	prompt := buildPrompt(persona, job)
	raw, err := callGroq(prompt)
	if err != nil {
		return PersonaResult{}, err
	}

	// Extract JSON substring robustly (model may wrap in markdown)
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start == -1 || end == -1 || end <= start {
		return PersonaResult{}, fmt.Errorf("no JSON in response: %q", raw)
	}
	jsonStr := raw[start : end+1]

	var decision groqPersonaDecision
	if err := json.Unmarshal([]byte(jsonStr), &decision); err != nil {
		return PersonaResult{}, fmt.Errorf("parse decision: %w", err)
	}

	// Sanitize action
	validActions := map[string]bool{"like": true, "share": true, "comment": true, "skip": true}
	if !validActions[decision.Action] {
		decision.Action = "skip"
	}
	if decision.Watch < 0 {
		decision.Watch = 0
	}
	if decision.Watch > 100 {
		decision.Watch = 100
	}
	if decision.Why == "" {
		decision.Why = "No reasoning provided."
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
		Action:    decision.Action,
		Watch:     decision.Watch,
		Why:       decision.Why,
		Round:     1,
	}, nil
}

// ──────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────

func getContentText(job SimulationJob) string {
	if job.ContentDescription != "" {
		return job.ContentDescription
	}
	if job.VideoFilename != "" {
		return fmt.Sprintf("[Video: %s]", job.VideoFilename)
	}
	return "[No content provided]"
}

func computeContentQuality(job SimulationJob) float64 {
	text := strings.ToLower(getContentText(job))
	if text == "" || text == "[no content provided]" {
		return 0.35
	}

	quality := 0.40

	viralKeywords := []string{
		"secret", "how to", "revenue", "built", "mistake", "framework",
		"growth", "viral", "hacks", "0 to 1", "guide", "system", "case study",
		"ai", "tool", "fast", "transform", "strategy", "breakthrough", "scaling",
		"lesson", "blueprint", "automation", "workflow", "method", "proven",
	}
	for _, word := range viralKeywords {
		if strings.Contains(text, word) {
			quality += 0.08
		}
	}

	weakKeywords := []string{"test", "demo", "sample", "video.mp4", "generic", "foo", "bar"}
	for _, word := range weakKeywords {
		if strings.Contains(text, word) {
			quality -= 0.15
		}
	}

	if len(text) > 40 {
		quality += 0.10
	}
	if quality < 0.15 {
		quality = 0.15
	}
	if quality > 0.88 {
		quality = 0.88
	}
	return quality
}

// behaviorModifiers maps each behavior type to (shareBoost, commentBoost, likeBoost, skipBoost, watchBoost).
// Positive values increase the probability/watch for that action; negative values decrease.
var behaviorModifiers = map[string][5]float64{
	// [shareΔ, commentΔ, likeΔ, skipΔ, watchΔ]
	"sharer":                  {+0.08, +0.01, +0.02, -0.05, +10},
	"influencer":               {+0.07, +0.03, +0.03, -0.06, +12},
	"content creator":          {+0.06, +0.03, +0.02, -0.04, +8},
	"commenter":                {-0.01, +0.12, +0.02, -0.04, +5},
	"enthusiast":               {+0.02, +0.03, +0.08, -0.04, +7},
	"trend setter":             {+0.05, +0.02, +0.05, -0.04, +6},
	"trend follower":           {+0.01, +0.01, +0.06, -0.02, +4},
	"early adopter":            {+0.03, +0.02, +0.06, -0.03, +5},
	"networker":                {+0.04, +0.04, +0.03, -0.03, +4},
	"thought leader":           {+0.06, +0.05, +0.02, -0.04, +8},
	"mentor":                   {+0.02, +0.06, +0.04, -0.03, +5},
	"risk taker":               {+0.04, +0.02, +0.03, -0.03, +3},
	"analytical thinker":       {-0.01, +0.06, +0.04, -0.02, +6},
	"creative problem solver":  {+0.02, +0.04, +0.05, -0.03, +4},
	"pragmatist":               {+0.00, +0.02, +0.04, -0.01, +2},
	"observer":                 {-0.01, +0.01, +0.03, +0.00, +2},
	"critic":                   {-0.02, +0.08, -0.01, +0.02, +3},
	"lurker":                   {-0.04, -0.04, -0.02, +0.12, -8},
	"skeptic":                  {-0.03, +0.05, -0.02, +0.06, -5},
}

// contentInterestOverlap checks how much a persona's interests appear in the content description.
func contentInterestOverlap(interests []string, contentDesc string) float64 {
	if contentDesc == "" {
		return 0
	}
	lower := strings.ToLower(contentDesc)
	score := 0.0
	for _, interest := range interests {
		if strings.Contains(lower, strings.ToLower(interest)) {
			score += 0.08
		}
	}
	if score > 0.20 {
		score = 0.20 // cap
	}
	return score
}

// ageEngagementFactor returns a multiplier for engagement based on age.
// Younger audiences (18-25) are more likely to share trendy content.
// Older audiences (50+) are more selective but more deliberate.
func ageEngagementFactor(age int) (shareDelta, likeDelta float64) {
	switch {
	case age <= 22:
		return +0.04, +0.03 // Gen Z — high share impulse
	case age <= 30:
		return +0.02, +0.02 // Millennial early — active sharers
	case age <= 40:
		return +0.00, +0.01 // Mid-millennial — moderate
	case age <= 50:
		return -0.01, +0.00 // Gen X — selective
	default:
		return -0.03, -0.01 // 50+ — lower impulse sharing
	}
}

// followerInfluenceDelta returns extra share probability for high-follower personas.
func followerInfluenceDelta(followers int) float64 {
	switch {
	case followers >= 5000:
		return +0.07 // micro/macro influencer — likely to amplify
	case followers >= 2000:
		return +0.04
	case followers >= 500:
		return +0.02
	default:
		return 0.0 // small account
	}
}

func fallbackResult(persona PersonaProfile, job SimulationJob, r *rand.Rand) PersonaResult {
	q := computeContentQuality(job)

	// Base probabilities differ for in-target vs out-of-target audiences
	var baseShare, baseComment, baseLike float64
	if persona.InTarget {
		baseShare = 0.04 + 0.10*q
		baseComment = 0.06 + 0.16*q
		baseLike = 0.12 + 0.28*q
	} else {
		baseShare = 0.01 + 0.04*q
		baseComment = 0.02 + 0.07*q
		baseLike = 0.05 + 0.13*q
	}

	// Apply behavior modifier
	bMod := behaviorModifiers[persona.Behavior] // zero-value if unknown: no change
	baseShare += bMod[0]
	baseComment += bMod[1]
	baseLike += bMod[2]

	// Apply age factor
	ageSd, ageLd := ageEngagementFactor(persona.Age)
	baseShare += ageSd
	baseLike += ageLd

	// Apply follower count influence
	baseShare += followerInfluenceDelta(persona.Followers)

	// Apply content-interest overlap
	overlap := contentInterestOverlap(persona.Interests, job.ContentDescription)
	baseShare += overlap * 0.5
	baseComment += overlap * 0.4
	baseLike += overlap * 0.6

	// Clamp to valid probability range
	clamp := func(v float64) float64 {
		if v < 0 {
			return 0
		}
		if v > 0.95 {
			return 0.95
		}
		return v
	}
	baseShare = clamp(baseShare)
	baseComment = clamp(baseComment)
	baseLike = clamp(baseLike)

	// Build cumulative thresholds
	pShare := baseShare
	pComment := pShare + baseComment
	pLike := pComment + baseLike

	roll := r.Float64()

	// Base watch retention, modulated by behavior
	baseWatch := 30
	if persona.InTarget {
		baseWatch = 45
	}
	baseWatch += int(bMod[4]) // behavior watch delta
	if baseWatch < 5 {
		baseWatch = 5
	}

	var action, why string
	var watch int

	switch {
	case roll < pShare:
		action = "share"
		watch = baseWatch + 30 + r.Intn(25)
		why = pickShareWhy(persona, job)
	case roll < pComment:
		action = "comment"
		watch = baseWatch + 15 + r.Intn(25)
		why = pickCommentWhy(persona, job)
	case roll < pLike:
		action = "like"
		watch = baseWatch + 5 + r.Intn(20)
		why = pickLikeWhy(persona, job)
	default:
		action = "skip"
		watch = 5 + r.Intn(baseWatch/2+5)
		why = pickSkipWhy(persona)
	}

	if watch > 100 {
		watch = 100
	}

	return PersonaResult{
		PersonaID: persona.ID,
		Name:      persona.Name,
		Age:       persona.Age,
		Location:  persona.Location,
		Shown:     persona.Shown,
		Followers: persona.Followers,
		Gender:    persona.Gender,
		Behavior:  persona.Behavior,
		Interests: persona.Interests,
		InTarget:  persona.InTarget,
		Action:    action,
		Watch:     watch,
		Why:       why,
		Round:     1,
	}
}

// ── Persona-aware "why" text generators ──────────────────────────────────────

func pickShareWhy(p PersonaProfile, job SimulationJob) string {
	switch p.Behavior {
	case "influencer", "content creator":
		return "This is exactly the signal my audience needs to see. Sharing immediately."
	case "sharer", "networker":
		return "Forwarded to three people in my circle — high-signal content worth amplifying."
	case "thought leader":
		return "A perspective I haven't seen articulated this cleanly before. Worth sharing."
	case "early adopter":
		return "Spotted this before it blows up. Sharing now while it's still niche."
	default:
		if p.InTarget {
			return "Directly relevant to what I'm working on right now. Sent to my team."
		}
		return "Surprising crossover with my world — sent to a colleague who'll appreciate it."
	}
}

func pickCommentWhy(p PersonaProfile, job SimulationJob) string {
	switch p.Behavior {
	case "critic":
		return "Watched it through, but I'm not fully convinced. Left a pushback comment."
	case "analytical thinker":
		return "Interesting framework, but the data behind claim #2 is questionable. Commented."
	case "commenter":
		return "This sparked something — left a comment to keep the conversation going."
	case "skeptic":
		return "I see the appeal but have questions. Commented to dig deeper."
	case "mentor":
		return "Saw a gap in the advice given here. Added context in the comments."
	default:
		return "Good hook. Left a question — genuinely curious about the details."
	}
}

func pickLikeWhy(p PersonaProfile, job SimulationJob) string {
	switch p.Behavior {
	case "trend follower":
		return "This is getting traction for a reason. Quick like and moved on."
	case "enthusiast":
		return "Love this — hit like without hesitation."
	case "lurker":
		return "Won't comment, but this earned a like. Solid content."
	case "observer":
		return "Watched quietly. The value was real, even if it's not my space to comment."
	default:
		if p.InTarget {
			return "Clear value, good production. Liked it."
		}
		return "Outside my main topics but well-made. Gave it a like."
	}
}

func pickSkipWhy(p PersonaProfile) string {
	switch p.Behavior {
	case "lurker":
		return "Scrolled past without engaging — just not something I'd surface react to."
	case "skeptic":
		return "Too many red flags in the framing. Swiped away early."
	case "critic":
		return "The opening claim felt overblown. Lost me in the first 5 seconds."
	case "pragmatist":
		return "Can't see an immediate application for this. Moved on."
	default:
		if p.Age > 45 {
			return "Not relevant to where I am right now. Scrolled past."
		}
		return "Hook didn't land fast enough. Gone before the 3-second mark."
	}
}
