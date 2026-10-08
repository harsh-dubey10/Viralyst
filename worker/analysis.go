// analysis.go
// Turns the free-text inputs (target demographic + content hook) into structured
// signals that drive the simulation: audience topics/age/regions, content topics,
// and hook strength. Everything downstream (persona generation, decisions, spread,
// verdict) is computed from these, so the prediction depends on the context given.

package main

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// topic is one interest category and the words/phrases that signal it in text.
type topic struct {
	Name     string
	Keywords []string
}

// topicLexicon is ordered (slice, not map) so generation stays deterministic.
var topicLexicon = []topic{
	{"AI tools", []string{"ai", "artificial intelligence", "llm", "llms", "gpt", "chatgpt", "machine learning", "chatbot", "ai agent", "ai agents", "automation", "automate", "prompt", "prompts"}},
	{"B2B SaaS", []string{"saas", "b2b", "mrr", "arr", "churn", "subscription", "subscriptions", "enterprise software"}},
	{"startup ops", []string{"startup", "startups", "founder", "founders", "bootstrapped", "indie hacker", "indie hackers", "build in public", "building in public", "0 to 1", "entrepreneur", "entrepreneurs", "entrepreneurship"}},
	{"growth hacking", []string{"growth", "viral", "virality", "distribution", "scale", "scaling", "user acquisition", "funnel", "playbook", "paid ads", "go to market"}},
	{"dev tools", []string{"developer", "developers", "devops", "debugging", "debug", "coding", "programmer", "programmers", "software engineer", "software engineers", "engineers", "microservices", "api", "apis", "github", "full stack", "backend", "frontend"}},
	{"product design", []string{"product design", "ux", "ui", "designer", "designers", "product manager", "product managers", "figma", "user research"}},
	{"content marketing", []string{"content marketing", "marketing", "marketers", "seo", "copywriting", "newsletter", "branding", "agency", "agencies"}},
	{"creator economy", []string{"creator", "creators", "content creator", "content creators", "youtube", "youtuber", "tiktok", "reels", "shorts", "instagram", "influencer", "influencers", "video editing", "editors"}},
	{"personal finance", []string{"personal finance", "budget", "budgeting", "savings", "debt", "salary", "frugal", "money"}},
	{"investing", []string{"investing", "investor", "investors", "stocks", "stock market", "index funds", "dividends", "venture capital"}},
	{"cryptocurrency", []string{"crypto", "bitcoin", "ethereum", "web3", "blockchain", "nft", "nfts", "defi"}},
	{"real estate", []string{"real estate", "property", "mortgage", "landlord", "housing"}},
	{"fitness", []string{"fitness", "gym", "workout", "workouts", "exercise", "exercises", "muscle", "muscles", "strength training", "weight loss", "lifting", "bodybuilding", "squat", "squats", "reps", "abs", "cardio"}},
	{"running", []string{"running", "runner", "runners", "marathon", "5k", "10k"}},
	{"yoga", []string{"yoga", "pilates", "stretching", "mobility"}},
	{"mental health", []string{"mental health", "anxiety", "therapy", "burnout", "stress", "mindfulness", "meditation"}},
	{"nutrition", []string{"nutrition", "diet", "healthy eating", "calories", "protein", "vegan", "plant based", "meal prep", "supplement", "supplements"}},
	{"cooking", []string{"cooking", "cook", "recipe", "recipes", "chef", "kitchen", "baking", "bake", "ingredients", "ingredient", "dinner", "breakfast", "lunch", "meal", "meals", "pasta", "dish", "dishes", "oven", "air fryer", "homemade"}},
	{"coffee", []string{"coffee", "espresso", "latte", "barista", "crema", "coffee pods"}},
	{"food", []string{"food", "foodie", "foodies", "restaurant", "restaurants", "street food", "snacks"}},
	{"sustainability", []string{"sustainable", "sustainability", "eco", "eco conscious", "plastic free", "zero waste", "climate", "planet", "recycling", "carbon"}},
	{"fashion", []string{"fashion", "outfit", "outfits", "streetwear", "clothing", "sneakers"}},
	{"beauty", []string{"beauty", "skincare", "makeup", "cosmetics", "haircare"}},
	{"travel", []string{"travel", "traveling", "travelling", "traveler", "travelers", "vacation", "backpacking", "backpacker", "backpackers", "flights", "hotel", "hotels", "hostel", "hostels", "destinations", "trip", "trips", "itinerary", "road trip", "abroad", "passport", "vietnam", "thailand", "bali"}},
	{"gaming", []string{"gaming", "gamer", "gamers", "video games", "esports", "playstation", "xbox", "nintendo", "twitch"}},
	{"music", []string{"music", "musician", "musicians", "song", "songs", "beats", "album", "concert"}},
	{"movies", []string{"movies", "movie", "film", "films", "cinema", "netflix", "tv shows"}},
	{"books", []string{"books", "book", "reading", "novel", "novels", "authors"}},
	{"sports", []string{"sports", "football", "soccer", "cricket", "basketball", "nba", "nfl", "tennis"}},
	{"parenting", []string{"parenting", "parents", "moms", "mom", "dads", "dad", "kids", "toddler", "toddlers", "baby", "babies"}},
	{"pets", []string{"pets", "pet", "dog", "dogs", "cat", "cats", "puppy"}},
	{"cars", []string{"cars", "car", "electric vehicles", "tesla", "automotive"}},
	{"photography", []string{"photography", "photographer", "photographers", "camera", "lens"}},
	{"art", []string{"art", "artist", "artists", "drawing", "painting", "illustration"}},
	{"DIY", []string{"diy", "crafts", "woodworking", "handmade"}},
	{"home decor", []string{"home decor", "interior design", "decor", "furniture"}},
	{"gardening", []string{"gardening", "garden", "gardens", "plants", "houseplants"}},
	{"productivity", []string{"productivity", "productive", "habits", "routine", "time management", "notion"}},
	{"leadership", []string{"leadership", "leaders", "management", "executives", "ceo", "ceos"}},
	{"remote work", []string{"remote work", "work from home", "wfh", "digital nomad", "digital nomads"}},
	{"education", []string{"education", "students", "student", "course", "courses", "teachers", "exam", "exams"}},
	{"science", []string{"science", "physics", "biology", "astronomy", "scientists"}},
	{"comedy", []string{"comedy", "funny", "memes", "meme", "jokes", "humor"}},
	{"fintech", []string{"fintech", "payments", "banking", "neobank"}},
	{"self-improvement", []string{"self improvement", "personal growth", "personal development", "discipline", "mindset", "motivation", "motivational", "stoic", "stoicism", "masculinity", "confidence", "self help"}},
	{"relationships", []string{"relationships", "relationship", "dating", "marriage", "couples", "breakup", "love life"}},
	{"true crime", []string{"true crime", "crime", "murder", "serial killer", "cold case", "detective"}},
	{"history", []string{"history", "historical", "ancient", "ww2", "world war", "empire", "empires"}},
	{"politics", []string{"politics", "political", "election", "elections", "government", "geopolitics", "current events", "breaking news"}},
	{"spirituality", []string{"spirituality", "spiritual", "religion", "faith", "astrology", "manifestation", "manifesting"}},
	{"careers", []string{"career", "careers", "job hunting", "job search", "interview", "interviews", "resume", "hiring", "internship", "internships", "corporate"}},
	{"anime", []string{"anime", "manga", "otaku"}},
	{"dance", []string{"dance", "dancing", "dancer", "dancers", "choreography"}},
	{"gadgets", []string{"gadgets", "gadget", "smartphone", "smartphones", "iphone", "android", "laptop", "laptops", "tech review", "unboxing"}},
}

