package main

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"testing"
)

// Scenario test runs: each scenario is simulated many times with different job IDs
// (i.e. different random seeds). A context-driven model must show (a) large
// differences between scenarios and (b) small spread within a scenario.

type scenario struct {
	name        string
	demographic string
	content     string
}

var scenarios = []scenario{
	{
		name:        "strong hook, matched audience",
		demographic: "Gen Z tech founders & SaaS product managers looking for viral growth strategies",
		content:     "Revealing the exact 3-step distribution playbook we used to scale an AI tool from 0 to 50k MRR without paid ads. Hook: Stop building features nobody asked for.",
	},
	{
		name:        "weak hook, matched audience",
		demographic: "Gen Z tech founders & SaaS product managers looking for viral growth strategies",
		content:     "Some thoughts on our startup and the things we have been doing lately with our product and the team over the last while.",
	},
	{
		name:        "strong hook, wrong audience",
		demographic: "Retirees interested in gardening, books and home decor",
		content:     "Revealing the exact 3-step distribution playbook we used to scale an AI tool from 0 to 50k MRR without paid ads. Hook: Stop building features nobody asked for.",
	},
	{
		name:        "fitness hook, fitness audience",
		demographic: "Millennial gym-goers into strength training and nutrition",
		content:     "Why 90% of people never build muscle after 30: the 3 workout mistakes you're making with protein and recovery.",
	},
	{
		name:        "salesy spam",
		demographic: "Millennial gym-goers into strength training and nutrition",
		content:     "BUY NOW!!! Limited offer on our supplement, link in bio, click to subscribe!!!",
	},
	{
		name:        "broad-appeal hook",
		demographic: "Gen Z students into personal finance",
		content:     "Why nobody tells you this: 5 money mistakes that cost students $10k a year — from food delivery to gym memberships, travel and gaming subscriptions. Which one are you making?",
	},
	{
		name:        "vague audience",
		demographic: "everyone",
		content:     "Why 90% of people never build muscle after 30: the 3 workout mistakes you're making with protein and recovery.",
	},
	{
		name:        "empty content",
		demographic: "General audience",
		content:     "",
	},
}

type scenarioStats struct {
	mean, sd float64
	verdicts map[string]int
	sample   *SimulationResult
}

func runScenario(sc scenario, runs int) scenarioStats {
	w := &Worker{concurrency: 8}
	st := scenarioStats{verdicts: map[string]int{}}
	vals := make([]float64, 0, runs)
	for i := 0; i < runs; i++ {
		res := w.ProcessSimulation(SimulationJob{
			JobID:              fmt.Sprintf("run-%d", i),
			Demographic:        sc.demographic,
			ContentDescription: sc.content,
			Population:         150,
			Boost:              6,
		})
		vals = append(vals, float64(res.Virality))
		st.verdicts[res.Verdict.Head]++
		if st.sample == nil {
			st.sample = res
		}
	}
	for _, v := range vals {
		st.mean += v
	}
	st.mean /= float64(runs)
	for _, v := range vals {
		st.sd += (v - st.mean) * (v - st.mean)
	}
	st.sd = math.Sqrt(st.sd / float64(runs))
	return st
}

func formatVerdicts(m map[string]int) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return m[keys[i]] > m[keys[j]] })
	parts := []string{}
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s ×%d", strings.TrimSuffix(k, "."), m[k]))
	}
	return strings.Join(parts, ", ")
}

// TestScenarioReport prints the scenario table. Run with: go test -run Scenario -v
func TestScenarioReport(t *testing.T) {
	const runs = 30
	for _, sc := range scenarios {
		st := runScenario(sc, runs)
		t.Logf("%-32s virality %5.1f ± %4.1f | %s", sc.name, st.mean, st.sd, formatVerdicts(st.verdicts))
	}
}

// TestScenarioOutcomes locks in that predictions follow the context, not the dice.
func TestScenarioOutcomes(t *testing.T) {
	const runs = 10
	want := map[string]string{
		"strong hook, matched audience":  "Niche hit.",
		"weak hook, matched audience":    "Weak hook.",
		"strong hook, wrong audience":    "Audience mismatch.",
		"fitness hook, fitness audience": "Niche hit.",
		"salesy spam":                    "Weak hook.",
		"broad-appeal hook":              "Viral hit.",
		"empty content":                  "Weak hook.",
	}
	mean := map[string]float64{}
	for _, sc := range scenarios {
		st := runScenario(sc, runs)
		mean[sc.name] = st.mean
		if w, ok := want[sc.name]; ok && st.verdicts[w] < runs*8/10 {
			t.Errorf("%s: want %q in ≥80%% of runs, got %s", sc.name, w, formatVerdicts(st.verdicts))
		}
		if st.sd > 12 {
			t.Errorf("%s: virality too noisy across seeds (sd %.1f)", sc.name, st.sd)
		}
	}
	// Same audience: a strong hook must clearly beat a weak one and spam.
	better := [][2]string{
		{"strong hook, matched audience", "weak hook, matched audience"},
		{"strong hook, matched audience", "strong hook, wrong audience"},
		{"fitness hook, fitness audience", "salesy spam"},
		{"fitness hook, fitness audience", "vague audience"},
	}
	for _, p := range better {
		if mean[p[0]] < mean[p[1]]+20 {
			t.Errorf("%q (%.1f) should beat %q (%.1f) by ≥20 points", p[0], mean[p[0]], p[1], mean[p[1]])
		}
	}
}
