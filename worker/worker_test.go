package main

import (
	"fmt"
	"math/rand"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/joho/godotenv"
)

func TestMain(m *testing.M) {
	if os.Getenv("VIRALYST_LLM_TEST") == "1" {
		// Opt-in live run against Groq using worker/.env (see llm_test.go).
		_ = godotenv.Load()
		os.Exit(m.Run())
	}
	// Force the heuristic engine so tests are offline and deterministic.
	for _, prefix := range []string{"GROQ", "GEMINI", "ANTHROPIC"} {
		os.Unsetenv(prefix + "_API_KEY")
		os.Unsetenv(prefix + "_API_KEYS")
		for i := 1; i <= 10; i++ {
			os.Unsetenv(fmt.Sprintf("%s_API_KEY_%d", prefix, i))
		}
	}
	os.Exit(m.Run())
}

func testJob() SimulationJob {
	return SimulationJob{
		JobID:              "00000000-0000-0000-0000-000000000001",
		Demographic:        "Gen Z AI developers",
		Population:         150,
		Boost:              6,
		ContentDescription: "How we built an AI tool growth framework from 0 to 1",
	}
}

func TestProcessSimulationDeterministic(t *testing.T) {
	w := &Worker{concurrency: 8}
	a := w.ProcessSimulation(testJob())
	b := w.ProcessSimulation(testJob())
	if !reflect.DeepEqual(a, b) {
		t.Fatal("same job produced different results; parallel evaluation broke reproducibility")
	}
}

func TestProcessSimulationInvariants(t *testing.T) {
	w := &Worker{concurrency: 4}
	job := testJob()
	res := w.ProcessSimulation(job)

	if len(res.Personas) != job.Population {
		t.Fatalf("got %d personas, want %d", len(res.Personas), job.Population)
	}
	for _, s := range []int{res.Virality, res.AlgoScore, res.Engagement} {
		if s < 0 || s > 100 {
			t.Errorf("score %d out of [0,100]", s)
		}
	}

	byID := map[int]PersonaResult{}
	for _, p := range res.Personas {
		byID[p.PersonaID] = p
		if p.Shown == (p.Action == "never") {
			t.Errorf("persona %d: shown=%v but action=%q", p.PersonaID, p.Shown, p.Action)
		}
		if p.Shown && (p.Round < 1 || p.Round > 3) {
			t.Errorf("persona %d shown with round %d", p.PersonaID, p.Round)
		}
	}
	for _, e := range res.Edges {
		to, ok := byID[e.To]
		if !ok || !to.Shown || to.Round != e.Round {
			t.Errorf("edge %+v points at persona not reached in that round (%+v)", e, to)
		}
	}
}

func TestComputeVerdict(t *testing.T) {
	cases := []struct {
		a         Analysis
		shareRate int
		virality  int
		want      string
	}{
		{Analysis{HookSkipRate: 80, RelevantSkipRate: 80}, 0, 5, "Weak hook."},
		{Analysis{HookSkipRate: 20, InTargetEngagement: 70, OutTargetEngagement: 45}, 15, 70, "Viral hit."},
		{Analysis{HookSkipRate: 30, InTargetEngagement: 55, OutTargetEngagement: 10}, 8, 40, "Niche hit."},
		{Analysis{HookSkipRate: 40, InTargetEngagement: 38}, 2, 25, "Echo chamber."},
		{Analysis{HookSkipRate: 50, InTargetEngagement: 10}, 0, 10, "Weak signal."},
	}
	for _, c := range cases {
		if head, _, _ := computeVerdict(c.a, c.shareRate, c.virality); head != c.want {
			t.Errorf("computeVerdict(%+v, %d, %d) = %q, want %q", c.a, c.shareRate, c.virality, head, c.want)
		}
	}
}

