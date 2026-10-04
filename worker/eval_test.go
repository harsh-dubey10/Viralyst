package main

import (
	"bytes"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSpearman(t *testing.T) {
	cases := []struct {
		name   string
		xs, ys []float64
		want   float64
	}{
		{"same order, different scales", []float64{10, 20, 30, 40}, []float64{1, 100, 1000, 5000}, 1},
		{"reversed", []float64{1, 2, 3, 4}, []float64{9, 7, 5, 1}, -1},
		{"ties get average ranks", []float64{1, 2, 2, 3}, []float64{1, 2, 3, 4}, 0.9486832980505138},
	}
	for _, c := range cases {
		if got := spearman(c.xs, c.ys); math.Abs(got-c.want) > 1e-9 {
			t.Errorf("%s: spearman = %v, want %v", c.name, got, c.want)
		}
	}
	if !math.IsNaN(spearman([]float64{5, 5, 5}, []float64{1, 2, 3})) {
		t.Error("constant scores should give NaN")
	}
}

func TestReadEvalPosts(t *testing.T) {
	csv := "# comment line\nid,demographic,hook,outcome,boost\n" +
		"a,Gen Z gamers,\"Stop, this 1 trick wins every match\",3.5,8\n" +
		"b,coffee lovers,Your pods are trash,0.2,\n"
	posts, err := readEvalPosts(strings.NewReader(csv))
	if err != nil {
		t.Fatal(err)
	}
	if len(posts) != 2 || posts[0].Hook != "Stop, this 1 trick wins every match" || posts[0].Boost != 8 || posts[1].Population != 100 || posts[1].Boost != 6 {
		t.Fatalf("parsed %+v", posts)
	}

	for name, bad := range map[string]string{
		"missing column":   "id,demographic,hook\na,b,c\n",
		"non-numeric":      "id,demographic,hook,outcome\na,b,c,lots\n",
		"population range": "id,demographic,hook,outcome,population\na,b,c,1,5000\n",
		"empty hook":       "id,demographic,hook,outcome\na,b,,1\n",
	} {
		if _, err := readEvalPosts(strings.NewReader(bad)); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

func TestRunEvalReportsCorrelation(t *testing.T) {
	rows := []string{"id,demographic,hook,outcome"}
	for _, s := range scenarios[:minEvalPosts] {
		rows = append(rows, strings.Join([]string{
			strings.ReplaceAll(s.name, ",", ""), `"` + s.demographic + `"`, `"` + strings.ReplaceAll(s.content, `"`, `""`) + `"`, "1",
		}, ","))
	}
	// Distinct outcomes so the correlation is defined.
	for i := range rows[1:] {
		rows[i+1] = strings.TrimSuffix(rows[i+1], "1") + string(rune('1'+i))
	}
	path := filepath.Join(t.TempDir(), "posts.csv")
	os.WriteFile(path, []byte(strings.Join(rows, "\n")+"\n"), 0o600)

	var out bytes.Buffer
	if err := runEval(path, &Worker{concurrency: 4}, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "offline model") || !strings.Contains(out.String(), "Spearman ρ") {
		t.Fatalf("unexpected report:\n%s", out.String())
	}

	os.WriteFile(path, []byte("id,demographic,hook,outcome\na,b,c,1\n"), 0o600)
	if err := runEval(path, &Worker{concurrency: 4}, &out); err == nil || !strings.Contains(err.Error(), "at least") {
		t.Fatalf("too few posts should be rejected, got %v", err)
	}
}
