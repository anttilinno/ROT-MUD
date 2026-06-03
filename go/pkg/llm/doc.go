// Package llm implements the LLM-driven NPC enhancement layer (Tier 1 dialog)
// described in .planning/LLM-NPC.md.
//
// The Go server is the single source of truth. The LLM never mutates game
// state directly: it returns a structured [Action] (a tool call), the server
// validates it, and only then acts. Any failure — endpoint down, timeout, bad
// JSON, illegal tool, or an open circuit breaker — surfaces as an error so the
// caller falls through to its scripted behavior. With the feature disabled the
// server behaves identically to before.
//
// # Async, never-blocking
//
// The game loop must never block on the LLM. [Pool] owns a worker goroutine
// set draining an inbox channel; results land on a result channel the game
// loop drains each tick. Per mob only one request is in flight at a time —
// a new [Request] for a mob already pending is dropped (overflow strategy).
//
// # Backend
//
// [Client] targets a llama.cpp server's OpenAI-compatible
// /v1/chat/completions endpoint using a grammar-constrained json_schema
// response format, so the model can only emit JSON matching the Tier 1 tool
// schema. (Ollama's looser /api/generate is intentionally not supported here;
// grammar constraint is what makes small local models reliable.)
package llm
