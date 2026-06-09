// gemini.go
// Calls Google Gemini Flash API for persona evaluation.
// Gemini Flash is free tier — perfect for development and thesis demos.
// Drop-in replacement for the Claude implementation.

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strings"
)

// Gemini API request structures
type geminiPart struct {
	Text string `json:"text"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiRequest struct {
	Contents         []geminiContent  `json:"contents"`
	GenerationConfig geminiGenConfig  `json:"generationConfig"`
}

type geminiGenConfig struct {
	MaxOutputTokens int     `json:"maxOutputTokens"`
	Temperature     float64 `json:"temperature"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

// EvaluatePersona sends one persona + content to Gemini Flash.
// Gemini plays the persona and decides how they react.
// This runs inside a goroutine so 100 of these fire simultaneously.
func EvaluatePersona(persona PersonaProfile, job SimulationJob) PersonaResult {
	apiKey := os.Getenv("GEMINI_API_KEY")

	prompt := fmt.Sprintf(`You are simulating a social media user with this exact profile:

Name: %s
Age: %d
Gender: %s
Location: %s
Followers: %d
Interests: %s
Behavior pattern: %s
In target demographic ("%s"): %v

You just saw this content on your feed:
"%s"

Respond ONLY with a valid JSON object, no other text, no markdown:
{
  "action": "like" or "share" or "comment" or "skip",
  "watch_percent": <0-100, how much of the content you watched>,
  "why": "<1-2 sentences explaining your reaction as this person>"
}

Be realistic. Your behavior pattern and interests should drive your decision.
If you are NOT in the target demographic, you are less likely to engage.`,
		persona.Name,
		persona.Age,
		persona.Gender,
		persona.Location,
		persona.Followers,
		strings.Join(persona.Interests, ", "),
		persona.Behavior,
		job.Demographic,
		persona.InTarget,
		getContentText(job),
	)

	reqBody := geminiRequest{
		Contents: []geminiContent{
			{Parts: []geminiPart{{Text: prompt}}},
		},
		GenerationConfig: geminiGenConfig{
			MaxOutputTokens: 200,
			Temperature:     0.7,
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fallbackResult(persona)
	}

	// Gemini Flash API endpoint
	url := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash:generateContent?key=%s",
		apiKey,
	)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return fallbackResult(persona)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fallbackResult(persona)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fallbackResult(persona)
	}

	var geminiResp geminiResponse
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return fallbackResult(persona)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return fallbackResult(persona)
	}

	text := geminiResp.Candidates[0].Content.Parts[0].Text

	// Extract JSON from response
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}") + 1
	if start == -1 || end == 0 {
		return fallbackResult(persona)
	}

	var decision struct {
		Action       string `json:"action"`
		WatchPercent int    `json:"watch_percent"`
		Why          string `json:"why"`
	}

	if err := json.Unmarshal([]byte(text[start:end]), &decision); err != nil {
		return fallbackResult(persona)
	}

	validActions := map[string]bool{"like": true, "share": true, "comment": true, "skip": true}
	if !validActions[decision.Action] {
		decision.Action = "skip"
	}

	return PersonaResult{
		PersonaID: persona.ID,
		Name:      persona.Name,
		Age:       persona.Age,
		Location:  persona.Location,
		InTarget:  persona.InTarget,
		Action:    decision.Action,
		Watch:     decision.WatchPercent,
		Why:       decision.Why,
		Round:     1,
	}
}

func getContentText(job SimulationJob) string {
	if job.ContentDescription != "" {
		return job.ContentDescription
	}
	if job.VideoFilename != "" {
		return fmt.Sprintf("[Video: %s]", job.VideoFilename)
	}
	return "[No content provided]"
}

func fallbackResult(persona PersonaProfile) PersonaResult {
	return PersonaResult{
		PersonaID: persona.ID,
		Name:      persona.Name,
		Age:       persona.Age,
		Location:  persona.Location,
		InTarget:  persona.InTarget,
		Action:    "skip",
		Watch:     rand.Intn(30),
		Why:       "API unavailable — fallback result",
		Round:     1,
	}
}