var allTopicNames = func() []string {
	names := make([]string, len(topicLexicon))
	for i, t := range topicLexicon {
		names[i] = t.Name
	}
	return names
}()

var nonAlnum = regexp.MustCompile(`[^a-z0-9%]+`)

// normalize lowercases text and pads words with spaces so " kw " matches whole words/phrases.
func normalize(text string) string {
	return " " + strings.TrimSpace(nonAlnum.ReplaceAllString(strings.ToLower(text), " ")) + " "
}

func containsPhrase(norm, phrase string) bool {
	return strings.Contains(norm, normalize(phrase))
}

// detectTopics returns topic names mentioned in text, in lexicon order, followed by any
// extra (custom niche) topics it mentions.
func detectTopics(text string, extra ...topic) []string {
	norm := normalize(text)
	found := []string{}
	for _, t := range append(topicLexicon[:len(topicLexicon):len(topicLexicon)], extra...) {
		for _, kw := range t.Keywords {
			if containsPhrase(norm, kw) {
				found = append(found, t.Name)
				break
			}
		}
	}
	return found
}

// ──────────────────────────────────────────────
// Niche (the creator's content domain)
// ──────────────────────────────────────────────

const maxNicheLen = 40

// Words too generic to identify a niche on their own ("cooking videos for busy people").
var nicheStopwords = map[string]bool{
	"and": true, "for": true, "with": true, "the": true, "into": true, "about": true, "from": true,
	"that": true, "this": true, "your": true, "their": true, "people": true, "lovers": true, "fans": true,
	"enthusiasts": true, "enthusiast": true, "content": true, "videos": true, "video": true, "tips": true,
	"ideas": true, "niche": true, "creators": true, "creator": true, "community": true, "channel": true,
	"stuff": true, "things": true, "daily": true, "life": true, "lifestyle": true,
}

