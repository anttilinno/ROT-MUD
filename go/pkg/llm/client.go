package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Client talks to a llama.cpp OpenAI-compatible endpoint. Safe for concurrent
// use; the underlying http.Client pools connections.
type Client struct {
	endpoint string
	model    string
	http     *http.Client
}

// NewClient builds a Client. The per-request timeout is supplied via the
// context passed to Chat, not here, so a single Client serves all workers.
func NewClient(endpoint, model string) *Client {
	return &Client{
		endpoint: strings.TrimRight(endpoint, "/"),
		model:    model,
		http:     &http.Client{},
	}
}

// Chat sends one turn and returns the parsed, schema-shaped Action. The persona
// is the system message; the player's speech is injected as escaped data inside a
// quoted sentence (never as instructions) to blunt prompt injection. Observed
// state and world context are appended as the NPC's own perception. A non-nil
// error means the caller must fall through to scripted behavior.
//
// Chat does NOT call Action.Validate — the Pool does, so validation failures
// are counted toward the circuit breaker alongside transport failures.
// antiHallucinationRule is appended to every persona. It keeps small local
// models from inventing world facts: when asked about something they were not
// told (distant lands, specific monsters, lore, directions beyond what is
// observed), they deflect in character instead of making things up.
const antiHallucinationRule = "\n\nIMPORTANT: Only speak of what your character would truly know plus what you are explicitly told you observe about the speaker and your immediate surroundings. If asked about anything beyond that — far-off places, specific monsters, lore, or events you were not given — do NOT invent facts or names. Instead admit, briefly and in character, that you do not know (a forgetful old soul whose memory has faded, or simply someone who has not heard such things). Never fabricate details to seem helpful."

func (c *Client) Chat(ctx context.Context, r Request) (Action, error) {
	// playerState/worldContext are appended as observed context (the NPC's own
	// perception), kept separate from quoted speech so they are never read as
	// instructions.
	var observed string
	if r.PlayerState != "" {
		observed = fmt.Sprintf(" You observe that %s %s.", r.PlayerName, r.PlayerState)
	}
	var world string
	if r.WorldContext != "" {
		world = " " + r.WorldContext
	}

	var userMsg string
	if r.Greeting {
		userMsg = fmt.Sprintf("A character named %s has just walked into your presence. Greet them in character. You may offer a brief word of wisdom or a hint about the surroundings, but keep it to one or two sentences.%s%s", r.PlayerName, observed, world)
	} else {
		userMsg = fmt.Sprintf("A character named %s says to you: %q. Reply in character in one or two short sentences.%s%s", r.PlayerName, r.PlayerSay, observed, world)
	}

	reqBody, _ := json.Marshal(map[string]any{
		"model": c.model,
		"messages": []map[string]string{
			{"role": "system", "content": r.Persona + antiHallucinationRule},
			{"role": "user", "content": userMsg},
		},
		// Lower temperature plus a repetition penalty keeps small local models
		// from degenerating into loops / token leakage. max_tokens caps a runaway
		// generation before it can fill the whole grammar window with garbage.
		"temperature":     0.5,
		"repeat_penalty":  1.15,
		"max_tokens":      160,
		"response_format": tier1Schema(),
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+"/v1/chat/completions", bytes.NewReader(reqBody))
	if err != nil {
		return Action{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return Action{}, fmt.Errorf("endpoint unreachable/timeout: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return Action{}, fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var env struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return Action{}, fmt.Errorf("bad envelope: %w", err)
	}
	if len(env.Choices) == 0 {
		return Action{}, fmt.Errorf("no choices in response")
	}

	var act Action
	if err := json.Unmarshal([]byte(env.Choices[0].Message.Content), &act); err != nil {
		return Action{}, fmt.Errorf("unparseable tool call: %w", err)
	}
	return act, nil
}
