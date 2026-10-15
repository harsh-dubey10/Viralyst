// anthropic.go
// Claude in the LLM rotation, through the official Anthropic Go SDK. Persona decisions,
// topic reading and hook rating can land here; video analysis stays on Groq.

package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/anthropics/anthropic-sdk-go/shared/constant"
)

// defaultAnthropicModel is Claude Opus 5.5; override with ANTHROPIC_MODEL.
const defaultAnthropicModel = "claude-opus-5-5"

// anthropicMaxTokens leaves room for low-effort thinking plus the short JSON reply.
const anthropicMaxTokens = 4000

var (
	anthropicKeys     []string
	anthropicKeysOnce sync.Once
	anthropicClients  sync.Map // API key → anthropic.Client
)

// loadAnthropicAPIKeys discovers ANTHROPIC_API_KEY, ANTHROPIC_API_KEY_1..10 and ANTHROPIC_API_KEYS.
func loadAnthropicAPIKeys() []string {
	anthropicKeysOnce.Do(func() {
		anthropicKeys = discoverKeys("ANTHROPIC")
		if len(anthropicKeys) > 0 {
			log.Printf("[anthropic] Loaded %d Anthropic API key(s) into the LLM rotation", len(anthropicKeys))
		}
	})
	return anthropicKeys
}

func anthropicModel() string {
	return envOr("ANTHROPIC_MODEL", defaultAnthropicModel)
}

// anthropicClient returns a client per key. The SDK's own retries are off: a 429 or
// overload returns at once so the rotation can fail over to another key.
func anthropicClient(key string) anthropic.Client {
	if c, ok := anthropicClients.Load(key); ok {
		return c.(anthropic.Client)
	}
	opts := []option.RequestOption{
		option.WithAPIKey(key),
		option.WithMaxRetries(0),
		option.WithRequestTimeout(90 * time.Second),
	}
	// Organization-level keys (not scoped to one workspace) must name the workspace to
	// bill: its wrkspc_… ID from the Anthropic Console.
	if ws := envOr("ANTHROPIC_WORKSPACE_ID", ""); ws != "" {
		opts = append(opts, option.WithHeader("anthropic-workspace-id", ws))
	}
	c := anthropic.NewClient(opts...)
	anthropicClients.Store(key, c)
	return c
}

// callAnthropicSingle runs one chat payload on Claude with a specific key.
// Returns (content, statusCode, retryAfter, error), like callChatSingle.
//
// Claude Opus 5.5 always thinks; effort "low" keeps these short JSON calls quick. It
// rejects sampling parameters, so the payload's temperature is not sent, and JSON is
// asked for in the prompt (callers already extract the {...} from the reply). Refusals
// are re-served by a fallback model in the same call (server-side fallbacks).
func callAnthropicSingle(payload groqRequest, ep llmEndpoint) (string, int, time.Duration, error) {
	var system []anthropic.BetaTextBlockParam
	var messages []anthropic.BetaMessageParam
	for _, m := range payload.Messages {
		text, ok := m.Content.(string)
		if !ok {
			return "", 0, 0, fmt.Errorf("anthropic: non-text message content isn't supported here")
		}
		if m.Role == "system" {
			system = append(system, anthropic.BetaTextBlockParam{Text: text})
			continue
		}
		messages = append(messages, anthropic.NewBetaUserMessage(anthropic.NewBetaTextBlock(text)))
	}

	params := anthropic.BetaMessageNewParams{
		Model:        anthropic.Model(ep.model),
		MaxTokens:    anthropicMaxTokens,
		System:       system,
		Messages:     messages,
		OutputConfig: anthropic.BetaOutputConfigParam{Effort: anthropic.BetaOutputConfigEffortLow},
		Fallbacks:    anthropic.BetaFallbacksParamUnion{OfDefault: constant.ValueOf[constant.Default]()},
		Betas:        []anthropic.AnthropicBeta{anthropic.AnthropicBetaServerSideFallback2026_07_01},
	}

	client := anthropicClient(ep.key)
	resp, err := client.Beta.Messages.New(context.Background(), params)
	if err != nil {
		var apierr *anthropic.Error
		if errors.As(err, &apierr) {
			var retryAfter time.Duration
			if apierr.Response != nil {
				if secs, perr := strconv.ParseFloat(apierr.Response.Header.Get("Retry-After"), 64); perr == nil {
					retryAfter = time.Duration(secs * float64(time.Second))
				}
			}
			status := apierr.StatusCode
			if status == http.StatusUnauthorized || status == http.StatusForbidden || strings.Contains(err.Error(), "anthropic-workspace-id") {
				// Every call with this key would fail the same way (bad key, or an
				// org-level key without ANTHROPIC_WORKSPACE_ID): take the provider out of
				// the rotation instead of failing over on each call.
				status = http.StatusNotFound
			}
			return "", status, retryAfter, fmt.Errorf("anthropic status %d: %w", apierr.StatusCode, err)
		}
		return "", 0, 0, fmt.Errorf("anthropic: %w", err)
	}
	if resp.StopReason == anthropic.BetaStopReasonRefusal {
		return "", 0, 0, fmt.Errorf("anthropic: request declined (%s)", resp.StopDetails.Category)
	}

	var out strings.Builder
	for _, block := range resp.Content {
		if t, ok := block.AsAny().(anthropic.BetaTextBlock); ok {
			out.WriteString(t.Text)
		}
	}
	if out.Len() == 0 {
		return "", 0, 0, fmt.Errorf("anthropic: no text in reply (stop reason %s)", resp.StopReason)
	}
	return out.String(), 200, 0, nil
}