// resolveNiche maps the creator's niche onto lexicon topics. A niche the lexicon doesn't
// cover becomes a custom topic of its own (returned as custom), so any domain — chess,
// knitting, aquariums — gets an audience that actually cares about it.
func resolveNiche(niche string) (topics []string, custom *topic) {
	niche = strings.Join(strings.Fields(niche), " ")
	if niche == "" {
		return nil, nil
	}
	for _, t := range topicLexicon {
		if strings.EqualFold(t.Name, niche) {
			return []string{t.Name}, nil
		}
	}
	if found := detectTopics(niche); len(found) > 0 {
		if len(found) > maxReadTopics {
			found = found[:maxReadTopics]
		}
		return found, nil
	}
	if len([]rune(niche)) > maxNicheLen {
		niche = strings.TrimSpace(string([]rune(niche)[:maxNicheLen]))
	}
	return []string{niche}, &topic{Name: niche, Keywords: nicheKeywords(niche)}
}

// nicheKeywords is the niche phrase plus its distinctive words (and their singular or
// plural form), so a hook that mentions "knit" or "knitting" matches a "Knitting" niche.
func nicheKeywords(niche string) []string {
	kws := []string{strings.ToLower(niche)}
	for _, w := range strings.Fields(strings.TrimSpace(nonAlnum.ReplaceAllString(strings.ToLower(niche), " "))) {
		if len(w) < 4 || nicheStopwords[w] {
			continue
		}
		kws = append(kws, w)
		if strings.HasSuffix(w, "s") {
			kws = append(kws, strings.TrimSuffix(w, "s"))
		} else {
			kws = append(kws, w+"s")
		}
		if stem := strings.TrimSuffix(w, "ing"); stem != w && len(stem) >= 3 {
			// knitting → knit, running → run, baking → bake
			if n := len(stem); stem[n-1] == stem[n-2] {
				stem = stem[:n-1]
			}
			kws = append(kws, stem, stem+"e")
		}
	}
	return kws
}

// mergeTopics returns a followed by the items of b not already in a, capped at maxReadTopics.
func mergeTopics(a, b []string) []string {
	out, seen := []string{}, map[string]bool{}
	for _, t := range append(append([]string{}, a...), b...) {
		if !seen[t] && len(out) < maxReadTopics {
			seen[t] = true
			out = append(out, t)
		}
	}
	return out
}

// ──────────────────────────────────────────────
// Audience (target demographic)
// ──────────────────────────────────────────────

const (
	minPersonaAge = 18
	maxPersonaAge = 65
)

type AudienceProfile struct {
	Topics  []string
	Core    []string // the niche's topics, if a niche was given: fit is measured against these
	MinAge  int
	MaxAge  int
	AgeNote string   // e.g. "Gen Z (18–27)"; empty if not specified
	Regions []string // location suffix keys, e.g. "India", "US"
}

