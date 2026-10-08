// eval.go
// Checks the model against reality: simulate posts whose real performance is known
// and measure how well the virality score ranks them.
//
//	go run . -eval ../eval/posts.csv            # uses Groq if keys are set
//	go run . -eval ../eval/posts.csv -offline   # offline model only (free, fast)

package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
)

// evalPost is one post with a known real-world outcome.
type evalPost struct {
	ID          string
	Demographic string
	Niche       string
	Hook        string
	Outcome     float64 // any metric where higher = more viral, the same for every row
	Population  int
	Boost       int
}

// readEvalPosts parses the eval CSV. Required columns: id, demographic, hook, outcome.
// Optional: niche, population (default 100), boost (default 6). Lines starting with # are skipped.
func readEvalPosts(r io.Reader) ([]evalPost, error) {
	cr := csv.NewReader(r)
	cr.Comment = '#'
	cr.FieldsPerRecord = -1
	rows, err := cr.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, errors.New("empty file: expected a header row")
	}

	col := map[string]int{}
	for i, h := range rows[0] {
		col[strings.ToLower(strings.TrimSpace(h))] = i
	}
	for _, req := range []string{"id", "demographic", "hook", "outcome"} {
		if _, ok := col[req]; !ok {
			return nil, fmt.Errorf("missing required column %q (have %v)", req, rows[0])
		}
	}
	field := func(row []string, name string) string {
		if i, ok := col[name]; ok && i < len(row) {
			return strings.TrimSpace(row[i])
		}
		return ""
	}
	intField := func(row []string, name string, def, lo, hi, line int) (int, error) {
		s := field(row, name)
		if s == "" {
			return def, nil
		}
		n, err := strconv.Atoi(s)
		if err != nil || n < lo || n > hi {
			return 0, fmt.Errorf("line %d: %s must be a whole number in %d–%d, got %q", line, name, lo, hi, s)
		}
		return n, nil
	}

	posts := []evalPost{}
	for i, row := range rows[1:] {
		line := i + 2
		p := evalPost{ID: field(row, "id"), Demographic: field(row, "demographic"), Niche: field(row, "niche"), Hook: field(row, "hook")}
		if p.ID == "" || p.Demographic == "" || p.Hook == "" {
			return nil, fmt.Errorf("line %d: id, demographic and hook are required", line)
		}
		if p.Outcome, err = strconv.ParseFloat(field(row, "outcome"), 64); err != nil || math.IsNaN(p.Outcome) || math.IsInf(p.Outcome, 0) {
			return nil, fmt.Errorf("line %d: outcome must be a number, got %q", line, field(row, "outcome"))
		}
		// Same limits as the API accepts.
		if p.Population, err = intField(row, "population", 100, 50, 250, line); err != nil {
			return nil, err
		}
		if p.Boost, err = intField(row, "boost", 6, 1, 10, line); err != nil {
			return nil, err
		}
		posts = append(posts, p)
	}
	return posts, nil
}

// ranks returns 1-based ranks, giving tied values their average rank.
func ranks(xs []float64) []float64 {
	idx := make([]int, len(xs))
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(a, b int) bool { return xs[idx[a]] < xs[idx[b]] })
	out := make([]float64, len(xs))
	for i := 0; i < len(idx); {
		j := i
		for j+1 < len(idx) && xs[idx[j+1]] == xs[idx[i]] {
			j++
		}
		avg := float64(i+j)/2 + 1
		for k := i; k <= j; k++ {
			out[idx[k]] = avg
		}
		i = j + 1
	}
	return out
}

// spearman is the rank correlation of xs and ys (-1..1), or NaN if either is constant.
// It only compares orderings, so outcome can be on any scale (views, ratios, rates).
func spearman(xs, ys []float64) float64 {
	rx, ry := ranks(xs), ranks(ys)
	n := float64(len(rx))
	var mx, my float64
	for i := range rx {
		mx += rx[i] / n
		my += ry[i] / n
	}
	var cov, vx, vy float64
	for i := range rx {
		dx, dy := rx[i]-mx, ry[i]-my
		cov += dx * dy
		vx += dx * dx
		vy += dy * dy
	}
	if vx == 0 || vy == 0 {
		return math.NaN()
	}
	return cov / math.Sqrt(vx*vy)
}

// minEvalPosts is the fewest posts a correlation is worth reporting for.
const minEvalPosts = 5

// runEval simulates every post and prints how well virality ranks the real outcomes.
func runEval(path string, w *Worker, out io.Writer) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	posts, err := readEvalPosts(f)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	if len(posts) < minEvalPosts {
		return fmt.Errorf("%s has %d post(s); add at least %d (ideally 30+) to measure anything", path, len(posts), minEvalPosts)
	}

	engine := "offline model"
	if hasGroqAPIKeys() {
		engine = "Groq LLM personas"
	}
	fmt.Fprintf(out, "Simulating %d posts with the %s...\n\n", len(posts), engine)

	virality, outcomes := make([]float64, len(posts)), make([]float64, len(posts))
	tw := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tVIRALITY\tVERDICT\tOUTCOME")
	for i, p := range posts {
		res := w.ProcessSimulation(SimulationJob{
			JobID:              "eval-" + p.ID,
			Demographic:        p.Demographic,
			Niche:              p.Niche,
			ContentDescription: p.Hook,
			Population:         p.Population,
			Boost:              p.Boost,
		})
		virality[i], outcomes[i] = float64(res.Virality), p.Outcome
		fmt.Fprintf(tw, "%s\t%d\t%s\t%g\n", p.ID, res.Virality, res.Verdict.Head, p.Outcome)
	}
	tw.Flush()

	rho := spearman(virality, outcomes)
	fmt.Fprintln(out)
	if math.IsNaN(rho) {
		fmt.Fprintln(out, "Spearman ρ: undefined — every post got the same virality score or the same outcome.")
		return nil
	}
	// Two-sided 5% threshold for ρ under no correlation (large-sample approximation).
	threshold := 1.96 / math.Sqrt(float64(len(posts)-1))
	fmt.Fprintf(out, "Spearman ρ = %.2f over %d posts (1 = perfect ranking, 0 = no better than chance).\n", rho, len(posts))
	if math.Abs(rho) < threshold {
		fmt.Fprintf(out, "Not distinguishable from chance at this sample size (|ρ| needs to exceed ~%.2f).\n", threshold)
	} else if rho > 0 {
		fmt.Fprintf(out, "Better than chance (|ρ| > ~%.2f): higher virality scores went with better real outcomes.\n", threshold)
	} else {
		fmt.Fprintf(out, "Worse than chance (ρ < -%.2f): the score ranks these posts backwards.\n", threshold)
	}
	return nil
}
