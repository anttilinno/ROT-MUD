package game

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"rotmud/pkg/llm"
	"rotmud/pkg/types"
)

// TestLive_SayTriggersLLMMob is a full-pipeline check of the Tier 1 dialog glue
// against a running llama.cpp server: a player's `say` in a room with an
// LLM-enabled mob must produce an in-character reply delivered back to the
// player. It exercises cmdSay -> notifyLLMMobs -> Pool -> drainLLM ->
// OnLLMResult. Opt-in: skips unless ROTMUD_LLM=1.
//
//	ROTMUD_LLM=1 go test ./pkg/game/ -run TestLive -v
func TestLive_SayTriggersLLMMob(t *testing.T) {
	cfg := llm.ConfigFromEnv()
	if !cfg.Enabled {
		t.Skip("set ROTMUD_LLM=1 (and run llama-server) to run the live test")
	}

	// Capture everything sent to each character.
	var mu sync.Mutex
	out := map[*types.Character]*strings.Builder{}
	capture := func(ch *types.Character, msg string) {
		mu.Lock()
		defer mu.Unlock()
		b := out[ch]
		if b == nil {
			b = &strings.Builder{}
			out[ch] = b
		}
		b.WriteString(msg)
	}
	seen := func(ch *types.Character) string {
		mu.Lock()
		defer mu.Unlock()
		if b := out[ch]; b != nil {
			return b.String()
		}
		return ""
	}

	d := NewCommandDispatcher()
	d.Output = capture

	pool := llm.NewPool(cfg)
	pool.Start()
	defer pool.Stop()
	d.LLM = pool

	// A game loop just to drive the result-draining glue under test.
	gl := NewGameLoop()
	gl.LLM = pool
	gl.OnLLMResult = func(res llm.Result) {
		mob, ok := res.Key.(*types.Character)
		if !ok || mob.InRoom == nil || res.Err != nil {
			return
		}
		name := mob.ShortDesc
		var line string
		switch res.Action.Tool {
		case "say", "refuse":
			line = fmt.Sprintf("%s says '%s'\r\n", name, res.Action.Line)
		case "emote":
			line = fmt.Sprintf("%s %s\r\n", name, res.Action.Line)
		}
		for _, p := range mob.InRoom.People {
			if !p.IsNPC() {
				capture(p, line)
			}
		}
	}

	// One room, one player, one LLM-enabled Otho.
	room := types.NewRoom(3334, "The Money Changer", "Otho's counting house.")
	otho := types.NewNPC(3162, "otho money changer", 50)
	otho.ShortDesc = "Otho the Changer"
	otho.LLMEnabled = true
	otho.LLMPersona = "You are Otho, a gruff, greedy money changer in Midgaard. " +
		"Speak in short clipped sentences, in character. You charge a 2% fee. " +
		"Give only the words Otho says aloud."
	player := types.NewCharacter("Conan")
	player.Level = 10
	CharToRoom(otho, room)
	CharToRoom(player, room)

	// Player speaks. This broadcasts and enqueues the LLM turn for Otho.
	d.cmdSay(player, "Can you change 50 gold for silver?")

	// Drain results until Otho speaks (or budget elapses).
	deadline := time.Now().Add(cfg.Timeout + 3*time.Second)
	for time.Now().Before(deadline) {
		gl.drainLLM()
		if strings.Contains(seen(player), "Otho the Changer") {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	got := seen(player)
	if !strings.Contains(got, "You say 'Can you change 50 gold") {
		t.Errorf("player should see their own say; got:\n%s", got)
	}
	if !strings.Contains(got, "Otho the Changer") {
		t.Fatalf("Otho did not respond via LLM within budget; player saw:\n%s", got)
	}
	t.Logf("player transcript:\n%s", got)
}
