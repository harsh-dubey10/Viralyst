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
	"strings"
	"time"
)

// hookSeconds is how much of the video counts as the hook.
const hookSeconds = 3

const (
	defaultVisionModel     = "qwen/qwen3.8-27b" // the only image-capable model Groq serves as of Oct 2026
	defaultTranscribeModel = "whisper-large-v3-turbo"
)

// Vision and transcription requests carry media, so they get a longer timeout.
var mediaHTTPClient = &http.Client{Timeout: 60 * time.Second}

const visionPrompt = "These frames are the first 3 seconds of a short social media video, in order. " +
	"In at most 2 sentences, describe what a viewer sees: the subject, the action, and the setting. " +
	"Quote any on-screen text exactly. Describe only what is visible; do not judge or guess intent."

// VideoInsight is what was extracted from a video's opening seconds.
type VideoInsight struct {
	Transcript string // spoken words, if any
	Visual     string // description of the frames
	Note       string // why the video couldn't be (fully) analysed; empty on success
}

// Hook is the video's opening as text, or "" if nothing was extracted.
func (v VideoInsight) Hook() string {
	parts := []string{}
	if v.Transcript != "" {
		parts = append(parts, fmt.Sprintf("Spoken: %q", v.Transcript))
	}
	if v.Visual != "" {
		parts = append(parts, "On screen: "+v.Visual)
	}
	return strings.Join(parts, "\n")
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
		v.Visual, err = describeFrames(frames)
		if err != nil {
			notes = append(notes, "frame description failed")
			logVideoErr("vision", filename, err)
		}
	} else {
		notes = append(notes, "couldn't read video frames")
		logVideoErr("ffmpeg frames", filename, err)
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

// extractFrames grabs one frame per second of the hook, scaled down to keep requests small.
func extractFrames(path, dir string) ([]string, error) {
	pattern := filepath.Join(dir, "frame%d.jpg")
	if err := runFFmpeg("-i", path, "-t", fmt.Sprint(hookSeconds), "-vf", "fps=1,scale=512:-2", "-frames:v", fmt.Sprint(hookSeconds), "-q:v", "4", pattern); err != nil {
		return nil, err
	}
	frames, _ := filepath.Glob(filepath.Join(dir, "frame*.jpg"))
	if len(frames) == 0 {
		return nil, errors.New("no frames decoded")
	}
	return frames, nil
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
		return statusCode, 0, nil
	})
	return text, err
}

// describeFrames asks a Groq vision model what a viewer sees in the frames.
func describeFrames(frames []string) (string, error) {
	parts := []map[string]any{{"type": "text", "text": visionPrompt}}
	for _, f := range frames {
		img, err := os.ReadFile(f)
		if err != nil {
			return "", err
		}
		parts = append(parts, map[string]any{
			"type":      "image_url",
			"image_url": map[string]string{"url": "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(img)},
		})
	}
	model := envOr("GROQ_VISION_MODEL", defaultVisionModel)
	payload := groqRequest{
		Model:       model,
		Messages:    []groqMessage{{Role: "user", Content: parts}},
		MaxTokens:   200,
		Temperature: 0.2,
	}
	if strings.HasPrefix(model, "qwen/") {
		payload.ReasoningEffort = "none" // a plain description needs no reasoning tokens
	}

	var desc string
	_, err := withGroqKeys(func(apiKey string) (int, time.Duration, error) {
		c, statusCode, retryAfter, err := callGroqChatSingleKey(mediaHTTPClient, payload, apiKey)
		desc = strings.TrimSpace(c)
		return statusCode, retryAfter, err
	})
	return desc, err
}
