// persona.go
// Generates diverse AI persona profiles.
// Each persona gets a unique identity that shapes how Claude responds.
// The more specific the persona, the more realistic the simulation.

package main

import (
	"fmt"
	"math/rand"
)

var names = []string{
	"Priya Nataraj", "Marcus Webb", "Sofia Mendez", "Liam Chen",
	"Ayesha Raza", "Jake Torres", "Zara Osei", "Rohan Kapoor",
	"Amara Diallo", "Nadia Kovac", "Omar Farooq", "Mei Lin",
	"Carlos Vega", "Ines Dubois", "Dev Sharma", "Kenji Tanaka",
	"Fatima Al-Hassan", "Lucas Borges", "Yuki Sato", "Rosario Vega",
	"Elena Petrov", "Sam Okafor", "Aria Patel", "Ben Nakamura",
}

var interests = []string{
	"growth hacking", "no-code tools", "founder communities",
	"revenue metrics", "B2B SaaS", "product design", "AI tools",
	"cold outreach", "content marketing", "startup ops",
	"dev tools", "fintech", "edtech", "creator economy",
	"fitness", "fashion", "gaming", "travel", "food", "music",
}

var locations = []string{
	"Bangalore, India", "New York, US", "London, UK",
	"São Paulo, BR", "Berlin, DE", "Lagos, NG",
	"Dubai, UAE", "Seoul, KR", "Sydney, AU", "Toronto, CA",
}

var behaviors = []string{
	"early adopter", "lurker", "commenter",
	"sharer", "observer", "influencer",
}

var genders = []string{"male", "female", "non-binary"}

// GeneratePersonas creates n diverse persona profiles.
// inTargetRatio controls what % are in the target demographic.
func GeneratePersonas(n int, inTargetRatio float64) []PersonaProfile {
	personas := make([]PersonaProfile, n)

	for i := 0; i < n; i++ {
		inTarget := rand.Float64() < inTargetRatio

		// Pick 2 unique interests
		i1 := interests[rand.Intn(len(interests))]
		i2 := interests[rand.Intn(len(interests))]
		for i2 == i1 {
			i2 = interests[rand.Intn(len(interests))]
		}

		personas[i] = PersonaProfile{
			ID:        i + 1,
			Name:      fmt.Sprintf("%s", names[i%len(names)]),
			Age:       22 + rand.Intn(35),
			Gender:    genders[rand.Intn(len(genders))],
			Location:  locations[rand.Intn(len(locations))],
			Followers: 10 + rand.Intn(9000),
			Interests: []string{i1, i2},
			Behavior:  behaviors[rand.Intn(len(behaviors))],
			InTarget:  inTarget,
		}
	}

	return personas
}