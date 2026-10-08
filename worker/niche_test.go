package main

import (
	"math/rand"
	"reflect"
	"testing"
)

func TestResolveNiche(t *testing.T) {
	cases := []struct {
		niche      string
		want       []string
		wantCustom bool
	}{
		{"", nil, false},
		{"Travel", []string{"travel"}, false},
		{"quick weeknight recipes", []string{"cooking"}, false},
		{"  Chess  openings ", []string{"Chess openings"}, true},
	}
	for _, c := range cases {
		got, custom := resolveNiche(c.niche)
		if !reflect.DeepEqual(got, c.want) || (custom != nil) != c.wantCustom {
			t.Errorf("resolveNiche(%q) = %v, custom=%v; want %v, custom=%v", c.niche, got, custom != nil, c.want, c.wantCustom)
		}
	}
}

func TestCustomNicheDrivesAudienceAndContent(t *testing.T) {
	job := SimulationJob{JobID: "n", Niche: "Knitting", Demographic: "retirees in the UK",
		ContentDescription: "I knit this sweater in 3 days with one ball of yarn. Here's the pattern.", Population: 120, Boost: 5}
	sc := newSimContext(job, VideoInsight{}, nil)
	if !reflect.DeepEqual(sc.audience.Topics, []string{"Knitting"}) {
		t.Fatalf("audience topics = %v, want the custom niche", sc.audience.Topics)
	}
	if !reflect.DeepEqual(sc.content.Topics, []string{"Knitting"}) {
		t.Fatalf("content topics = %v: a hook about knitting should match the custom niche", sc.content.Topics)
	}
	if fit := AudienceFit(sc.audience, sc.content); fit != 100 {
		t.Errorf("audience fit = %d, want 100", fit)
	}

	personas := GeneratePersonas(job.Population, sc.audience, inTargetRatio, rand.New(rand.NewSource(1)))
	for _, p := range personas {
		has := len(matchedTopics(p.Interests, []string{"Knitting"})) > 0
		if p.InTarget != has {
			t.Fatalf("persona %d in_target=%v but interests %v", p.ID, p.InTarget, p.Interests)
		}
	}
}

func TestNicheLeadsDemographicTopics(t *testing.T) {
	job := SimulationJob{JobID: "n2", Niche: "travel", Demographic: "Gen Z gamers", ContentDescription: "x"}
	sc := newSimContext(job, VideoInsight{}, nil)
	if !reflect.DeepEqual(sc.audience.Topics, []string{"travel", "gaming"}) {
		t.Fatalf("audience topics = %v, want niche first then demographic", sc.audience.Topics)
	}
}

func TestLLMHookRatingReplacesKeywordScore(t *testing.T) {
	job := SimulationJob{JobID: "h", Niche: "cooking", ContentDescription: "This 10-minute garlic pasta beats every restaurant version."}
	var gotAudience string
	var gotExtra []string
	reader := func(_, _, audience string, extra []string) (TopicReading, error) {
		gotAudience, gotExtra = audience, extra
		return TopicReading{Content: []string{"cooking"}, Hook: &HookRating{Score: 81, Strengths: []string{"clear payoff"}, Fixes: []string{"Show the pasta in frame 1."}}}, nil
	}
	sc := newSimContext(job, VideoInsight{}, reader)
	if gotAudience != "cooking" || gotExtra != nil {
		t.Errorf("reader got audience=%q extra=%v", gotAudience, gotExtra)
	}
	if sc.hookSrc != "llm" || sc.content.HookScore != 0.81 || !reflect.DeepEqual(sc.content.Issues, []string{"Show the pasta in frame 1."}) {
		t.Fatalf("hook not taken from the LLM rating: src=%s score=%v issues=%v", sc.hookSrc, sc.content.HookScore, sc.content.Issues)
	}
}

func TestDecideAction(t *testing.T) {
	r := rand.New(rand.NewSource(7))
	count := func(d groqPersonaDecision) map[string]int {
		n := map[string]int{}
		for i := 0; i < 2000; i++ {
			n[decideAction(d, r)]++
		}
		return n
	}
	if got := count(groqPersonaDecision{Watch: 10, Like: 10, Share: 10}); got["skip"] != 2000 {
		t.Errorf("a persona who swiped away must skip, got %v", got)
	}
	keen := count(groqPersonaDecision{Watch: 90, Like: 9, Comment: 3, Share: 8})
	meh := count(groqPersonaDecision{Watch: 50, Like: 5, Comment: 1, Share: 2})
	if keen["share"] < 600 || meh["share"] > 100 {
		t.Errorf("share odds should track the share score: keen=%v meh=%v", keen, meh)
	}
	if keen["skip"] >= meh["skip"] {
		t.Errorf("keener personas should engage more: keen=%v meh=%v", keen, meh)
	}
}

func TestCustomNicheNamedInHookCountsEvenIfLLMMissesIt(t *testing.T) {
	job := SimulationJob{JobID: "c", Niche: "chess", Demographic: "club players", ContentDescription: "This 5-move chess trap wins the queen."}
	reader := func(_, _, _ string, _ []string) (TopicReading, error) {
		return TopicReading{Audience: []string{"gaming"}, Content: []string{}}, nil
	}
	sc := newSimContext(job, VideoInsight{}, reader)
	if !reflect.DeepEqual(sc.content.Topics, []string{"chess"}) || !reflect.DeepEqual(sc.audience.Core, []string{"chess"}) {
		t.Fatalf("content=%v core=%v", sc.content.Topics, sc.audience.Core)
	}
	if fit := AudienceFit(sc.audience, sc.content); fit != 100 {
		t.Errorf("fit = %d: the extra 'gaming' interest must not dilute a niche match", fit)
	}
}