var ageCohorts = []struct {
	phrases  []string
	min, max int
	label    string
}{
	{[]string{"gen z", "genz", "zoomer", "zoomers"}, 18, 27, "Gen Z"},
	{[]string{"millennial", "millennials"}, 28, 43, "Millennials"},
	{[]string{"gen x"}, 44, 59, "Gen X"},
	{[]string{"boomer", "boomers", "baby boomers"}, 58, 65, "Boomers"},
	{[]string{"retirees", "retired", "seniors"}, 60, 65, "Retirees"},
	{[]string{"teen", "teens", "teenagers"}, 18, 19, "Teens (18+)"},
	{[]string{"college", "students", "university", "undergrads"}, 18, 24, "Students"},
	{[]string{"young professionals"}, 23, 32, "Young professionals"},
}

var ageRangeRe = regexp.MustCompile(`\b(\d{2})\s*(?:-|–|to)\s*(\d{2})\b`)
var agePlusRe = regexp.MustCompile(`\b(\d{2})\s*\+`)
var usRe = regexp.MustCompile(`\b(US|USA|U\.S\.)\b`)

var regionKeywords = []struct {
	phrases []string
	regions []string
}{
	{[]string{"india", "indian", "bangalore", "bengaluru", "mumbai", "delhi"}, []string{"India"}},
	{[]string{"usa", "united states", "american", "americans", "america", "new york", "silicon valley"}, []string{"US"}},
	{[]string{"north america", "north american"}, []string{"US", "CA"}},
	{[]string{"uk", "united kingdom", "british", "britain", "london"}, []string{"UK"}},
	{[]string{"europe", "european", "eu"}, []string{"UK", "DE", "FR"}},
	{[]string{"germany", "german", "berlin"}, []string{"DE"}},
	{[]string{"france", "french", "paris"}, []string{"FR"}},
	{[]string{"brazil", "brazilian"}, []string{"BR"}},
	{[]string{"latam", "latin america", "mexico", "mexican"}, []string{"BR", "MX"}},
	{[]string{"canada", "canadian"}, []string{"CA"}},
	{[]string{"japan", "japanese", "tokyo"}, []string{"JP"}},
	{[]string{"korea", "korean", "seoul"}, []string{"KR"}},
	{[]string{"china", "chinese"}, []string{"CN"}},
	{[]string{"singapore"}, []string{"SG"}},
	{[]string{"asia", "asian", "apac"}, []string{"India", "SG", "KR", "JP", "CN"}},
	{[]string{"australia", "australian"}, []string{"AU"}},
	{[]string{"uae", "dubai", "middle east"}, []string{"UAE", "IL"}},
	{[]string{"africa", "african", "south africa"}, []string{"ZA"}},
}

func ParseAudience(demographic string) AudienceProfile {
	a := AudienceProfile{Topics: detectTopics(demographic), MinAge: minPersonaAge, MaxAge: maxPersonaAge}
	norm := normalize(demographic)

	if m := ageRangeRe.FindStringSubmatch(demographic); m != nil {
		lo, _ := strconv.Atoi(m[1])
		hi, _ := strconv.Atoi(m[2])
		a.MinAge, a.MaxAge = clampInt(lo, minPersonaAge, maxPersonaAge), clampInt(hi, minPersonaAge, maxPersonaAge)
		a.AgeNote = fmt.Sprintf("%d–%d", a.MinAge, a.MaxAge)
	} else if m := agePlusRe.FindStringSubmatch(demographic); m != nil {
		lo, _ := strconv.Atoi(m[1])
		a.MinAge = clampInt(lo, minPersonaAge, maxPersonaAge)
		a.AgeNote = fmt.Sprintf("%d+", a.MinAge)
	} else {
		for _, c := range ageCohorts {
			if anyPhrase(norm, c.phrases) {
				a.MinAge, a.MaxAge = c.min, c.max
				a.AgeNote = fmt.Sprintf("%s (%d–%d)", c.label, c.min, c.max)
				break
			}
		}
	}
	if a.MinAge > a.MaxAge {
		a.MinAge, a.MaxAge = a.MaxAge, a.MinAge
	}

	seen := map[string]bool{}
	addRegions := func(rs []string) {
		for _, r := range rs {
			if !seen[r] {
				seen[r] = true
				a.Regions = append(a.Regions, r)
			}
		}
	}
	if usRe.MatchString(demographic) {
		addRegions([]string{"US"})
	}
	for _, rk := range regionKeywords {
		if anyPhrase(norm, rk.phrases) {
			addRegions(rk.regions)
		}
	}
	return a
}

