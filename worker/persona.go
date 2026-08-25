// persona.go
// Generates diverse AI persona profiles with region-specific names and correct gender assignment.

package main

import (
	"math/rand"
	"strings"
)

// nameGender maps every name in our pools to its correct gender.
var nameGender = map[string]string{
	// India
	"Priya Nataraj": "female", "Rohan Kapoor": "male", "Ananya Iyer": "female",
	"Vikram Malhotra": "male", "Divya Menon": "female", "Arjun Sharma": "male",
	"Meera Pillai": "female", "Karthik Subramanian": "male", "Sneha Patel": "female",
	"Rahul Gupta": "male", "Pooja Reddy": "female", "Aditya Verma": "male",
	"Nandini Rao": "female", "Siddharth Joshi": "male", "Deepika Nair": "female",
	"Manish Agarwal": "male", "Swati Singh": "female", "Nikhil Bhatia": "male",
	"Ishaan Chaudhary": "male", "Riya Desai": "female", "Harsh Dubey": "male",
	"Madhav Verma": "male", "Ram Sharma": "male", "Aria Patel": "female",
	"Dev Sharma": "male",
	// US
	"Marcus Webb": "male", "Jake Torres": "male", "Emily Carter": "female",
	"Tyler Johnson": "male", "Megan Williams": "female", "Brandon Scott": "male",
	"Ashley Davis": "female", "Jordan Mitchell": "non-binary", "Chloe Anderson": "female",
	"Ryan Thompson": "male", "Brittany Moore": "female", "Derek Harris": "male",
	"Lauren Wilson": "female", "Kyle Martin": "male", "Stephanie Taylor": "female",
	"Nathan Lewis": "male", "Kayla Brown": "female", "Zach White": "male",
	"Amanda Clark": "female", "Justin Hall": "male",
	// UK
	"Oliver Smith": "male", "Charlotte Davies": "female", "Harry Evans": "male",
	"Amelia Jones": "female", "George Williams": "male", "Poppy Brown": "female",
	"James Taylor": "male", "Isabella Wilson": "female", "William Thomas": "male",
	"Grace Roberts": "female", "Noah Johnson": "male", "Lily Robinson": "female",
	"Jack White": "male", "Sophie Walker": "female", "Ethan Lewis": "male",
	"Emily Hall": "female",
	// Brazil
	"Lucas Borges": "male", "Rosario Vega": "female", "Ana Lima": "female",
	"Gabriel Costa": "male", "Juliana Alves": "female", "Pedro Oliveira": "male",
	"Fernanda Santos": "female", "Mateus Sousa": "male", "Carolina Ferreira": "female",
	"Bruno Pereira": "male", "Larissa Rodrigues": "female", "Thiago Carvalho": "male",
	// Germany
	"Maximilian Müller": "male", "Lena Schmidt": "female", "Felix Wagner": "male",
	"Anna Becker": "female", "Jonas Fischer": "male", "Laura Hoffmann": "female",
	"Lukas Weber": "male", "Sarah Koch": "female", "Tobias Richter": "male",
	"Julia Schäfer": "female", "Paul Bauer": "male", "Marie Zimmermann": "female",
	// China
	"Mei Lin": "female", "Wei Zhang": "male", "Xiao Chen": "male",
	"Fang Liu": "female", "Jing Wang": "female", "Hao Yang": "male",
	"Li Na": "female", "Ming Zhao": "male", "Ling Huang": "female",
	"Chao Xu": "male", "Yan Zhou": "female", "Rui Zhu": "male",
	// Japan
	"Kenji Tanaka": "male", "Yuki Sato": "female", "Ben Nakamura": "male",
	"Hana Suzuki": "female", "Ren Watanabe": "male", "Aoi Ito": "female",
	// South Korea
	"Ji-woo Kim": "female", "Min-jun Lee": "male", "Seo-yeon Park": "female",
	"Jun-seo Choi": "male", "Ha-eun Jung": "female",
	// Israel
	"Yael Cohen": "female", "Avi Levi": "male", "Noa Katz": "female",
	"Omer Shapiro": "male", "Maya Friedman": "female",
	// UAE
	"Fatima Al-Hassan": "female", "Omar Farooq": "male", "Ayesha Raza": "female",
	"Khalid Al-Mansoori": "male", "Sara Al-Rashidi": "female",
	// Australia
	"Liam Chen": "male", "Chloe Thompson": "female", "Noah Williams": "male",
	"Ruby Harris": "female", "Ethan Martin": "male",
	// Canada
	"Sophie Tremblay": "female", "Liam MacDonald": "male", "Emma Gagnon": "female",
	"Noah Okafor": "male", "Olivia Singh": "female",
	// France
	"Ines Dubois": "female", "Clara Fontaine": "female", "Hugo Martin": "male",
	"Camille Bernard": "female", "Louis Petit": "male",
	// Russia
	"Elena Petrov": "female", "Alexei Volkov": "male", "Natasha Ivanova": "female",
	"Dmitri Sokolov": "male", "Olga Popova": "female",
	// Mexico
	"Carlos Vega": "male", "Sofia Ramirez": "female", "Diego Morales": "male",
	"Valentina Torres": "female", "Andrés Jiménez": "male",
	// South Africa
	"Zara Osei": "female", "Sam Okafor": "male", "Thabo Ndlovu": "male",
	"Amara Diallo": "female", "Sipho Dlamini": "male",
	// Singapore
	"Ryan Tan": "male", "Michelle Lee": "female", "David Lim": "male",
	"Priya Nair": "female", "Kevin Wong": "male",
	// Fallback / neutral
	"Alex Rivera": "non-binary", "Morgan Ellis": "female", "Jamie Chen": "non-binary",
	"Casey Park": "non-binary", "Avery Kim": "female", "Taylor Nguyen": "non-binary",
	"Jordan Patel": "non-binary", "Sam Singh": "male",
}

