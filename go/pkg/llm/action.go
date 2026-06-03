package llm

import (
	"fmt"
	"strings"
)

// maxLineLen caps NPC output length (Tier 1 say/emote length cap).
const maxLineLen = 200

// Action is a validated tool call returned by the model. Tier 1 surface only:
// say, emote, refuse. The game server dispatches this; it never trusts the raw
// model output until Validate passes.
type Action struct {
	Tool string `json:"tool"` // "say" | "emote" | "refuse"
	Line string `json:"line"` // the words / emote text
}

// Validate is the authority gate. Returns a non-nil error describing why the
// action is illegal, or nil if the server may dispatch it.
func (a Action) Validate() error {
	switch a.Tool {
	case "say", "emote", "refuse":
	default:
		return fmt.Errorf("unknown tool %q", a.Tool)
	}
	if strings.TrimSpace(a.Line) == "" {
		return fmt.Errorf("empty line")
	}
	if len(a.Line) > maxLineLen {
		return fmt.Errorf("line too long (%d > %d)", len(a.Line), maxLineLen)
	}
	return nil
}

// tier1Schema is the OpenAI response_format that llama.cpp compiles into a GBNF
// grammar, pinning the sampler to JSON matching the Action shape.
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
					"line": map[string]any{"type": "string", "maxLength": maxLineLen},
				},
				"required":             []string{"tool", "line"},
				"additionalProperties": false,
			},
		},
	}
}
