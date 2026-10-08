// topics.go
// Reads which interest topics the audience and the hook are about using the LLM, so
// wording the keyword lexicon doesn't list ("men into self-improvement", "the most
// dangerous thing a man can do") still maps to the right topics. Keyword matching in
// analysis.go remains the offline fallback.

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

// TopicReading is which lexicon topics the audience cares about and the hook is about,
// plus the LLM's rating of the hook for that audience (nil if rating failed).
type TopicReading struct {
	Audience []string
	Content  []string
	Hook     *HookRating
}

// HookRating is an LLM's judgement of how well a hook stops the scroll for its audience.
type HookRating struct {
	Score     int      // 0–100
	Strengths []string // what works
	Fixes     []string // specific changes that would make it stronger
}

// topicReader reads topics from the audience description and hook text. extra lists
// custom niche topics allowed alongside the lexicon; audience describes who the post is
// for (niche + demographic) when rating the hook.
type topicReader func(demographic, hook, audience string, extra []string) (TopicReading, error)

// maxReadTopics caps each list; keyword detection rarely finds more than this either.
const maxReadTopics = 3

// The audience and the post are classified in separate calls: shown together, the model
// reads the post as being about whatever the audience likes, so every fit looks perfect.
const (
	audienceTopicPrompt = `Which interest topics does this social media audience clearly care about?

Allowed topics (use these exact names only): %s

Audience: %q

Reply with JSON: {"topics": [...]} — only topics the description names or directly implies, most important first: usually 1 or 2, at most 3. Do not add loosely related topics. [] if the description says nothing about their interests.`

	contentTopicPrompt = `Which interest topics is this social media post about?

Allowed topics (use these exact names only): %s

Post (what the viewer hears and reads in the first 3 seconds): %q

Reply with JSON: {"topics": [...]} — up to 3, most important first. Judge the subject, not props or settings that merely appear. [] if no topic fits.`

	hookRatingPrompt = `You are a short-form video strategist. Rate how well this post's first 3 seconds would stop someone in its intended audience from scrolling past.

Intended audience: %q

Post (what the viewer hears, reads and sees in the first 3 seconds): %q

Judge it against other posts in the same niche, not against all of social media. Consider: is the payoff or tension clear immediately, is it specific, does it feel native to the niche (not an ad), would this audience instantly know it's for them.
Scale: 50 = a typical post in this niche, 70 = better than most, 85+ = top 5%%, below 30 = most people would swipe.

Reply with JSON: {"score": <0-100>, "strengths": ["<at most 2 short phrases>"], "fixes": ["<at most 3 specific changes for THIS post, e.g. a rewritten opening line>"]}`
)

const maxHookNotes = 3

// readTopicsWithLLM asks the persona model to classify each input onto the lexicon (plus
// any custom niche topics), and to rate the hook for its audience. A failed hook rating
// is logged and left nil; the keyword score is used instead.
func readTopicsWithLLM(demographic, hook, audienceDesc string, extra []string) (TopicReading, error) {
	audience := []string{}
	if strings.TrimSpace(demographic) != "" {
		var err error
		if audience, err = classifyTopics(audienceTopicPrompt, demographic, extra); err != nil {
			return TopicReading{}, fmt.Errorf("audience: %w", err)
		}
	}
	content := []string{}
	var rating *HookRating
	if strings.TrimSpace(hook) != "" {
		var err error
		if content, err = classifyTopics(contentTopicPrompt, hook, extra); err != nil {
			return TopicReading{}, fmt.Errorf("content: %w", err)
		}
		if rating, err = rateHook(audienceDesc, hook); err != nil {
			log.Printf("[groq] hook rating failed, using keyword score: %v", err)
		}
	}
	return TopicReading{Audience: audience, Content: content, Hook: rating}, nil
}

// jsonCompletion runs a deterministic JSON-mode completion and decodes the reply into out.
func jsonCompletion(prompt string, maxTokens int, out any) error {
	model := groqModel()
	payload := groqRequest{
		Model:    model,
		Messages: []groqMessage{{Role: "user", Content: prompt}},
		// Reasoning tokens count toward max_tokens, so leave room beyond the short reply.
		MaxTokens:      maxTokens,
		Temperature:    0,
		ResponseFormat: map[string]string{"type": "json_object"},
	}
	if strings.HasPrefix(model, "openai/gpt-oss") || strings.HasPrefix(model, "qwen/") {
		payload.ReasoningEffort = "low"
	}

	raw, err := callGroqChat(payload)
	if err != nil {
		return err
	}
	start, end := strings.Index(raw, "{"), strings.LastIndex(raw, "}")
	if start < 0 || end <= start {
		return fmt.Errorf("no JSON in reply: %q", raw)
	}
	if err := json.Unmarshal([]byte(raw[start:end+1]), out); err != nil {
		return fmt.Errorf("parse reply: %w", err)
	}
	return nil
}

func classifyTopics(promptTemplate, text string, extra []string) ([]string, error) {
	allowed := append(allTopicNames[:len(allTopicNames):len(allTopicNames)], extra...)
	var reply struct {
		Topics []string `json:"topics"`
	}
	if err := jsonCompletion(fmt.Sprintf(promptTemplate, strings.Join(allowed, ", "), text), 500, &reply); err != nil {
		return nil, err
	}
	return canonicalTopics(reply.Topics, extra...), nil
}

// rateHook asks the LLM how strong the hook is for this audience, with concrete fixes.
func rateHook(audience, hook string) (*HookRating, error) {
	if strings.TrimSpace(audience) == "" {
		audience = "a general social media audience"
	}
	var reply struct {
		Score     *int     `json:"score"`
		Strengths []string `json:"strengths"`
		Fixes     []string `json:"fixes"`
	}
	if err := jsonCompletion(fmt.Sprintf(hookRatingPrompt, audience, hook), 900, &reply); err != nil {
		return nil, err
	}
	if reply.Score == nil {
		return nil, fmt.Errorf("hook rating has no score")
	}
	return &HookRating{Score: clampInt(*reply.Score, 0, 100), Strengths: cleanNotes(reply.Strengths), Fixes: cleanNotes(reply.Fixes)}, nil
}

// cleanNotes trims notes, drops empty ones and keeps at most maxHookNotes. Never returns nil.
func cleanNotes(notes []string) []string {
	out := []string{}
	for _, n := range notes {
		if n = strings.TrimSpace(n); n != "" && len(out) < maxHookNotes {
			out = append(out, n)
		}
	}
	return out
}

// canonicalTopics keeps only known topic names (the lexicon plus extra, matched
// case-insensitively), in the given order, without duplicates, capped at maxReadTopics.
// Never returns nil.
func canonicalTopics(names []string, extra ...string) []string {
	known := map[string]string{}
	for _, t := range append(allTopicNames[:len(allTopicNames):len(allTopicNames)], extra...) {
		known[strings.ToLower(t)] = t
	}
	out, seen := []string{}, map[string]bool{}
	for _, n := range names {
		t, ok := known[strings.ToLower(strings.TrimSpace(n))]
		if ok && !seen[t] && len(out) < maxReadTopics {
			seen[t] = true
			out = append(out, t)
		}
	}
	return out
}