// namesByRegion maps a location keyword to culturally appropriate names.
var namesByRegion = map[string][]string{
	"India": {
		"Priya Nataraj", "Rohan Kapoor", "Ananya Iyer", "Vikram Malhotra",
		"Divya Menon", "Arjun Sharma", "Meera Pillai", "Karthik Subramanian",
		"Sneha Patel", "Rahul Gupta", "Pooja Reddy", "Aditya Verma",
		"Nandini Rao", "Siddharth Joshi", "Deepika Nair", "Manish Agarwal",
		"Swati Singh", "Nikhil Bhatia", "Ishaan Chaudhary", "Riya Desai",
		"Harsh Dubey", "Madhav Verma", "Ram Sharma", "Aria Patel", "Dev Sharma",
	},
	"US": {
		"Marcus Webb", "Jake Torres", "Emily Carter", "Tyler Johnson",
		"Megan Williams", "Brandon Scott", "Ashley Davis", "Jordan Mitchell",
		"Chloe Anderson", "Ryan Thompson", "Brittany Moore", "Derek Harris",
		"Lauren Wilson", "Kyle Martin", "Stephanie Taylor", "Nathan Lewis",
		"Kayla Brown", "Zach White", "Amanda Clark", "Justin Hall",
	},
	"UK": {
		"Oliver Smith", "Charlotte Davies", "Harry Evans", "Amelia Jones",
		"George Williams", "Poppy Brown", "James Taylor", "Isabella Wilson",
		"William Thomas", "Grace Roberts", "Noah Johnson", "Lily Robinson",
		"Jack White", "Sophie Walker", "Ethan Lewis", "Emily Hall",
	},
	"BR": {
		"Lucas Borges", "Rosario Vega", "Ana Lima", "Gabriel Costa",
		"Juliana Alves", "Pedro Oliveira", "Fernanda Santos", "Mateus Sousa",
		"Carolina Ferreira", "Bruno Pereira", "Larissa Rodrigues", "Thiago Carvalho",
	},
	"DE": {
		"Maximilian Müller", "Lena Schmidt", "Felix Wagner", "Anna Becker",
		"Jonas Fischer", "Laura Hoffmann", "Lukas Weber", "Sarah Koch",
		"Tobias Richter", "Julia Schäfer", "Paul Bauer", "Marie Zimmermann",
	},
	"CN": {
		"Mei Lin", "Wei Zhang", "Xiao Chen", "Fang Liu",
		"Jing Wang", "Hao Yang", "Li Na", "Ming Zhao",
		"Ling Huang", "Chao Xu", "Yan Zhou", "Rui Zhu",
	},
	"JP": {"Kenji Tanaka", "Yuki Sato", "Ben Nakamura", "Hana Suzuki", "Ren Watanabe", "Aoi Ito"},
	"KR": {"Ji-woo Kim", "Min-jun Lee", "Seo-yeon Park", "Jun-seo Choi", "Ha-eun Jung"},
	"IL": {"Yael Cohen", "Avi Levi", "Noa Katz", "Omer Shapiro", "Maya Friedman"},
	"UAE": {"Fatima Al-Hassan", "Omar Farooq", "Ayesha Raza", "Khalid Al-Mansoori", "Sara Al-Rashidi"},
	"AU": {"Liam Chen", "Chloe Thompson", "Noah Williams", "Ruby Harris", "Ethan Martin"},
	"CA": {"Sophie Tremblay", "Liam MacDonald", "Emma Gagnon", "Noah Okafor", "Olivia Singh"},
	"FR": {"Ines Dubois", "Clara Fontaine", "Hugo Martin", "Camille Bernard", "Louis Petit"},
	"RU": {"Elena Petrov", "Alexei Volkov", "Natasha Ivanova", "Dmitri Sokolov", "Olga Popova"},
	"MX": {"Carlos Vega", "Sofia Ramirez", "Diego Morales", "Valentina Torres", "Andrés Jiménez"},
	"ZA": {"Zara Osei", "Sam Okafor", "Thabo Ndlovu", "Amara Diallo", "Sipho Dlamini"},
	"SG": {"Ryan Tan", "Michelle Lee", "David Lim", "Priya Nair", "Kevin Wong"},
}

