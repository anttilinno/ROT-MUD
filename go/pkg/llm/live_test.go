package llm

import (
	"testing"
	"time"
)

// TestLive_OthoDialog exercises the real Client → Pool → breaker → validation
// path against a running llama.cpp server. Opt-in: it skips unless ROTMUD_LLM
// is set, so the normal `go test ./...` run never touches the network.
//
//	ROTMUD_LLM=1 go test ./pkg/llm/ -run TestLive -v
func TestLive_OthoDialog(t *testing.T) {
	cfg := ConfigFromEnv()
	if !cfg.Enabled {
		t.Skip("set ROTMUD_LLM=1 (and run llama-server) to run the live test")
	}

	pool := NewPool(cfg)
	pool.Start()
	defer pool.Stop()

	const persona = "You are Otho, a gruff, greedy money changer in Midgaard. " +
		"Speak in short clipped sentences, in character. You charge a 2% fee. " +
		"Give only the words Otho says aloud."

	if !pool.Submit(Request{
		Key:        "otho",
		Persona:    persona,
		PlayerName: "Conan",
		PlayerSay:  "Can you change 50 gold for silver?",
	}) {
		t.Fatal("Submit dropped the request")
	}

	select {
	case res := <-pool.Results():
		if res.Err != nil {
			t.Fatalf("live LLM call failed: %v", res.Err)
		}
		t.Logf("Otho (%s): %q", res.Action.Tool, res.Action.Line)
		if err := res.Action.Validate(); err != nil {
			t.Fatalf("returned action is invalid: %v", err)
		}
	case <-time.After(cfg.Timeout + 2*time.Second):
		t.Fatal("timed out waiting for live result")
	}
}
