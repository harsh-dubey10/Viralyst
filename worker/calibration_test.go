package main

import (
	"fmt"
	"testing"
)

// Calibration checks for the offline model, from cases whose outcome we know or that
// the app itself showcases. Before recalibration every one of these scored 0–21 and
// was labelled "Weak hook", including a post that really reached 1,038× its followers.

func simulateMean(t *testing.T, niche, demographic, hook string, runs int) (float64, map[string]int) {
	t.Helper()
	w := &Worker{concurrency: 4}
	sum, verdicts := 0, map[string]int{}
	for i := 0; i < runs; i++ {
		r := w.ProcessSimulation(SimulationJob{JobID: fmt.Sprintf("cal-%d", i), Niche: niche, Demographic: demographic, ContentDescription: hook, Population: 100, Boost: 6})
		sum += r.Virality
		verdicts[r.Verdict.Head]++
	}
	return float64(sum) / float64(runs), verdicts
}

func TestKnownHitBeatsBlandHook(t *testing.T) {
	// eval/posts.csv: 3.4M views on 3,275 followers.
	hit, hv := simulateMean(t, "K-drama", "K-drama fans, 18-35",
		`Spoken: "I just finished watching the highest rated K-drama on IMDb" On-screen text: "I just finished watching the highest-rated drama" Scene: A split-screen video shows a woman talking into a microphone while a clip from a Korean drama plays.`, 6)
	bland, _ := simulateMean(t, "cooking", "busy parents", "Here is a pasta recipe I made today", 6)
	if hit < 40 || hv["Niche hit."]+hv["Viral hit."] < 4 {
		t.Errorf("known hit: virality %.0f, verdicts %v — want ≥40 and mostly a hit", hit, hv)
	}
	if hit < bland+20 {
		t.Errorf("known hit (%.0f) should beat a bland hook (%.0f) by ≥20", hit, bland)
	}
}

func TestShowcaseExamplesAreNotWeak(t *testing.T) {
	// The examples offered in the UI (frontend/src/constants.ts) are deliberately good hooks.
	for _, ex := range [][3]string{
		{"cooking", "Busy parents in the US, 28–45, who cook dinner most nights", "This 10-minute garlic butter pasta uses only 4 ingredients — and it beats every restaurant version I've had."},
		{"travel", "Budget backpackers in their 20s planning trips to Southeast Asia", "I spent 30 days in Vietnam on $600. Here's exactly where every dollar went."},
		{"chess", "Club players rated 1200–1800", "This 5-move chess trap wins the queen against the Sicilian — and 80% of players fall for it."},
	} {
		mean, v := simulateMean(t, ex[0], ex[1], ex[2], 6)
		if v["Weak hook."]+v["Weak signal."] > 2 || mean < 35 {
			t.Errorf("%s example: virality %.0f, verdicts %v — a strong hook shouldn't read as weak", ex[0], mean, v)
		}
	}
}

func TestNicheHookWithoutTopicWordsIsOnNiche(t *testing.T) {
	sc := newSimContext(SimulationJob{Niche: "chess", Demographic: "club players", ContentDescription: "The 3 opening traps that won me 200 games in a row"}, VideoInsight{}, nil)
	if len(sc.content.Topics) != 1 || sc.content.Topics[0] != "chess" {
		t.Fatalf("content topics = %v: a hook posted in a niche is about that niche unless it says otherwise", sc.content.Topics)
	}
	for _, issue := range sc.content.Issues {
		if issue == noTopicIssue {
			t.Errorf("still reports %q", noTopicIssue)
		}
	}
}

func TestSpokenLineIsJudgedNotTheScene(t *testing.T) {
	c := AnalyzeContent(SimulationJob{ContentDescription: `Spoken: "Stop doing this with your knives" Scene: A long description of a kitchen with a wooden counter, a window, plants, a dog walking past, and soft afternoon light falling on the chopping board.`}, nil)
	for _, issue := range c.Issues {
		if len(issue) > 16 && issue[:16] == "Opening sentence" {
			t.Errorf("a short spoken line was penalised for the scene's length: %q", issue)
		}
	}
}

func TestSeedScalesWithPopulation(t *testing.T) {
	w := &Worker{concurrency: 4}
	seeds := func(pop int) int {
		r := w.ProcessSimulation(SimulationJob{JobID: "seed", Niche: "cooking", ContentDescription: "Pasta in 10 minutes", Population: pop, Boost: 6})
		n := 0
		for _, p := range r.Personas {
			if p.Round == 1 {
				n++
			}
		}
		return n
	}
	if s100, s250 := seeds(100), seeds(250); s100 != 25 || s250 != 63 {
		t.Errorf("seed sizes = %d (pop 100), %d (pop 250); want 25 and 63 (frontend seedSize mirrors this)", s100, s250)
	}
}