func anyPhrase(norm string, phrases []string) bool {
	for _, p := range phrases {
		if containsPhrase(norm, p) {
			return true
		}
	}
	return false
}

// ──────────────────────────────────────────────
// Content (hook)
// ──────────────────────────────────────────────

type ContentProfile struct {
	Topics    []string
	HookScore float64 // 0..1
	Debate    bool    // poses a question or a contrarian claim → invites comments
	Signals   []string
	Issues    []string
	HasText   bool
}

var (
	digitRe      = regexp.MustCompile(`\d`)
	sentenceEnd  = regexp.MustCompile(`[.!?\n]`)
	allCapsWord  = regexp.MustCompile(`\b[A-Z]{4,}\b`)
	curiosityKWs = []string{"why", "secret", "secrets", "nobody", "no one", "mistake", "mistakes", "stop", "truth", "exact", "revealed", "revealing", "so you don t have to", "the only", "never", "what happens", "surprising", "myth", "myths", "wrong", "what if", "guess", "which one", "can you"}
	payoffKWs    = []string{"how to", "step", "steps", "playbook", "framework", "guide", "template", "checklist", "tested", "lessons", "breakdown", "case study", "formula"}
	directKWs    = []string{"you", "your", "you re"}
	contrarian   = []string{"stop", "wrong", "myth", "myths", "overrated", "nobody", "never", "unpopular opinion"}
	hypeKWs      = []string{"buy now", "link in bio", "limited offer", "click", "subscribe", "dm me", "discount", "promo code", "act now", "giveaway"}
)

const noTopicIssue = "No clear topic detected — name the subject so the right people stop scrolling."

// AnalyzeContent scores the hook text. topics, if non-nil, replaces keyword detection
// (e.g. topics read by the LLM); extra are custom niche topics to detect alongside the lexicon.
func AnalyzeContent(job SimulationJob, topics []string, extra ...topic) ContentProfile {
	text := strings.TrimSpace(job.ContentDescription)
	if topics == nil {
		topics = detectTopics(text, extra...)
	}
	c := ContentProfile{Topics: topics, HasText: text != ""}

	if !c.HasText {
		c.HookScore = 0.30
		// A video that couldn't be analysed is reported by newSimContext.
		if job.VideoFilename == "" {
			c.Issues = append(c.Issues, "No hook text provided — personas have nothing concrete to react to.")
		}
		return c
	}

	norm := normalize(text)
	words := strings.Fields(strings.TrimSpace(nonAlnum.ReplaceAllString(strings.ToLower(text), " ")))
	score := 0.30

	if digitRe.MatchString(text) {
		score += 0.12
		c.Signals = append(c.Signals, "Concrete numbers / results")
	} else {
		c.Issues = append(c.Issues, "No concrete number or result — add one (e.g. \"3 steps\", \"50k MRR\", \"in 7 days\").")
	}

	curiosity := 0
	for _, kw := range curiosityKWs {
		if containsPhrase(norm, kw) {
			curiosity++
		}
	}
	if strings.Contains(text, "?") {
		curiosity++ // a question is the classic way to open a loop
	}
	if curiosity > 0 {
		score += 0.08 * float64(minInt(curiosity, 3))
		c.Signals = append(c.Signals, "Curiosity / pattern-interrupt language")
	} else {
		c.Issues = append(c.Issues, "No tension in the opening — try a question, a \"stop doing X\", or a surprising claim.")
	}

	if anyPhrase(norm, payoffKWs) {
		score += 0.08
		c.Signals = append(c.Signals, "Clear payoff (steps / framework / tested result)")
	}
	if anyPhrase(norm, directKWs) {
		score += 0.06
		c.Signals = append(c.Signals, "Speaks directly to the viewer")
	}
	if strings.Contains(text, "?") || anyPhrase(norm, contrarian) {
		c.Debate = true
	}

	firstSentence := strings.Fields(sentenceEnd.Split(text, 2)[0])
	switch {
	case len(words) < 6:
		score -= 0.15
		c.Issues = append(c.Issues, "Hook is too thin to judge — describe what the viewer sees and hears.")
	case len(firstSentence) <= 16:
		score += 0.08
		c.Signals = append(c.Signals, fmt.Sprintf("Punchy opening line (%d words)", len(firstSentence)))
	case len(firstSentence) > 28:
		score -= 0.10
		c.Issues = append(c.Issues, fmt.Sprintf("Opening sentence is %d words — cut it under 15 so it lands in 3 seconds.", len(firstSentence)))
	}

	hype := 0
	for _, kw := range hypeKWs {
		if containsPhrase(norm, kw) {
			hype++
		}
	}
	if hype > 0 || strings.Count(text, "!") >= 3 || len(allCapsWord.FindAllString(text, -1)) >= 2 {
		score -= 0.12 + 0.06*float64(minInt(hype, 3))
		c.Issues = append(c.Issues, "Reads as an ad (salesy CTA / shouting) — audiences swipe past these fastest.")
	}

	if len(c.Topics) == 0 {
		c.Issues = append(c.Issues, noTopicIssue)
	}

	c.HookScore = clampF(score, 0.05, 0.95)
	return c
}