var fallbackNames = []string{
	"Alex Rivera", "Morgan Ellis", "Jamie Chen", "Casey Park",
	"Avery Kim", "Taylor Nguyen", "Jordan Patel", "Sam Singh",
}

func getRegionKey(location string) string {
	for key := range namesByRegion {
		if strings.HasSuffix(location, ", "+key) || strings.HasSuffix(location, " "+key) {
			return key
		}
	}
	for key := range namesByRegion {
		if strings.Contains(location, key) {
			return key
		}
	}
	return ""
}

func pickRegionalName(location string, r *rand.Rand) string {
	key := getRegionKey(location)
	pool, ok := namesByRegion[key]
	if !ok || len(pool) == 0 {
		pool = fallbackNames
	}
	return pool[r.Intn(len(pool))]
}

// genderForName returns the correct gender for a given name.
// Falls back to "non-binary" for unknown names.
func genderForName(name string) string {
	if g, ok := nameGender[name]; ok {
		return g
	}
	return "non-binary"
}

var interests = []string{
	"growth hacking", "no-code tools", "founder communities",
	"revenue metrics", "B2B SaaS", "product design", "AI tools",
	"cold outreach", "content marketing", "startup ops",
	"dev tools", "fintech", "edtech", "creator economy",
	"fitness", "fashion", "gaming", "travel", "food", "music",
	"movies", "books", "sports", "parenting", "mental health",
	"personal finance", "cryptocurrency", "photography", "art",
	"DIY", "home decor", "gardening", "pets", "cars", "technology",
	"science", "history", "comedy", "memes", "cooking", "baking",
	"yoga", "meditation", "hiking", "cycling", "running",
	"entrepreneurship", "investing", "real estate",
	"productivity", "leadership", "remote work", "self-care",
}

var locations = []string{
	"Bangalore, India", "Mumbai, India", "Delhi, India",
	"Kolkata, India", "Chennai, India", "Hyderabad, India",
	"Pune, India", "Ahmedabad, India", "Gurugram, India",
	"New York, US", "Los Angeles, US", "Chicago, US",
	"Houston, US", "Miami, US", "Atlanta, US",
	"London, UK",
	"São Paulo, BR",
	"Berlin, DE",
	"Tel Aviv, IL",
	"Shanghai, CN",
	"Dubai, UAE",
	"Seoul, KR",
	"Sydney, AU",
	"Toronto, CA",
	"Paris, FR",
	"Moscow, RU",
	"Mexico City, MX",
	"Cape Town, ZA",
	"Singapore, SG",
}

var behaviors = []string{
	"early adopter", "lurker", "commenter", "sharer",
	"observer", "influencer", "skeptic", "enthusiast",
	"critic", "networker", "content creator", "trend follower",
	"trend setter", "analytical thinker", "creative problem solver",
	"thought leader", "mentor", "risk taker", "pragmatist",
}

// GeneratePersonas creates n diverse persona profiles with region-matched names and correct genders.
func GeneratePersonas(n int, inTargetRatio float64, r *rand.Rand) []PersonaProfile {
	personas := make([]PersonaProfile, n)
	for i := 0; i < n; i++ {
		inTarget := r.Float64() < inTargetRatio
		location := locations[r.Intn(len(locations))]
		name := pickRegionalName(location, r)
		// Gender derived from the name — never random
		gender := genderForName(name)

		i1 := interests[r.Intn(len(interests))]
		i2 := interests[r.Intn(len(interests))]
		for i2 == i1 {
			i2 = interests[r.Intn(len(interests))]
		}

		personas[i] = PersonaProfile{
			ID:        i + 1,
			Name:      name,
			Age:       18 + r.Intn(42), // 18–59
			Gender:    gender,
			Location:  location,
			Followers: 50 + r.Intn(9950),
			Interests: []string{i1, i2},
			Behavior:  behaviors[r.Intn(len(behaviors))],
			InTarget:  inTarget,
			Shown:     false,
		}
	}
	return personas
}
