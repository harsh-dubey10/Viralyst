// persona.go
// Generates diverse AI persona profiles with unique, region-specific names.

package main

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
)

// namePool holds one region's first names (by gender) and surnames. Names are composed
// first × surname, so each region has hundreds of distinct names, and GeneratePersonas
// never repeats one within a simulation. (Whole-name lists of 5–25 per region produced
// the same persona name several times in a 100-person run, since audience targeting
// puts most in-target personas in one or two regions.)
type namePool struct {
	female, male, neutral []string
	surnames              []string
	femaleSurnames        []string // only where surnames are gendered (Russian); else surnames is used
}

// Given name first, as the UI shows it, for every region.
var namePools = map[string]namePool{
	"India": {
		female:   []string{"Priya", "Ananya", "Divya", "Meera", "Sneha", "Pooja", "Nandini", "Swati", "Deepika", "Riya", "Kavya", "Ishita", "Aditi", "Neha"},
		male:     []string{"Rohan", "Vikram", "Arjun", "Karthik", "Rahul", "Aditya", "Siddharth", "Manish", "Nikhil", "Ishaan", "Harsh", "Madhav", "Dev", "Varun"},
		surnames: []string{"Sharma", "Kapoor", "Iyer", "Malhotra", "Menon", "Pillai", "Subramanian", "Patel", "Gupta", "Reddy", "Verma", "Rao", "Joshi", "Nair", "Agarwal", "Singh", "Bhatia", "Chaudhary", "Desai", "Dubey", "Mehta", "Banerjee", "Mukherjee", "Kulkarni"},
	},
	"US": {
		female:   []string{"Emily", "Megan", "Ashley", "Chloe", "Brittany", "Lauren", "Stephanie", "Kayla", "Amanda", "Jessica", "Madison", "Hannah", "Olivia", "Destiny"},
		male:     []string{"Marcus", "Jake", "Tyler", "Brandon", "Ryan", "Derek", "Kyle", "Nathan", "Zach", "Justin", "Tyrone", "Carlos", "Ethan", "Logan"},
		neutral:  []string{"Jordan", "Taylor", "Alex", "Casey"},
		surnames: []string{"Webb", "Torres", "Carter", "Johnson", "Williams", "Scott", "Davis", "Mitchell", "Anderson", "Thompson", "Moore", "Harris", "Wilson", "Martin", "Lewis", "Brown", "White", "Clark", "Hall", "Nguyen", "Garcia", "Robinson", "Kim", "Jackson"},
	},
	"UK": {
		female:   []string{"Charlotte", "Amelia", "Poppy", "Isabella", "Grace", "Lily", "Sophie", "Emily", "Freya", "Isla", "Evie", "Ruby"},
		male:     []string{"Oliver", "Harry", "George", "James", "William", "Noah", "Jack", "Ethan", "Alfie", "Callum", "Rhys", "Arjun"},
		neutral:  []string{"Alex", "Sam"},
		surnames: []string{"Smith", "Davies", "Evans", "Jones", "Williams", "Brown", "Taylor", "Wilson", "Thomas", "Roberts", "Robinson", "Walker", "Hall", "Wright", "Hughes", "Patel", "Khan", "Murphy"},
	},
	"BR": {
		female:   []string{"Ana", "Mariana", "Juliana", "Beatriz", "Larissa", "Camila", "Fernanda", "Gabriela", "Letícia", "Rafaela", "Isabela", "Luana"},
		male:     []string{"Lucas", "Pedro", "Gabriel", "Mateus", "Rafael", "Thiago", "Felipe", "Bruno", "Gustavo", "Leonardo", "João", "Caio"},
		surnames: []string{"Silva", "Santos", "Oliveira", "Souza", "Lima", "Pereira", "Costa", "Ferreira", "Rodrigues", "Almeida", "Borges", "Carvalho", "Gomes", "Ribeiro", "Martins"},
	},
	"DE": {
		female:   []string{"Anna", "Lena", "Laura", "Julia", "Lea", "Sophie", "Hannah", "Marie", "Lisa", "Katharina", "Jana", "Clara"},
		male:     []string{"Lukas", "Leon", "Jonas", "Felix", "Maximilian", "Paul", "Tim", "Niklas", "Jan", "Moritz", "Tobias", "Florian"},
		surnames: []string{"Müller", "Schmidt", "Schneider", "Fischer", "Weber", "Meyer", "Wagner", "Becker", "Schulz", "Hoffmann", "Koch", "Richter", "Wolf", "Klein", "Neumann"},
	},
	"CN": {
		female:   []string{"Mei", "Xiao", "Fang", "Jing", "Na", "Ling", "Yan", "Xin", "Yue", "Ting", "Hui", "Qian"},
		male:     []string{"Wei", "Hao", "Ming", "Chao", "Rui", "Jun", "Lei", "Tao", "Bo", "Jie", "Peng", "Kai"},
		surnames: []string{"Lin", "Zhang", "Chen", "Liu", "Wang", "Yang", "Li", "Zhao", "Huang", "Xu", "Zhou", "Zhu", "Wu", "Sun", "Ma"},
	},
	"JP": {
		female:   []string{"Yuki", "Hana", "Aoi", "Sakura", "Yui", "Mio", "Rin", "Haruka", "Nanami", "Misaki", "Emi", "Saki"},
		male:     []string{"Kenji", "Ren", "Haruto", "Sota", "Yuto", "Takumi", "Daiki", "Kaito", "Riku", "Shota", "Kenta", "Hiroshi"},
		surnames: []string{"Tanaka", "Sato", "Nakamura", "Suzuki", "Watanabe", "Ito", "Takahashi", "Kobayashi", "Yamamoto", "Kato", "Yoshida", "Yamada", "Sasaki", "Matsumoto", "Inoue"},
	},
	"KR": {
		female:   []string{"Ji-woo", "Seo-yeon", "Ha-eun", "Su-bin", "Min-ji", "Ye-jin", "Da-eun", "Soo-ah", "Yu-na", "Eun-ji", "Hye-jin", "Na-yeon"},
		male:     []string{"Min-jun", "Jun-seo", "Do-yoon", "Seo-jun", "Ha-joon", "Ji-ho", "Hyun-woo", "Joon-young", "Tae-yang", "Woo-jin", "Sung-min", "Dong-hyun"},
		surnames: []string{"Kim", "Lee", "Park", "Choi", "Jung", "Kang", "Cho", "Yoon", "Jang", "Lim", "Han", "Shin", "Seo", "Kwon", "Hwang"},
	},
	"IL": {
		female:   []string{"Yael", "Noa", "Maya", "Tamar", "Shira", "Michal", "Talia", "Hila", "Neta", "Liat", "Rivka", "Dana"},
		male:     []string{"Avi", "Omer", "Itai", "Yonatan", "Daniel", "Eitan", "Noam", "Ariel", "Guy", "Tomer", "Yosef", "Ido"},
		surnames: []string{"Cohen", "Levi", "Katz", "Shapiro", "Friedman", "Mizrahi", "Peretz", "Biton", "Dahan", "Avraham", "Ben-David", "Goldberg", "Rosen", "Azoulay", "Klein"},
	},
	"UAE": {
		female:   []string{"Fatima", "Ayesha", "Sara", "Mariam", "Noura", "Hessa", "Layla", "Amna", "Shamma", "Reem", "Maitha", "Huda"},
		male:     []string{"Omar", "Khalid", "Ahmed", "Mohammed", "Rashid", "Saeed", "Hamdan", "Sultan", "Faisal", "Yousef", "Ali", "Hamad"},
		surnames: []string{"Al-Hassan", "Al-Mansoori", "Al-Rashidi", "Al-Nuaimi", "Al-Mazrouei", "Al-Ketbi", "Al-Shamsi", "Al-Hammadi", "Al-Falasi", "Al-Suwaidi", "Al-Marri", "Al-Zaabi", "Farooq", "Raza", "Khan"},
	},
	"AU": {
		female:   []string{"Chloe", "Ruby", "Isla", "Charlotte", "Mia", "Grace", "Zoe", "Matilda", "Sienna", "Ella", "Harper", "Olivia"},
		male:     []string{"Liam", "Noah", "Ethan", "Jack", "Oliver", "Lachlan", "Cooper", "Hamish", "Mitchell", "Riley", "Thomas", "Kai"},
		neutral:  []string{"Charlie", "Frankie"},
		surnames: []string{"Thompson", "Williams", "Harris", "Martin", "Smith", "Jones", "Brown", "Wilson", "Taylor", "Kelly", "Ryan", "Chen", "Nguyen", "Walker", "Campbell"},
	},
	"CA": {
		female:   []string{"Sophie", "Emma", "Olivia", "Chloé", "Charlotte", "Léa", "Ava", "Maya", "Zoe", "Amelia", "Gabrielle", "Priya"},
		male:     []string{"Liam", "Noah", "William", "Logan", "Félix", "Nathan", "Ethan", "Owen", "Jacob", "Samuel", "Mathis", "Arjun"},
		neutral:  []string{"Jordan", "Alex"},
		surnames: []string{"Tremblay", "MacDonald", "Gagnon", "Okafor", "Singh", "Roy", "Côté", "Bouchard", "Smith", "Brown", "Wilson", "Martin", "Li", "Campbell", "Leblanc"},
	},
	"FR": {
		female:   []string{"Inès", "Clara", "Camille", "Léa", "Manon", "Chloé", "Emma", "Jade", "Louise", "Zoé", "Juliette", "Lina"},
		male:     []string{"Hugo", "Louis", "Lucas", "Gabriel", "Arthur", "Jules", "Théo", "Nathan", "Raphaël", "Maël", "Adam", "Paul"},
		surnames: []string{"Dubois", "Fontaine", "Martin", "Bernard", "Petit", "Durand", "Leroy", "Moreau", "Simon", "Laurent", "Lefebvre", "Michel", "Garcia", "Roux", "Girard"},
	},
	"RU": {
		female:         []string{"Elena", "Natasha", "Olga", "Anastasia", "Maria", "Daria", "Ekaterina", "Svetlana", "Irina", "Polina", "Yulia", "Ksenia"},
		male:           []string{"Alexei", "Dmitri", "Ivan", "Sergei", "Mikhail", "Nikolai", "Andrei", "Pavel", "Artem", "Maxim", "Kirill", "Vladimir"},
		surnames:       []string{"Petrov", "Volkov", "Ivanov", "Sokolov", "Popov", "Smirnov", "Kuznetsov", "Lebedev", "Kozlov", "Novikov", "Morozov", "Fedorov", "Orlov", "Pavlov", "Egorov", "Belov"},
		femaleSurnames: []string{"Petrova", "Volkova", "Ivanova", "Sokolova", "Popova", "Smirnova", "Kuznetsova", "Lebedeva", "Kozlova", "Novikova", "Morozova", "Fedorova", "Orlova", "Pavlova", "Egorova", "Belova"},
	},
	"MX": {
		female:   []string{"Sofía", "Valentina", "María", "Fernanda", "Ximena", "Camila", "Regina", "Daniela", "Mariana", "Andrea", "Paola", "Lucía"},
		male:     []string{"Carlos", "Diego", "Andrés", "José", "Luis", "Santiago", "Mateo", "Emiliano", "Alejandro", "Jorge", "Miguel", "Ricardo"},
		surnames: []string{"Vega", "Ramírez", "Morales", "Torres", "Jiménez", "Hernández", "García", "López", "Martínez", "González", "Rodríguez", "Pérez", "Sánchez", "Flores", "Cruz"},
	},
	"ZA": {
		female:   []string{"Zanele", "Thandiwe", "Naledi", "Lerato", "Nomvula", "Amahle", "Palesa", "Refilwe", "Lindiwe", "Busisiwe", "Nandi", "Thuli"},
		male:     []string{"Thabo", "Sipho", "Bongani", "Mandla", "Themba", "Lwazi", "Kabelo", "Tshepo", "Sizwe", "Musa", "Jabu", "Mpho"},
		surnames: []string{"Ndlovu", "Dlamini", "Nkosi", "Mokoena", "Khumalo", "Mthembu", "Zulu", "Molefe", "Sithole", "Naidoo", "van der Merwe", "Botha", "Pillay", "Mahlangu", "Ngcobo"},
	},
	"SG": {
		female:   []string{"Michelle", "Rachel", "Hui Min", "Jia Hui", "Sarah", "Wei Ling", "Charmaine", "Grace", "Xin Yi", "Jasmine", "Shu Ting", "Cheryl"},
		male:     []string{"Ryan", "David", "Kevin", "Wei Jie", "Jun Hao", "Marcus", "Daniel", "Benjamin", "Darren", "Zhi Hao", "Jonathan", "Kai Wen"},
		surnames: []string{"Tan", "Lee", "Lim", "Ng", "Wong", "Ong", "Goh", "Teo", "Chua", "Koh", "Chan", "Low", "Yeo", "Sim", "Ho"},
	},
}

