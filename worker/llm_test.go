package main

import (
	"os"
	"strings"
	"testing"
)

// TestLLMScenarios is a live test run against Groq. It is skipped unless enabled:
//
//	VIRALYST_LLM_TEST=1 go test -run LLMScenarios -v
func TestLLMScenarios(t *testing.T) {
	if os.Getenv("VIRALYST_LLM_TEST") != "1" {
		t.Skip("set VIRALYST_LLM_TEST=1 to run live LLM scenarios")
	}
	if !hasGroqAPIKeys() {
		t.Skip("no Groq API keys configured")
	}
	w := &Worker{concurrency: 4}
	for _, name := range []string{"strong hook, matched audience", "strong hook, wrong audience", "salesy spam"} {
		var sc scenario
		for _, s := range scenarios {
			if s.name == name {
				sc = s
			}
		}
		res := w.ProcessSimulation(SimulationJob{JobID: "llm-" + name, Demographic: sc.demographic, ContentDescription: sc.content, Population: 50, Boost: 3})
		a := res.Analysis
		t.Logf("== %s → %s virality=%d engine=%s inEng=%d%% outEng=%d%% relSkip=%d%%", name, res.Verdict.Head, res.Virality, a.Engine, a.InTargetEngagement, a.OutTargetEngagement, a.RelevantSkipRate)
		shown := 0
		for _, p := range res.Personas {
			if p.Shown && p.Round == 1 && shown < 3 {
				shown++
				t.Logf("   %s (%v, in_target=%v): %s — %q", p.Name, p.Interests, p.InTarget, p.Action, p.Why)
			}
		}
	}
}

// llmCalibrationPosts span known outcomes: a real hit, a decent niche post, a bland
// hook, and a mismatched audience. The LLM engine should rank them in that order.
var llmCalibrationPosts = []struct{ name, niche, demographic, hook, rest string }{
	{"known hit (3.4M views)", "K-drama", "K-drama fans, 18-35",
		`Spoken: "I just finished watching the highest rated K-drama on IMDb"` + "\n" + `On-screen text: "I just finished watching the highest-rated drama"` + "\nScene: A split-screen video shows a woman talking into a microphone while a clip from a Korean drama plays.", ""},
	{"hackathon story", "AI tools", "",
		`Spoken: "to build a virality prediction simulator"` + "\n" + `On-screen text: "Using Guildly to make a winning hackathon project And we just built a virality prediction simulator"` + "\nScene: Two men wearing glasses stand in a room speaking to the camera while a computer interface is shown.",
		"A screen recording of the data visualization tool, then the creator explains how it works with an enthusiastic, proud tone. The payoff: it was his first hackathon and he won a Mac Mini."},
	{"bland hook", "cooking", "busy parents", "Here is a pasta recipe I made today", ""},
	{"wrong audience", "gardening", "retirees into gardening", "Stop building SaaS features nobody asked for: our 3-step 50k MRR playbook", ""},
}

// TestLLMCalibration prints the LLM engine's verdict for each calibration post.
//
//	VIRALYST_LLM_TEST=1 go test -run LLMCalibration -timeout 30m -v   (VIRALYST_LLM_ONLY=bland runs one post)
func TestLLMCalibration(t *testing.T) {
	if os.Getenv("VIRALYST_LLM_TEST") != "1" {
		t.Skip("set VIRALYST_LLM_TEST=1 to run live LLM calibration")
	}
	if !hasGroqAPIKeys() {
		t.Skip("no Groq API keys configured")
	}
	w := &Worker{concurrency: 4}
	only := os.Getenv("VIRALYST_LLM_ONLY") // optional substring filter on post names
	for _, p := range llmCalibrationPosts {
		if only != "" && !strings.Contains(p.name, only) {
			continue
		}
		sc := newSimContext(SimulationJob{JobID: "cal-" + p.name, Niche: p.niche, Demographic: p.demographic, ContentDescription: p.hook, Population: 60, Boost: 6}, VideoInsight{Rest: p.rest}, readTopicsWithLLM)
		res := w.simulateOnce(sc, 42)
		a := res.Analysis
		t.Logf("%-24s virality=%3d %-18s engine=%s(%d%%) hook=%d inEng=%d%% outEng=%d%% relSkip=%d%% shares=%d",
			p.name, res.Virality, res.Verdict.Head, a.Engine, a.LLMShare, a.HookScore, a.InTargetEngagement, a.OutTargetEngagement, a.RelevantSkipRate, res.Breakdown.Shares)
	}
}
