// topics.go
// Reads which interest topics the audience and the hook are about using the LLM, so
// wording the keyword lexicon doesn't list ("men into self-improvement", "the most
// dangerous thing a man can do") still maps to the right topics. Keyword matching in
// analysis.go remains the offline fallback.

package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

// TopicReading is which lexicon topics the audience cares about and the hook is about.
type TopicReading struct {
	Audience []string
	Content  []string
}

// topicReader reads topics from the audience description and hook text.
type topicReader func(demographic, hook string) (TopicReading, error)

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
)

// readTopicsWithLLM asks the persona model to classify each input onto the lexicon.
func readTopicsWithLLM(demographic, hook string) (TopicReading, error) {
	audience, err := classifyTopics(audienceTopicPrompt, demographic)
	if err != nil {
		return TopicReading{}, fmt.Errorf("audience: %w", err)
	}
	content := []string{}
	if strings.TrimSpace(hook) != "" {
		if content, err = classifyTopics(contentTopicPrompt, hook); err != nil {
			return TopicReading{}, fmt.Errorf("content: %w", err)
		}
	}
	return TopicReading{Audience: audience, Content: content}, nil
}

func classifyTopics(promptTemplate, text string) ([]string, error) {
	model := groqModel()
	payload := groqRequest{
		Model:    model,
		Messages: []groqMessage{{Role: "user", Content: fmt.Sprintf(promptTemplate, strings.Join(allTopicNames, ", "), text)}},
		// Reasoning tokens count toward max_tokens, so leave room beyond the short reply.
		MaxTokens:      500,
		Temperature:    0,
		ResponseFormat: map[string]string{"type": "json_object"},
	}
	if strings.HasPrefix(model, "openai/gpt-oss") || strings.HasPrefix(model, "qwen/") {
		payload.ReasoningEffort = "low"
	}

	raw, err := callGroqChat(payload)
	if err != nil {
		return nil, err
	}
	var reply struct {
		Topics []string `json:"topics"`
	}
	start, end := strings.Index(raw, "{"), strings.LastIndex(raw, "}")
	if start < 0 || end <= start {
		return nil, fmt.Errorf("no JSON in topic reply: %q", raw)
	}
	if err := json.Unmarshal([]byte(raw[start:end+1]), &reply); err != nil {
		return nil, fmt.Errorf("parse topic reply: %w", err)
	}
	return canonicalTopics(reply.Topics), nil
}

// canonicalTopics keeps only known topic names (matched case-insensitively), in the
// given order, without duplicates, capped at maxReadTopics. Never returns nil.
func canonicalTopics(names []string) []string {
	known := map[string]string{}
	for _, t := range allTopicNames {
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