func TestParseAudience(t *testing.T) {
	a := ParseAudience("Gen Z tech founders & SaaS product managers in India and the US")
	if a.MinAge != 18 || a.MaxAge != 27 {
		t.Errorf("age = %d–%d, want 18–27", a.MinAge, a.MaxAge)
	}
	for _, want := range []string{"startup ops", "B2B SaaS", "product design"} {
		if len(matchedTopics([]string{want}, a.Topics)) == 0 {
			t.Errorf("topics %v missing %q", a.Topics, want)
		}
	}
	if !reflect.DeepEqual(a.Regions, []string{"US", "India"}) {
		t.Errorf("regions = %v", a.Regions)
	}
	if got := ParseAudience("people aged 35-50"); got.MinAge != 35 || got.MaxAge != 50 {
		t.Errorf("explicit range parsed as %d–%d", got.MinAge, got.MaxAge)
	}
	// Whole-word matching: "said" must not trigger the AI topic, "us" (pronoun) not the US.
	if got := ParseAudience("people who said join us"); len(got.Topics) != 0 || len(got.Regions) != 0 {
		t.Errorf("false positives: topics=%v regions=%v", got.Topics, got.Regions)
	}
}

func TestInTargetPersonasMatchAudience(t *testing.T) {
	job := testJob()
	res := (&Worker{concurrency: 4}).ProcessSimulation(job)
	audience := ParseAudience(job.Demographic)
	for _, p := range res.Personas {
		hits := len(matchedTopics(p.Interests, audience.Topics))
		if p.InTarget && (hits == 0 || p.Age < audience.MinAge || p.Age > audience.MaxAge) {
			t.Errorf("in-target persona %d doesn't fit audience: %+v", p.PersonaID, p)
		}
		if !p.InTarget && hits != 0 {
			t.Errorf("out-of-target persona %d has audience interests %v", p.PersonaID, p.Interests)
		}
	}
}

// Verdict explanations must describe what happened, not assume it.
func TestVerdictTextMatchesTheSpread(t *testing.T) {
	_, sub, _ := computeVerdict(Analysis{AudienceFit: 100, RelevantSkipRate: 20, InTargetEngagement: 40, OutTargetEngagement: 50, Amplification: 2.3}, 5, 48)
	if strings.Contains(sub, "never escaped") || !strings.Contains(sub, "2.3×") {
		t.Errorf("echo chamber that reached 2.3× the seed: %q", sub)
	}
	_, sub, _ = computeVerdict(Analysis{AudienceFit: 100, RelevantSkipRate: 20, InTargetEngagement: 40, OutTargetEngagement: 20, Amplification: 1.1}, 3, 30)
	if !strings.Contains(sub, "never escaped") {
		t.Errorf("echo chamber that stayed in the seed: %q", sub)
	}
	_, sub, _ = computeVerdict(Analysis{AudienceFit: 100, RelevantSkipRate: 20, InTargetEngagement: 55, OutTargetEngagement: 45, Amplification: 2}, 7, 55)
	if strings.Contains(sub, "limited") {
		t.Errorf("niche hit whose outsiders engaged 45%%: %q", sub)
	}
}

// Rationales read correctly with or without a matching interest.
func TestRationalesHaveNoHoles(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	p := PersonaProfile{Followers: 300, Interests: []string{"chess"}}
	for i := 0; i < 50; i++ {
		for _, behavior := range []string{"lurker", "critic"} {
			p.Behavior = behavior
			for _, debate := range []bool{false, true} {
				c := ContentProfile{Debate: debate}
				for _, s := range []string{shareWhy(p, nil, r), commentWhy(p, c, nil, r), shareWhy(p, []string{"chess"}, r), commentWhy(p, c, []string{"chess"}, r)} {
					for _, bad := range []string{"the this", "my my", "  ", " ,", "about the  "} {
						if strings.Contains(s, bad) {
							t.Errorf("%q contains %q", s, bad)
						}
					}
				}
			}
		}
	}
}
