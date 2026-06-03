package llm

import (
	"fmt"
	"strings"
	"unicode"
)

// maxLineLen caps NPC output length (Tier 1 say/emote length cap). It is a
// safety rail against runaway generation, not the target length — the prompt
// asks the model for one or two sentences, which finish well under this. Set
// generously so normal replies are never truncated mid-sentence.
const maxLineLen = 400

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
	if reason := garbageReason(a.Line); reason != "" {
		return fmt.Errorf("line looks degenerate: %s", reason)
	}
	return nil
}

// garbageReason detects the failure modes small local models fall into —
// leaking markup/JSON tokens, emitting code or email-like strings, control
// characters, or repetition loops — and returns a short reason if the line is
// not plausible NPC speech. Empty string means the line is acceptable. When a
// line is rejected the Pool treats it as a failure and the caller falls back to
// scripted behavior (silent), which is preferable to printing garbage.
func garbageReason(line string) string {
	// Markup / structured-output / code leakage never appears in NPC speech.
	for _, bad := range []string{"`", "@", "{", "}", "</", "/>", "```", "json", "http", "\\u", "\\n"} {
		if strings.Contains(strings.ToLower(line), bad) {
			return "contains " + bad
		}
	}
	// Control characters (other than ordinary spaces) signal corruption.
	for _, r := range line {
		if r != '\t' && r != '\n' && r != '\r' && unicode.IsControl(r) {
			return "control character"
		}
	}
	// Repetition loop: any 16-char window repeated 3+ times.
	if isRepetitive(line) {
		return "repetition loop"
	}
	// Real speech has real words. A line with almost no letters (e.g. "1") is a
	// degenerate single-token output, not dialog.
	letters := 0
	for _, r := range line {
		if unicode.IsLetter(r) {
			letters++
		}
	}
	if letters < 2 {
		return "too few letters"
	}
	return ""
}

// isRepetitive reports whether a 16-character substring occurs three or more
// times — the signature of a degenerate generation loop.
func isRepetitive(s string) bool {
	const win = 16
	if len(s) < win*3 {
		return false
	}
	for i := 0; i+win <= len(s); i++ {
		if strings.Count(s, s[i:i+win]) >= 3 {
			return true
		}
	}
	return false
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
