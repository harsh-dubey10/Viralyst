// video.go
// Turns an uploaded video into hook text the simulation can react to: ffmpeg pulls
// frames and audio from the opening seconds, Groq Whisper transcribes the audio, and
// a Groq vision model describes the frames (including any on-screen text).

package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// hookSeconds is how much of the video counts as the hook.
const hookSeconds = 3

// The rest of the video: whether people like, comment on or share a post depends on its
// payoff, not just its opening, so the whole video is summarised too — from the full
// transcript (up to storyMaxSeconds) and storyFrames frames spread across it.
const (
	storyMaxSeconds = 180
	storyFrames     = 6
	maxRestWords    = 80
)

const (
	defaultVisionModel     = "qwen/qwen3.8-27b" // the only image-capable model Groq serves as of Oct 2026
	defaultTranscribeModel = "whisper-large-v3-turbo"
)

// Vision and transcription requests carry media, so they get a longer timeout.
var mediaHTTPClient = &http.Client{Timeout: 60 * time.Second}

const visionPrompt = "This image shows 3 frames side by side (left to right) from the first 3 seconds of a short social media video. " +
	`Reply with JSON: {"on_screen_text": "any text overlaid on the video, quoted exactly, or empty", ` +
	`"scene": "one sentence: the subject, action and setting a viewer sees", ` +
	`"about": "what the video is about, in under 8 words"}. Describe only what is visible; do not judge it.`

const restPrompt = "This image shows %d frames (left to right, then top to bottom) spread evenly across a short social media video. " +
	"Full transcript of what is said: %s\n" +
	`Reply with JSON: {"rest": "2-3 sentences on what happens after the first few seconds, as a viewer experiences it: the story, payoff or reveal, and the tone or energy"}. Describe what happens; do not judge whether it is good.`

// VideoInsight is what was extracted from a video: its opening seconds in detail, and a
// summary of the rest.
type VideoInsight struct {
	Transcript   string // spoken words in the opening, if any
	OnScreenText string // text overlaid on the opening
	Scene        string // what a viewer sees in the opening
	About        string // the video's subject, in a few words
	Rest         string // what happens after the opening ("" if short or not analysed)
	Note         string // why the video couldn't be (fully) analysed; empty on success
}

func labelled(pairs ...string) string {
	parts := []string{}
	for i := 0; i < len(pairs); i += 2 {
		if pairs[i+1] != "" {
			parts = append(parts, pairs[i]+": "+pairs[i+1])
		}
	}
	return strings.Join(parts, "\n")
}

func quoted(s string) string {
	if s == "" {
		return ""
	}
	return fmt.Sprintf("%q", s)
}

// Hook is everything extracted from the opening, for the LLM personas and the UI.
func (v VideoInsight) Hook() string {
	return labelled("Spoken", quoted(v.Transcript), "On-screen text", quoted(v.OnScreenText), "Scene", v.Scene)
}

