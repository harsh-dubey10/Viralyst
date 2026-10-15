package main

import (
	"math/rand"
	"strings"
	"testing"
)

func TestPersonaNamesAreUnique(t *testing.T) {
	// Audience concentrated in one small region is the worst case: 85% of in-target
	// personas land there.
	for _, demo := range []string{"K-drama fans in Korea", "Gen Z in India", "everyone"} {
		for seed := int64(0); seed < 20; seed++ {
			ps := GeneratePersonas(250, ParseAudience(demo), 0.5, rand.New(rand.NewSource(seed)))
			seen := map[string]int{}
			for _, p := range ps {
				seen[p.Name]++
				if seen[p.Name] == 2 {
					t.Errorf("%q seed %d: %q appears more than once", demo, seed, p.Name)
				}
			}
		}
	}
}

func TestEveryLocationHasAnAmpleNamePool(t *testing.T) {
	for _, loc := range locations {
		region := loc[strings.LastIndex(loc, ", ")+2:]
		np, ok := namePools[region]
		if !ok {
			t.Errorf("%s has no name pool (would borrow US names)", loc)
			continue
		}
		// Each gender alone must cover a full 250-person run from one region.
		for _, firsts := range [][]string{np.female, np.male} {
			if n := len(firsts) * len(np.surnames); n < 150 {
				t.Errorf("%s: only %d combinations per gender", region, n)
			}
		}
		if np.femaleSurnames != nil && len(np.femaleSurnames) != len(np.surnames) {
			t.Errorf("%s: gendered surname lists differ in length", region)
		}
	}
}

func TestGenderMatchesFirstName(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	for region, np := range namePools {
		in := func(list []string, first string) bool {
			for _, f := range list {
				if strings.HasPrefix(first, f+" ") {
					return true
				}
			}
			return false
		}
		for i := 0; i < 200; i++ {
			name, gender := np.draw(r)
			want := map[string][]string{"female": np.female, "male": np.male, "non-binary": np.neutral}[gender]
			if !in(want, name) {
				t.Fatalf("%s: %q drawn as %s", region, name, gender)
			}
			if region == "RU" && gender == "female" && !strings.HasSuffix(name, "a") {
				t.Errorf("RU: %q should carry the feminine surname", name)
			}
		}
	}
}

func TestUniqueNameFallsBackToMiddleInitial(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	used := map[string]bool{}
	tiny := namePools["JP"]
	defer func() { namePools["JP"] = tiny }()
	namePools["JP"] = namePool{female: []string{"Yuki"}, male: []string{"Ren"}, surnames: []string{"Sato"}}
	for i := 0; i < 6; i++ {
		uniqueName("Tokyo, JP", used, r)
	}
	if len(used) != 6 {
		t.Fatalf("got %d distinct names, want 6: %v", len(used), used)
	}
}