// poolFor returns the name pool for a "City, CODE" location (US names if unknown).
func poolFor(location string) namePool {
	region := location
	if i := strings.LastIndex(location, ", "); i >= 0 {
		region = location[i+2:]
	}
	if p, ok := namePools[region]; ok {
		return p
	}
	return namePools["US"]
}

// draw picks a gender, then a matching first name and a surname. Where the region has
// gender-neutral names, a small share of personas are non-binary.
func (np namePool) draw(r *rand.Rand) (name, gender string) {
	var first string
	surnames := np.surnames
	switch roll := r.Float64(); {
	case len(np.neutral) > 0 && roll < 0.04:
		first, gender = np.neutral[r.Intn(len(np.neutral))], "non-binary"
	case roll < 0.52:
		first, gender = np.female[r.Intn(len(np.female))], "female"
		if len(np.femaleSurnames) > 0 {
			surnames = np.femaleSurnames
		}
	default:
		first, gender = np.male[r.Intn(len(np.male))], "male"
	}
	return first + " " + surnames[r.Intn(len(surnames))], gender
}

// uniqueName draws a name for the location that isn't in used yet, and records it.
// Each region has 300+ combinations against at most 250 personas, so a free one is
// almost always found within a few draws; the middle initial is a last resort.
func uniqueName(location string, used map[string]bool, r *rand.Rand) (name, gender string) {
	pool := poolFor(location)
	for i := 0; i < 50; i++ {
		if name, gender = pool.draw(r); !used[name] {
			used[name] = true
			return name, gender
		}
	}
	first, last, _ := strings.Cut(name, " ")
	for c := 'A'; c <= 'Z'; c++ {
		if alt := fmt.Sprintf("%s %c. %s", first, c, last); !used[alt] {
			used[alt] = true
			return alt, gender
		}
	}
	return name, gender
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
	"Tokyo, JP",
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

// GeneratePersonas creates n personas. In-target personas are built from the parsed
// audience (its interests, age range and regions); out-of-target personas are drawn
// from everyone else, so "in target" means something about who the person is.
func GeneratePersonas(n int, audience AudienceProfile, inTargetRatio float64, r *rand.Rand) []PersonaProfile {
	audienceSet := map[string]bool{}
	for _, t := range audience.Topics {
		audienceSet[t] = true
	}
	others := []string{}
	for _, t := range allTopicNames {
		if !audienceSet[t] {
			others = append(others, t)
		}
	}
	regionLocations := []string{}
	for _, loc := range locations {
		for _, reg := range audience.Regions {
			if strings.HasSuffix(loc, ", "+reg) {
				regionLocations = append(regionLocations, loc)
			}
		}
	}

	personas := make([]PersonaProfile, n)
	usedNames := map[string]bool{}
	for i := 0; i < n; i++ {
		inTarget := r.Float64() < inTargetRatio

		location := locations[r.Intn(len(locations))]
		age := minPersonaAge + r.Intn(maxPersonaAge-minPersonaAge+1)
		var picked []string
		if inTarget {
			if len(regionLocations) > 0 && r.Float64() < 0.85 {
				location = regionLocations[r.Intn(len(regionLocations))]
			}
			age = audience.MinAge + r.Intn(audience.MaxAge-audience.MinAge+1)
			if len(audience.Topics) > 0 {
				// A creator's audience follows them for the niche: every in-target persona
				// cares about it, plus maybe one of the audience's other interests. (Drawing
				// from all audience topics left ~25% of "busy parents who cook" not caring
				// about cooking, which dragged every cooking post down.)
				primary := audience.Core
				if len(primary) == 0 {
					primary = audience.Topics
				}
				picked = pickDistinct(primary, 1, nil, r)
				if len(audience.Topics) > 1 && r.Float64() < 0.5 {
					picked = append(picked, pickDistinct(audience.Topics, 1, picked, r)...)
				}
			}
		} else {
			picked = pickDistinct(others, 1, nil, r)
		}
		// Fill to 3 interests with topics outside the audience's niche.
		picked = append(picked, pickDistinct(others, 3-len(picked), picked, r)...)

		name, gender := uniqueName(location, usedNames, r)
		personas[i] = PersonaProfile{
			ID:       i + 1,
			Name:     name,
			Age:      age,
			Gender:   gender, // matches the first name drawn
			Location: location,
			// Log-uniform 50–10,000: most accounts are small, a few are large.
			Followers: int(50 * math.Pow(200, r.Float64())),
			Interests: picked,
			Behavior:  behaviors[r.Intn(len(behaviors))],
			InTarget:  inTarget,
		}
	}
	return personas
}

// pickDistinct draws k distinct items from pool that are not already in exclude.
func pickDistinct(pool []string, k int, exclude []string, r *rand.Rand) []string {
	taken := map[string]bool{}
	for _, e := range exclude {
		taken[e] = true
	}
	avail := []string{}
	for _, p := range pool {
		if !taken[p] {
			avail = append(avail, p)
		}
	}
	r.Shuffle(len(avail), func(i, j int) { avail[i], avail[j] = avail[j], avail[i] })
	if k > len(avail) {
		k = len(avail)
	}
	if k < 0 {
		k = 0
	}
	return avail[:k]
}
