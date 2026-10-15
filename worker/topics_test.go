package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestCanonicalTopics(t *testing.T) {
	got := canonicalTopics([]string{"Self-Improvement", "made up", "movies", "self-improvement", "history", "politics"})
	want := []string{"self-improvement", "movies", "history"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("canonicalTopics = %v, want %v", got, want)
	}
	if got := canonicalTopics(nil); got == nil || len(got) != 0 {
		t.Fatalf("want empty non-nil slice, got %#v", got)
	}
}

func TestNewKeywordTopics(t *testing.T) {
	for text, want := range map[string]string{
		"men 25-40 into self-improvement and motivation": "self-improvement",
		"true crime fans":                  "true crime",
		"people job hunting after college": "careers",
	} {
		if got := detectTopics(text); len(got) == 0 || got[0] != want {
			t.Errorf("detectTopics(%q) = %v, want %s first", text, got, want)
		}
	}
}

func TestLLMTopicsOverrideKeywords(t *testing.T) {
	job := SimulationJob{JobID: "t", Demographic: "college students in India", ContentDescription: "What if the most dangerous thing a man can do is stay comfortable?"}
	var gotDemo, gotHook string
	reader := func(demo, hook, _ string, _ []string) (TopicReading, error) {
		gotDemo, gotHook = demo, hook
		return TopicReading{Audience: []string{"careers", "education"}, Content: []string{"self-improvement"}}, nil
	}

	sc := newSimContext(job, VideoInsight{}, reader)
	if gotDemo != job.Demographic || !strings.Contains(gotHook, "most dangerous thing") {
		t.Errorf("reader got demo=%q hook=%q", gotDemo, gotHook)
	}
	if !reflect.DeepEqual(sc.audience.Topics, []string{"careers", "education"}) || !reflect.DeepEqual(sc.content.Topics, []string{"self-improvement"}) || sc.topicSrc != "llm" {
		t.Fatalf("audience=%v content=%v src=%s", sc.audience.Topics, sc.content.Topics, sc.topicSrc)
	}
	if sc.audience.AgeNote != "Students (18–24)" || len(sc.audience.Regions) == 0 {
		t.Errorf("age/region parsing should still apply: %+v", sc.audience)
	}

	failing := func(string, string, string, []string) (TopicReading, error) {
		return TopicReading{}, errors.New("rate limited")
	}
	fb := newSimContext(job, VideoInsight{}, failing)
	if fb.topicSrc != "keywords" || !reflect.DeepEqual(fb.audience.Topics, detectTopics(job.Demographic)) {
		t.Fatalf("failed read should fall back to keywords, got %v (%s)", fb.audience.Topics, fb.topicSrc)
	}
}

func TestReadTopicsWithLLMClassifiesSeparately(t *testing.T) {
	var prompts []string
	withFakeGroq(t, func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Messages []struct {
				Content string `json:"content"`
			} `json:"messages"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		prompt := req.Messages[0].Content
		prompts = append(prompts, prompt)
		if strings.Contains(prompt, "Audience:") {
			w.Write([]byte(`{"choices":[{"message":{"content":"{\"topics\":[\"Self-improvement\",\"nonsense\"]}"}}]}`))
		} else {
			w.Write([]byte(`{"choices":[{"message":{"content":"{\"topics\":[\"movies\",\"self-improvement\"]}"}}]}`))
		}
	})
	got, err := readTopicsWithLLM("men into bettering themselves", "What if the most dangerous thing a man can do", "men into bettering themselves", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Audience, []string{"self-improvement"}) || !reflect.DeepEqual(got.Content, []string{"movies", "self-improvement"}) {
		t.Fatalf("got %+v", got)
	}
	// The post must be classified without seeing the audience, and vice versa. (The
	// third call rates the hook for its audience; the fake's reply has no score.)
	if got.Hook != nil {
		t.Errorf("a reply without a score should leave the hook unrated, got %+v", got.Hook)
	}
	if len(prompts) != 3 || strings.Contains(prompts[1], "bettering themselves") || strings.Contains(prompts[0], "dangerous thing") {
		t.Fatalf("audience and post leaked into each other's prompt: %q", prompts)
	}
}

func TestCleanNotesSplitsMashedList(t *testing.T) {
	// Seen from the hook rater: one "fix" holding three quoted fixes.
	got := cleanNotes([]string{`Start with a bold claim: "We just built a virality predictor!","Show a quick visual of the simulator in action","Add a question to engage: "Want to win your next hackathon?"`})
	if len(got) != 3 || got[0] != `Start with a bold claim: "We just built a virality predictor!"` || got[1] != "Show a quick visual of the simulator in action" {
		t.Fatalf("cleanNotes = %q", got)
	}
	if got := cleanNotes([]string{"Lead with the payoff, not the setup."}); len(got) != 1 || got[0] != "Lead with the payoff, not the setup." {
		t.Errorf("a normal note changed: %q", got)
	}
}
