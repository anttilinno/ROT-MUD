// Command llmnpc-spike is a throwaway proof of the LLM-driven NPC dialog loop.
//
// It does NOT touch the MUD. It exercises the Tier 1 path from LLM-NPC.md:
// build a persona + world-state prompt, hit a local LLM endpoint, parse the
// model's structured tool-call JSON, validate it, and fall back to a scripted
// line on timeout / unreachable / garbage. Run the MUD server separately; this
// just proves the dialog loop before any integration is written.
//
// Backends:
//
//   - llamacpp (default): llama.cpp server OpenAI-compatible /v1/chat/completions.
//     Uses grammar-constrained json_schema sampling so the model CANNOT emit
//     invalid output — the schema is pinned at the sampler (LLM-NPC.md leans on
//     this exact technique). Start it with:
//     llama-server -m model.gguf --host 127.0.0.1 --port 8080 -ngl 99 --jinja
//
//   - ollama: /api/generate with format=json (looser; model may still wander).
//
//     go run ./cmd/llmnpc-spike -say "I want to change 50 gold"
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// persona is the fixed identity injected every turn as the system message.
// Mirrors the TOML `persona` field described in LLM-NPC.md Tier 1.
const persona = `You are Otho, the money changer in the city of Midgaard.
You are gruff, impatient, and care only about coin. You speak in short,
clipped sentences. You charge a 2% fee. You never break character and never
mention being an AI. Give the words Otho says aloud.`

// scriptedFallback is the canned line used whenever the LLM path fails for any
// reason (unreachable, timeout, bad JSON, invalid tool). The MUD must behave
// identically with the LLM off — this stands in for that scripted behavior.
const scriptedFallback = `Otho says 'I don't deal in that kind of thing. Move along.'`

// action is the validated tool call the server would dispatch. Tier 1 surface.
type action struct {
	Tool string `json:"tool"`
	Line string `json:"line"`
}

func main() {
	backend := flag.String("backend", "llamacpp", "llamacpp | ollama")
	endpoint := flag.String("endpoint", "", "base URL (default :8080 for llamacpp, :11434 for ollama)")
	model := flag.String("model", "qwen", "model name (ollama needs the real tag)")
	playerSay := flag.String("say", "Greetings. Can you change 50 gold for me?", "what the player says to the NPC")
	budget := flag.Duration("budget", 800*time.Millisecond, "latency budget before scripted fallback")
	flag.Parse()

	if *endpoint == "" {
		if *backend == "ollama" {
			*endpoint = "http://localhost:11434"
		} else {
			*endpoint = "http://127.0.0.1:8080"
		}
	}

	fmt.Printf("== llmnpc-spike ==\nNPC:     Otho the money changer\nBackend: %s @ %s\nModel:   %s\nBudget:  %s\nPlayer:  %q\n\n",
		*backend, *endpoint, *model, *budget, *playerSay)

	act, raw, lat, err := askLLM(*backend, *endpoint, *model, *playerSay, *budget)
	if raw != "" {
		fmt.Printf("[latency %s] raw model output: %s\n", lat.Round(time.Millisecond), raw)
	}
	if err != nil {
		fmt.Printf("[LLM path failed: %v]\n", err)
		fmt.Printf("[fallback] %s\n", scriptedFallback)
		return
	}
	if reason := validate(act); reason != "" {
		fmt.Printf("[tool-call rejected: %s]\n", reason)
		fmt.Printf("[fallback] %s\n", scriptedFallback)
		return
	}

	fmt.Printf("[LLM ok] tool=%s\n", act.Tool)
	switch act.Tool {
	case "say":
		fmt.Printf("Otho says '%s'\n", act.Line)
	case "emote":
		fmt.Printf("Otho %s\n", act.Line)
	case "refuse":
		fmt.Printf("Otho refuses: '%s'\n", act.Line)
	}
}

// askLLM dispatches to the chosen backend. Returns the parsed action, the raw
// model text (for display), latency, and an error. Any error means the caller
// must fall through to the scripted line.
func askLLM(backend, endpoint, model, playerSay string, budget time.Duration) (action, string, time.Duration, error) {
	ctx, cancel := context.WithTimeout(context.Background(), budget)
	defer cancel()

	// Player input is injected as escaped DATA inside a quoted sentence, never
	// as instructions, to blunt prompt injection (LLM-NPC.md risk table).
	userMsg := fmt.Sprintf("A traveler says to you: %q", playerSay)

	var body []byte
	switch backend {
	case "ollama":
		body, _ = json.Marshal(map[string]any{
			"model":  model,
			"system": persona,
			"prompt": userMsg + "\n\nReply with ONLY a JSON object: {\"tool\":\"say\"|\"emote\"|\"refuse\",\"line\":\"<words>\"}",
			"stream": false,
			"format": "json",
		})
	default: // llamacpp OpenAI-compatible + grammar-constrained schema
		body, _ = json.Marshal(map[string]any{
			"model": model,
			"messages": []map[string]string{
				{"role": "system", "content": persona},
				{"role": "user", "content": userMsg},
			},
			"temperature":     0.8,
			"response_format": tier1Schema(),
		})
	}

	path := "/v1/chat/completions"
	if backend == "ollama" {
		path = "/api/generate"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint+path, bytes.NewReader(body))
	if err != nil {
		return action{}, "", 0, err
	}
	req.Header.Set("Content-Type", "application/json")

	start := time.Now()
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return action{}, "", time.Since(start), fmt.Errorf("endpoint unreachable/timeout: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	lat := time.Since(start)
	if resp.StatusCode != http.StatusOK {
		return action{}, "", lat, fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	content, err := extractContent(backend, raw)
	if err != nil {
		return action{}, "", lat, err
	}

	var act action
	if err := json.Unmarshal([]byte(content), &act); err != nil {
		return action{}, content, lat, fmt.Errorf("unparseable tool call: %w", err)
	}
	return act, content, lat, nil
}

// tier1Schema returns the OpenAI response_format that llama.cpp compiles into a
// GBNF grammar — the sampler can only produce JSON matching this shape.
func tier1Schema() map[string]any {
	return map[string]any{
		"type": "json_schema",
		"json_schema": map[string]any{
			"name":   "npc_action",
			"strict": true,
			"schema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"tool": map[string]any{"type": "string", "enum": []string{"say", "emote", "refuse"}},
					"line": map[string]any{"type": "string", "maxLength": 200},
				},
				"required":             []string{"tool", "line"},
				"additionalProperties": false,
			},
		},
	}
}

// extractContent pulls the model's text payload out of the backend envelope.
func extractContent(backend string, raw []byte) (string, error) {
	if backend == "ollama" {
		var r struct {
			Response string `json:"response"`
		}
		if err := json.Unmarshal(raw, &r); err != nil {
			return "", fmt.Errorf("bad envelope: %w", err)
		}
		return r.Response, nil
	}
	var r struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &r); err != nil {
		return "", fmt.Errorf("bad envelope: %w", err)
	}
	if len(r.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}
	return r.Choices[0].Message.Content, nil
}

// validate is the authority gate: the server trusts no LLM output. Returns a
// non-empty reason string if the tool call is illegal. Tier 1 surface only.
func validate(a action) string {
	switch a.Tool {
	case "say", "emote", "refuse":
	default:
		return fmt.Sprintf("unknown tool %q", a.Tool)
	}
	if strings.TrimSpace(a.Line) == "" {
		return "empty line"
	}
	if len(a.Line) > 200 {
		return fmt.Sprintf("line too long (%d > 200)", len(a.Line))
	}
	return ""
}
