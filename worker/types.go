// types.go
// All shared data types used across the worker.
// In Go, types are defined separately and imported where needed.
// This is equivalent to your models.py in FastAPI.

package main

// SimulationJob is what we receive from Redis queue.
// The json tags tell Go how to parse the JSON string from Redis.
// e.g. "job_id" in JSON maps to JobID in Go
type SimulationJob struct {
	JobID              string `json:"job_id"`
	Demographic        string `json:"demographic"`
	Population         int    `json:"population"`
	Boost              int    `json:"boost"`
	ContentDescription string `json:"content_description"`
	VideoFilename      string `json:"video_filename"`
	Status             string `json:"status"`
}

// PersonaProfile is one AI persona — their identity
type PersonaProfile struct {
	ID        int      `json:"id"`
	Name      string   `json:"name"`
	Age       int      `json:"age"`
	Gender    string   `json:"gender"`
	Location  string   `json:"location"`
	Followers int      `json:"followers"`
	Interests []string `json:"interests"`
	Behavior  string   `json:"behavior"`
	InTarget  bool     `json:"in_target"`
	Shown     bool     `json:"shown"`
}

// PersonaResult is what a persona decides after seeing the content
type PersonaResult struct {
	PersonaID int    `json:"persona_id"`
	Name      string `json:"name"`
	Age       int    `json:"age"`
	Location  string `json:"location"`
	Shown     bool   `json:"shown"`

	Followers int      `json:"followers"`
	Gender    string   `json:"gender"`
	Behavior  string   `json:"behavior"`
	Interests []string `json:"interests"`

	InTarget bool   `json:"in_target"`
	Action   string `json:"action"`
	Watch    int    `json:"watch"`
	Why      string `json:"why"`
	Round    int    `json:"round"`

	llm bool // decided by the LLM (true) or the offline model (false); not serialized
}
type EdgeResult struct {
	From  int    `json:"from"`
	To    int    `json:"to"`
	Type  string `json:"type"` // "direct" or "algo"
	Round int    `json:"round"`
}

// SimulationResult is the full output written back to Redis
type SimulationResult struct {
	JobID      string          `json:"job_id"`
	Status     string          `json:"status"`
	Personas   []PersonaResult `json:"personas"`
	Edges      []EdgeResult    `json:"edges"`
	Virality   int             `json:"virality"`
	AlgoScore  int             `json:"algo_score"`
	Engagement int             `json:"engagement"`
	Breakdown  ActionBreakdown `json:"breakdown"`
	Verdict    VerdictData     `json:"verdict"`
	Analysis   Analysis        `json:"analysis"`
}

// ActionBreakdown counts each action type
type ActionBreakdown struct {
	Likes    int `json:"likes"`
	Shares   int `json:"shares"`
	Comments int `json:"comments"`
	Scrolls  int `json:"scrolls"`
}

// VerdictData is the final analysis
type VerdictData struct {
	Head      string `json:"head"`
	Sub       string `json:"sub"`
	Verdict   string `json:"verdict"`
	InPct     int    `json:"in_pct"`
	ShareRate int    `json:"share_rate"`
}

// Analysis explains which context signals drove the verdict.
type Analysis struct {
	AudienceTopics      []string `json:"audience_topics"`
	AudienceAge         string   `json:"audience_age"`
	AudienceRegions     []string `json:"audience_regions"`
	ContentTopics       []string `json:"content_topics"`
	AudienceFit         int      `json:"audience_fit"` // 0–100, or -1 if the audience has no detectable interests
	HookScore           int      `json:"hook_score"`   // 0–100
	HookSignals         []string `json:"hook_signals"`
	Recommendations     []string `json:"recommendations"`
	HookSkipRate        int      `json:"hook_skip_rate"`        // % of round-1 seed viewers who skipped
	RelevantSkipRate    int      `json:"relevant_skip_rate"`    // same, among seed viewers the topic is relevant to (-1 if none)
	InTargetEngagement  int      `json:"in_target_engagement"`  // % of reached in-target personas who liked/commented/shared
	OutTargetEngagement int      `json:"out_target_engagement"` // same, out-of-target
	Amplification       float64  `json:"amplification"`         // reached ÷ seed audience
	Engine              string   `json:"engine"`                // "llm", "heuristic" or "mixed"

	// Offline engine only: verdict counts across replicate simulations, and the share
	// that agree with the reported verdict. Empty/-1 for LLM runs (too costly to repeat).
	VerdictDistribution map[string]int `json:"verdict_distribution"`
	Confidence          int            `json:"confidence"`
	Replicates          int            `json:"replicates"`
}
