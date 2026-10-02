package main

import (
	"os"
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
