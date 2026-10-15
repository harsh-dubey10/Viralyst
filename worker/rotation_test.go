package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// fakeProviders points Groq and Gemini at local servers, each with one key, and records
// which model each provider was asked for. notFound makes that provider answer 404.
func fakeProviders(t *testing.T, notFound string) (calls map[string][]string, mu *sync.Mutex) {
	t.Helper()
	calls, mu = map[string][]string{}, &sync.Mutex{}
	serve := func(provider string) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req groqRequest
			json.NewDecoder(r.Body).Decode(&req)
			mu.Lock()
			calls[provider] = append(calls[provider], req.Model+"|"+req.ReasoningEffort+"|"+r.Header.Get("Authorization"))
			mu.Unlock()
			if provider == notFound {
				http.Error(w, `{"error":{"message":"model not found"}}`, http.StatusNotFound)
				return
			}
			w.Write([]byte(`{"choices":[{"message":{"content":"{\"ok\":true}"}}]}`))
		}))
	}
	groqSrv, geminiSrv := serve("groq"), serve("gemini")
	loadGroqAPIKeys()
	loadGeminiAPIKeys()
	oldGroq, oldGemini, oldGK, oldMK := groqBaseURL, geminiBaseURL, groqKeys, geminiKeys
	groqBaseURL, geminiBaseURL, groqKeys, geminiKeys = groqSrv.URL, geminiSrv.URL, []string{"groq-key"}, []string{"gemini-key"}
	t.Cleanup(func() {
		groqBaseURL, geminiBaseURL, groqKeys, geminiKeys = oldGroq, oldGemini, oldGK, oldMK
		disabledProviders.Delete("groq")
		disabledProviders.Delete("gemini")
		llmDisabled.Store(false)
		groqSrv.Close()
		geminiSrv.Close()
	})
	return calls, mu
}

func TestChatRotatesAcrossGroqAndGemini(t *testing.T) {
	calls, _ := fakeProviders(t, "")
	for i := 0; i < 6; i++ {
		if _, err := callGroq("hi"); err != nil {
			t.Fatal(err)
		}
	}
	if len(calls["groq"]) != 3 || len(calls["gemini"]) != 3 {
		t.Fatalf("calls not split evenly: groq=%d gemini=%d", len(calls["groq"]), len(calls["gemini"]))
	}
	if got := calls["groq"][0]; got != defaultGroqModel+"|low|Bearer groq-key" {
		t.Errorf("groq request = %q", got)
	}
	if got := calls["gemini"][0]; got != defaultGeminiModel+"|low|Bearer gemini-key" {
		t.Errorf("gemini request = %q (Gemini rejects reasoning_effort none)", got)
	}
}

func TestMissingModelDropsOnlyThatProvider(t *testing.T) {
	calls, _ := fakeProviders(t, "gemini")
	for i := 0; i < 4; i++ {
		if _, err := callGroq("hi"); err != nil {
			t.Fatalf("call %d failed although Groq works: %v", i, err)
		}
	}
	if len(calls["gemini"]) != 1 {
		t.Errorf("gemini asked %d times; after its 404 it should leave the rotation", len(calls["gemini"]))
	}
	if !hasLLMKeys() || llmDisabled.Load() {
		t.Error("Groq still works, so the LLM engine must stay on")
	}
}

func TestVideoCallsStayOnGroq(t *testing.T) {
	fakeProviders(t, "")
	used := map[string]int{}
	for i := 0; i < 4; i++ {
		withGroqKeys(func(apiKey string) (int, time.Duration, error) {
			used[apiKey]++
			return http.StatusOK, 0, nil
		})
	}
	if used["groq-key"] != 4 || len(used) != 1 {
		t.Errorf("video (Groq-only) calls used keys %v", used)
	}
}

// The layout in worker/.env: GROQ_API_KEY_1..4 and GEMINI_API_KEY_1..2 → six endpoints,
// alternating providers while both have keys left.
func TestAllNumberedKeysJoinTheRotation(t *testing.T) {
	for i, k := range []string{"g1", "g2", "g3", "g4"} {
		t.Setenv(fmt.Sprintf("GROQ_API_KEY_%d", i+1), k)
	}
	t.Setenv("GEMINI_API_KEY_1", "m1")
	t.Setenv("GEMINI_API_KEY_2", "m2")
	loadGroqAPIKeys()
	loadGeminiAPIKeys()
	oldGK, oldMK := groqKeys, geminiKeys
	groqKeys, geminiKeys = discoverKeys("GROQ"), discoverKeys("GEMINI")
	t.Cleanup(func() { groqKeys, geminiKeys = oldGK, oldMK })

	got := []string{}
	for _, ep := range chatEndpoints() {
		got = append(got, ep.key)
	}
	if want := []string{"g1", "m1", "g2", "m2", "g3", "g4"}; fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("rotation order = %v, want %v", got, want)
	}
}

func TestRateLimitedKeyIsSkippedWhileCoolingDown(t *testing.T) {
	var mu sync.Mutex
	hits := map[string]int{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("Authorization")
		mu.Lock()
		hits[key]++
		mu.Unlock()
		if key == "Bearer tired" {
			w.Header().Set("Retry-After", "60")
			http.Error(w, `{"error":{"message":"tokens per day"}}`, http.StatusTooManyRequests)
			return
		}
		w.Write([]byte(`{"choices":[{"message":{"content":"{}"}}]}`))
	}))
	defer srv.Close()
	t.Cleanup(func() { keyCooldowns.Delete("tired") })

	eps := []llmEndpoint{{provider: "groq", baseURL: srv.URL, key: "tired"}, {provider: "groq", baseURL: srv.URL, key: "fresh"}}
	var counter uint64
	for i := 0; i < 6; i++ {
		_, _, err := withEndpoints(eps, &counter, func(ep llmEndpoint) (int, time.Duration, error) {
			_, st, ra, err := callChatSingle(httpClient, ep.baseURL, groqRequest{}, ep.key)
			return st, ra, err
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if hits["Bearer tired"] != 1 || hits["Bearer fresh"] != 6 {
		t.Errorf("hits = %v: a 429'd key should be tried once, then skipped until its Retry-After", hits)
	}
}

// An org-level Anthropic key without ANTHROPIC_WORKSPACE_ID fails every call the same
// way; Claude must leave the rotation once, and the other providers carry on.
func TestUnusableAnthropicKeyLeavesRotation(t *testing.T) {
	calls, _ := fakeProviders(t, "")
	var claudeHits int
	claude := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claudeHits++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"type":"error","error":{"type":"invalid_request_error","message":"This API key is not scoped to a workspace, so this request must include the anthropic-workspace-id header"}}`))
	}))
	defer claude.Close()
	t.Setenv("ANTHROPIC_BASE_URL", claude.URL)
	loadAnthropicAPIKeys()
	oldKeys := anthropicKeys
	anthropicKeys = []string{"sk-ant-test-unscoped"}
	t.Cleanup(func() {
		anthropicKeys = oldKeys
		anthropicClients.Delete("sk-ant-test-unscoped")
		disabledProviders.Delete("anthropic")
	})

	for i := 0; i < 9; i++ {
		if _, err := callGroq("hi"); err != nil {
			t.Fatalf("call %d failed although Groq and Gemini work: %v", i, err)
		}
	}
	if claudeHits != 1 {
		t.Errorf("Claude asked %d times; after an unusable-key error it should leave the rotation", claudeHits)
	}
	if len(calls["groq"])+len(calls["gemini"]) != 9 {
		t.Errorf("groq+gemini served %d of 9 calls", len(calls["groq"])+len(calls["gemini"]))
	}
}