// relevance scores how much a persona cares about the content (0..1).
func relevance(p PersonaProfile, c ContentProfile) float64 {
	return interestRelevance(p.Interests, c)
}

func interestRelevance(interests []string, c ContentProfile) float64 {
	if len(c.Topics) == 0 {
		return 0.20 // generic content: mild, uniform appeal
	}
	switch matched := len(matchedTopics(interests, c.Topics)); {
	case matched >= 2:
		return 1.0
	case matched == 1:
		return 0.65
	default:
		return 0.0
	}
}

func matchedTopics(interests, topics []string) []string {
	out := []string{}
	for _, i := range interests {
		for _, t := range topics {
			if i == t {
				out = append(out, i)
			}
		}
	}
	return out
}

// fitTopics are the audience interests a post is expected to speak to: the niche when
// one was given (a cooking post for "busy parents" needn't be about parenting),
// otherwise everything read from the audience description.
func fitTopics(a AudienceProfile) []string {
	if len(a.Core) > 0 {
		return a.Core
	}
	return a.Topics
}

// AudienceFit is the share of the audience's interests the content speaks to (0..100), or -1 if unknown.
func AudienceFit(a AudienceProfile, c ContentProfile) int {
	topics := fitTopics(a)
	if len(topics) == 0 {
		return -1
	}
	matched := len(matchedTopics(topics, c.Topics))
	denom := minInt(len(topics), 2)
	return minInt(100, matched*100/denom)
}

func buildRecommendations(a AudienceProfile, c ContentProfile, fit int) []string {
	recs := []string{}
	if len(a.Topics) == 0 {
		recs = append(recs, "Your audience has no clear interests — pick a niche (e.g. cooking, travel, chess) so the right people are simulated.")
	} else if fit == 0 && len(c.Topics) > 0 {
		recs = append(recs, fmt.Sprintf("The hook is about %s, but your audience cares about %s — lead with one of their interests or retarget.",
			strings.Join(c.Topics, ", "), strings.Join(a.Topics, ", ")))
	} else if fit > 0 && fit < 100 {
		missing := []string{}
		for _, t := range fitTopics(a) {
			if len(matchedTopics([]string{t}, c.Topics)) == 0 {
				missing = append(missing, t)
			}
		}
		sort.Strings(missing)
		if len(missing) > 0 {
			recs = append(recs, fmt.Sprintf("Partial audience fit — the hook doesn't touch %s.", strings.Join(missing, ", ")))
		}
	}
	return append(recs, c.Issues...)
}

func clampF(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
