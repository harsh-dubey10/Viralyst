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
}

// PersonaResult is what a persona decides after seeing the content
type PersonaResult struct {
	PersonaID int    `json:"persona_id"`
	Name      string `json:"name"`
	Age       int    `json:"age"`
	Location  string `json:"location"`
	InTarget  bool   `json:"in_target"`
	Action    string `json:"action"`  // like, share, comment, skip
	Watch     int    `json:"watch"`   // % of content watched
	Why       string `json:"why"`     // AI reasoning
	Round     int    `json:"round"`
}

// SimulationResult is the full output written back to Redis
type SimulationResult struct {
	JobID       string          `json:"job_id"`
	Status      string          `json:"status"`
	Personas    []PersonaResult `json:"personas"`
	Virality    int             `json:"virality"`
	AlgoScore   int             `json:"algo_score"`
	Engagement  int             `json:"engagement"`
	Breakdown   ActionBreakdown `json:"breakdown"`
	Verdict     VerdictData     `json:"verdict"`
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