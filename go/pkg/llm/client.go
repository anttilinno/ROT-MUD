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
// is the system message; playerSay is injected as escaped data inside a quoted
// sentence (never as instructions) to blunt prompt injection. A non-nil error
// means the caller must fall through to scripted behavior.
//
// Chat does NOT call Action.Validate — the Pool does, so validation failures
// are counted toward the circuit breaker alongside transport failures.
func (c *Client) Chat(ctx context.Context, persona, playerName, playerSay string) (Action, error) {
	userMsg := fmt.Sprintf("A character named %s says to you: %q", playerName, playerSay)

	reqBody, _ := json.Marshal(map[string]any{
		"model": c.model,
		"messages": []map[string]string{
			{"role": "system", "content": persona},
			{"role": "user", "content": userMsg},
		},
		"temperature":     0.8,
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