// HookText is what the viewer is told, plus the subject, for topic detection and hook
// scoring. The scene is left out: props and settings ("in a car") aren't the topic.
func (v VideoInsight) HookText() string {
	return labelled("Spoken", quoted(v.Transcript), "On-screen text", quoted(v.OnScreenText), "About", v.About)
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

// uploadDir is where the API stores uploads. The default works when the worker runs
// from worker/ next to api/ (run.sh); Docker sets UPLOAD_DIR to the shared volume.
func uploadDir() string {
	return envOr("UPLOAD_DIR", filepath.Join("..", "api", "uploads"))
}

// AnalyzeVideo extracts the hook from a job's uploaded video. It never fails the job:
// problems are reported in Note and the simulation falls back to the text description.
func AnalyzeVideo(filename string) VideoInsight {
	if filename == "" {
		return VideoInsight{}
	}
	if len(loadGroqAPIKeys()) == 0 {
		return VideoInsight{Note: "no Groq API key is configured"}
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return VideoInsight{Note: "ffmpeg is not installed on the worker"}
	}

	// The API generates the filename, but never let it escape the upload directory.
	path := filepath.Join(uploadDir(), filepath.Base(filename))
	if _, err := os.Stat(path); err != nil {
		return VideoInsight{Note: "the uploaded file isn't reachable from the worker"}
	}

	tmp, err := os.MkdirTemp("", "viralyst-video-")
	if err != nil {
		return VideoInsight{Note: "couldn't create a temp directory"}
	}
	defer os.RemoveAll(tmp)

	var v VideoInsight
	var notes []string

	// Audio: a video with no audio track is normal, so only a failed transcription is noted.
	if audio, err := extractAudio(path, tmp); err == nil {
		if v.Transcript, err = transcribe(audio); err != nil {
			notes = append(notes, "audio transcription failed")
			logVideoErr("transcribe", filename, err)
		}
	}

	frames, err := extractFrames(path, tmp)
	if err == nil {
		err = describeFrames(frames, &v)
		if err != nil {
			notes = append(notes, "frame description failed")
			logVideoErr("vision", filename, err)
		}
	} else {
		notes = append(notes, "couldn't read video frames")
		logVideoErr("ffmpeg frames", filename, err)
	}

	if rest, err := summarizeRest(path, tmp); err != nil {
		notes = append(notes, "the rest of the video couldn't be summarised")
		logVideoErr("rest of video", filename, err)
	} else {
		v.Rest = rest
	}

	v.Note = strings.Join(notes, "; ")
	return v
}

func logVideoErr(step, filename string, err error) {
	log.Printf("[video] %s failed for %s: %v", step, filename, err)
}

func runFFmpeg(args ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "ffmpeg", append([]string{"-v", "error", "-y"}, args...)...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// extractAudio writes the opening seconds as 16 kHz mono WAV, the format Whisper uses.
func extractAudio(path, dir string) (string, error) {
	out := filepath.Join(dir, "hook.wav")
	if err := runFFmpeg("-i", path, "-t", fmt.Sprint(hookSeconds), "-vn", "-ac", "1", "-ar", "16000", out); err != nil {
		return "", err
	}
	return out, nil
}

// extractFrames grabs one frame per second of the hook and tiles them side by side into
// a single small image. Vision models bill per image pixel, and three separate 512px
// frames (~6k tokens) nearly used up Groq's free-tier 7k tokens/minute on their own.
func extractFrames(path, dir string) ([]string, error) {
	out := filepath.Join(dir, "frames.jpg")
	filter := fmt.Sprintf("fps=1,scale=320:-2,tile=%dx1", hookSeconds)
	if err := runFFmpeg("-i", path, "-t", fmt.Sprint(hookSeconds), "-vf", filter, "-frames:v", "1", "-q:v", "4", out); err != nil {
		return nil, err
	}
	if _, err := os.Stat(out); err != nil {
		return nil, errors.New("no frames decoded")
	}
	return []string{out}, nil
}

// probeDuration returns the video's length in seconds.
func probeDuration(path string) (float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=nw=1:nk=1", path).Output()
	if err != nil {
		return 0, err
	}
	return strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
}

// summarizeRest describes what happens after the opening, from the full transcript and
// frames spread across the video. Videos barely longer than the hook return "".
func summarizeRest(path, dir string) (string, error) {
	duration, err := probeDuration(path)
	if err != nil {
		return "", fmt.Errorf("ffprobe: %w", err)
	}
	if duration < hookSeconds+2 {
		return "", nil
	}
	if duration > storyMaxSeconds {
		duration = storyMaxSeconds
	}

	transcript := ""
	audio := filepath.Join(dir, "full.wav")
	if err := runFFmpeg("-i", path, "-t", fmt.Sprint(storyMaxSeconds), "-vn", "-ac", "1", "-ar", "16000", audio); err == nil {
		if transcript, err = transcribe(audio); err != nil {
			return "", fmt.Errorf("transcribe: %w", err)
		}
	}
	if transcript == "" {
		transcript = "(no speech)"
	}

	frames := filepath.Join(dir, "story.jpg")
	filter := fmt.Sprintf("fps=%d/%.2f,scale=256:-2,tile=3x2", storyFrames, duration)
	if err := runFFmpeg("-i", path, "-t", fmt.Sprint(storyMaxSeconds), "-vf", filter, "-frames:v", "1", "-q:v", "5", frames); err != nil {
		return "", fmt.Errorf("frames: %w", err)
	}
	img, err := os.ReadFile(frames)
	if err != nil {
		return "", err
	}

	var reply struct {
		Rest string `json:"rest"`
	}
	prompt := fmt.Sprintf(restPrompt, storyFrames, quoted(transcript))
	if _, err := visionJSON(prompt, [][]byte{img}, 300, &reply); err != nil {
		return "", err
	}
	return truncateWords(strings.TrimSpace(reply.Rest), maxRestWords), nil
}

// truncateWords keeps at most n words of s.
func truncateWords(s string, n int) string {
	words := strings.Fields(s)
	if len(words) <= n {
		return s
	}
	return strings.Join(words[:n], " ") + "…"
}

// whisperNonSpeech are Whisper's well-known outputs for music, noise or silence.
var whisperNonSpeech = map[string]bool{
	"music": true, "music playing": true, "upbeat music": true, "applause": true, "laughter": true,
	"silence": true, "thank you": true, "thanks for watching": true, "you": true, "bye": true,
}

// isNonSpeech reports whether a transcript is only a non-speech marker like "[Music]" or "♪".
func isNonSpeech(t string) bool {
	t = strings.ToLower(strings.TrimSpace(t))
	t = strings.Trim(t, "[]()*♪♫.!, ")
	return t == "" || whisperNonSpeech[t]
}

// transcribe sends audio to Groq's speech-to-text endpoint and returns the text.
func transcribe(audioPath string) (string, error) {
	audio, err := os.ReadFile(audioPath)
	if err != nil {
		return "", err
	}
	model := envOr("GROQ_TRANSCRIBE_MODEL", defaultTranscribeModel)

	var text string
	_, err = withGroqKeys(func(apiKey string) (int, time.Duration, error) {
		var body bytes.Buffer
		mw := multipart.NewWriter(&body)
		fw, err := mw.CreateFormFile("file", "hook.wav")
		if err != nil {
			return 0, 0, err
		}
		if _, err := fw.Write(audio); err != nil {
			return 0, 0, err
		}
		_ = mw.WriteField("model", model)
		_ = mw.WriteField("response_format", "json")
		if err := mw.Close(); err != nil {
			return 0, 0, err
		}

		req, err := http.NewRequest("POST", groqBaseURL+"/audio/transcriptions", &body)
		if err != nil {
			return 0, 0, err
		}
		req.Header.Set("Content-Type", mw.FormDataContentType())
		b, statusCode, retryAfter, err := doGroqRequest(mediaHTTPClient, req, apiKey)
		if err != nil {
			return statusCode, retryAfter, err
		}
		var tr struct {
			Text string `json:"text"`
		}
		if err := json.Unmarshal(b, &tr); err != nil {
			return statusCode, 0, fmt.Errorf("decode: %w", err)
		}
		text = strings.TrimSpace(tr.Text)
		if isNonSpeech(text) {
			text = ""
		}
		return statusCode, 0, nil
	})
	return text, err
}

// errNotJSON means the vision model answered, but not with the JSON asked for.
var errNotJSON = errors.New("vision reply is not JSON")

// visionJSON sends a prompt plus JPEG images to the vision model and decodes its JSON
// reply into out. It also returns the raw reply, so a caller can use non-JSON text.
func visionJSON(prompt string, images [][]byte, maxTokens int, out any) (string, error) {
	parts := []map[string]any{{"type": "text", "text": prompt}}
	for _, img := range images {
		parts = append(parts, map[string]any{
			"type":      "image_url",
			"image_url": map[string]string{"url": "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(img)},
		})
	}
	model := envOr("GROQ_VISION_MODEL", defaultVisionModel)
	payload := groqRequest{
		Model:          model,
		Messages:       []groqMessage{{Role: "user", Content: parts}},
		MaxTokens:      maxTokens,
		Temperature:    0.2,
		ResponseFormat: map[string]string{"type": "json_object"},
	}
	if strings.HasPrefix(model, "qwen/") {
		payload.ReasoningEffort = "none" // a plain description needs no reasoning tokens
	}

	var raw string
	_, err := withGroqKeys(func(apiKey string) (int, time.Duration, error) {
		c, statusCode, retryAfter, err := callGroqChatSingleKey(mediaHTTPClient, payload, apiKey)
		raw = c
		return statusCode, retryAfter, err
	})
	if err != nil {
		return "", err
	}
	start, end := strings.Index(raw, "{"), strings.LastIndex(raw, "}")
	if start < 0 || end <= start || json.Unmarshal([]byte(raw[start:end+1]), out) != nil {
		return raw, fmt.Errorf("%w: %q", errNotJSON, raw)
	}
	return raw, nil
}

// describeFrames asks a Groq vision model what a viewer sees and reads, filling v.
func describeFrames(frames []string, v *VideoInsight) error {
	images := [][]byte{}
	for _, f := range frames {
		img, err := os.ReadFile(f)
		if err != nil {
			return err
		}
		images = append(images, img)
	}
	var d struct {
		OnScreenText string `json:"on_screen_text"`
		Scene        string `json:"scene"`
		About        string `json:"about"`
	}
	raw, err := visionJSON(visionPrompt, images, 250, &d)
	if err != nil && !errors.Is(err, errNotJSON) {
		return err
	}
	if d.Scene == "" && d.About == "" && d.OnScreenText == "" {
		// Not the JSON we asked for: keep the reply as a plain scene description.
		d.Scene = raw
	}
	v.OnScreenText, v.Scene, v.About = strings.TrimSpace(d.OnScreenText), strings.TrimSpace(d.Scene), strings.TrimSpace(d.About)
	return nil
}
