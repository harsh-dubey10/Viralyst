package main

import (
	"fmt"
	"math/rand"
	"testing"
)

func TestAssignByIntentFollowsWhatPersonasSaid(t *testing.T) {
	sc := newSimContext(SimulationJob{Niche: "chess", ContentDescription: "This 5-move chess trap wins the queen — and 80% of players fall for it."}, VideoInsight{}, nil)
	rel := func(v int) *int { return &v }
	var items []llmItem
	kind := map[int]string{}
	for i := 0; i < 40; i++ {
		p := PersonaProfile{ID: i + 1, Behavior: "observer", Followers: 200, Interests: []string{"chess"}, InTarget: true, Shown: true}
		var d groqPersonaDecision
		switch i % 4 {
		case 0:
			d = groqPersonaDecision{Relevant: rel(10), Watch: 95, Like: 9, Comment: 3, Share: 9, Why: "Sending this to my chess club."}
			kind[p.ID] = "sharer"
		case 1:
			d = groqPersonaDecision{Relevant: rel(9), Watch: 80, Like: 8, Comment: 1, Share: 1, Why: "Nice trap, liked it."}
			kind[p.ID] = "liker"
		case 2:
			d = groqPersonaDecision{Relevant: rel(6), Watch: 45, Like: 3, Comment: 0, Share: 0, Why: "Watched a bit."}
			kind[p.ID] = "meh"
		default:
			d = groqPersonaDecision{Relevant: rel(2), Watch: 5, Like: 0, Comment: 0, Share: 0, Why: "Not for me."}
			kind[p.ID] = "swiper"
		}
		items = append(items, llmItem{persona: p, d: d, seed: int64(1000 + i)})
	}

	// The counts must be exactly what the calibrated draws produce.
	want := map[string]int{}
	for _, it := range items {
		a, w := reaction(it.persona, sc.content, llmRelevance(it.d), llmTilt(it.d), rand.New(rand.NewSource(it.seed)))
		want[category(a, w)]++
	}
	out := assignByIntent(sc, items)
	got := map[string]int{}
	for _, r := range out {
		got[category(r.Action, r.Watch)]++
	}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("outcome counts %v, calibrated draws %v", got, want)
	}

	// Who: shares go to the eager sharers first, swipes to those who said they'd swipe.
	agree := 0
	for _, r := range out {
		k := kind[r.PersonaID]
		if r.Action == "share" && k != "sharer" && want[catShare] <= 10 {
			t.Errorf("persona %d (%s) got a share slot while eager sharers were available", r.PersonaID, k)
		}
		if category(r.Action, r.Watch) == catSwipe && k != "swiper" && want[catSwipe] <= 10 {
			t.Errorf("persona %d (%s) was made to swipe while swipers were available", r.PersonaID, k)
		}
		said := llmImpliedAction(items[r.PersonaID-1].d)
		if r.Action == said {
			agree++
			if r.Why != items[r.PersonaID-1].d.Why {
				t.Errorf("persona %d kept its action but lost its own words: %q", r.PersonaID, r.Why)
			}
		}
	}
	t.Logf("counts %v; %d/40 personas did what they said", got, agree)
	if agree < 20 {
		t.Errorf("only %d/40 personas did what they said", agree)
	}
}
