package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// withFakeGroq points the Groq client at a local server and configures one key.
func withFakeGroq(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	srv := httptest.NewServer(handler)
	loadGroqAPIKeys() // run the once-only env discovery before overriding
	oldURL, oldKeys := groqBaseURL, groqKeys
	groqBaseURL, groqKeys = srv.URL, []string{"test-key"}
	t.Cleanup(func() {
		groqBaseURL, groqKeys = oldURL, oldKeys
		srv.Close()
	})
}

func TestTranscribeSendsAudioAndModel(t *testing.T) {
	withFakeGroq(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/audio/transcriptions" || r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("unexpected request %s auth=%q", r.URL.Path, r.Header.Get("Authorization"))
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatal(err)
		}
		if got := r.FormValue("model"); got != defaultTranscribeModel {
			t.Errorf("model = %q", got)
		}
		f, _, err := r.FormFile("file")
		if err != nil {
			t.Fatal(err)
		}
		if b, _ := io.ReadAll(f); string(b) != "RIFFfake" {
			t.Errorf("audio body = %q", b)
		}
		w.Write([]byte(`{"text":"  Stop wasting money on coffee pods. "}`))
	})

	audio := filepath.Join(t.TempDir(), "hook.wav")
	os.WriteFile(audio, []byte("RIFFfake"), 0o600)
	got, err := transcribe(audio)
	if err != nil || got != "Stop wasting money on coffee pods." {
		t.Fatalf("transcribe = %q, %v", got, err)
	}
}

func TestDescribeFramesSendsImagesInOrder(t *testing.T) {
	withFakeGroq(t, func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Model    string `json:"model"`
			Messages []struct {
				Content []map[string]any `json:"content"`
			} `json:"messages"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		parts := req.Messages[0].Content
		if req.Model != defaultVisionModel || len(parts) != 3 || parts[0]["type"] != "text" {
			t.Fatalf("unexpected payload: model=%q parts=%d", req.Model, len(parts))
		}
		for _, p := range parts[1:] {
			url := p["image_url"].(map[string]any)["url"].(string)
			if !strings.HasPrefix(url, "data:image/jpeg;base64,") {
				t.Errorf("image url = %.40q", url)
			}
		}
		w.Write([]byte(`{"choices":[{"message":{"content":"A barista slams a coffee pod into a bin. Text: \"STOP\""}}]}`))
	})

	dir := t.TempDir()
	frames := []string{filepath.Join(dir, "frame1.jpg"), filepath.Join(dir, "frame2.jpg")}
	for _, f := range frames {
		os.WriteFile(f, []byte{0xff, 0xd8}, 0o600)
	}
	got, err := describeFrames(frames)
	if err != nil || !strings.Contains(got, "coffee pod") {
		t.Fatalf("describeFrames = %q, %v", got, err)
	}
}

func TestVisionModelNotFoundDoesNotDisablePersonaLLM(t *testing.T) {
	withFakeGroq(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":{"message":"model not found"}}`, http.StatusNotFound)
	})
	f := filepath.Join(t.TempDir(), "frame1.jpg")
	os.WriteFile(f, []byte{0xff}, 0o600)
	if _, err := describeFrames([]string{f}); err == nil {
		t.Fatal("expected an error for a missing vision model")
	}
	if llmDisabled.Load() {
		t.Fatal("a missing vision model must not switch persona decisions offline")
	}
}

func TestAnalyzeVideoWithoutKeysExplainsWhy(t *testing.T) {
	v := AnalyzeVideo("job_clip.mp4")
	if v.Hook() != "" || !strings.Contains(v.Note, "Groq API key") {
		t.Fatalf("got hook=%q note=%q", v.Hook(), v.Note)
	}
	if (AnalyzeVideo("") != VideoInsight{}) {
		t.Fatal("no video should produce an empty insight")
	}
}

func TestVideoHookFeedsTheSimulation(t *testing.T) {
	job := SimulationJob{JobID: "v", Demographic: "eco-conscious coffee lovers", VideoFilename: "v_clip.mp4"}
	video := VideoInsight{Transcript: "Your coffee pods are filling landfills", Visual: "A pile of plastic coffee pods."}

	sc := newSimContext(job, video)
	if !strings.Contains(sc.job.ContentDescription, "coffee pods are filling landfills") {
		t.Fatalf("hook not merged into content: %q", sc.job.ContentDescription)
	}
	if len(sc.content.Topics) == 0 || sc.content.Topics[0] != "coffee" {
		t.Errorf("topics from video = %v, want coffee first", sc.content.Topics)
	}

	failed := newSimContext(job, VideoInsight{Note: "ffmpeg is not installed on the worker"})
	issues := strings.Join(failed.content.Issues, " ")
	if !strings.Contains(issues, "ffmpeg is not installed") || !strings.Contains(issues, "add a text description") {
		t.Errorf("missing-video issue not reported: %v", failed.content.Issues)
	}

	job.ContentDescription = "Your coffee habit is wrecking the planet"
	partial := newSimContext(job, VideoInsight{Note: "audio transcription failed"})
	if !strings.Contains(strings.Join(partial.content.Issues, " "), "text description only") {
		t.Errorf("text fallback not reported: %v", partial.content.Issues)
	}
}
