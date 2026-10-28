package game

import (
	"strings"
	"testing"

	"rotmud/pkg/types"
)

func TestBriefHidesDescriptionOnMove(t *testing.T) {
	d := NewCommandDispatcher()
	var out string
	d.Output = func(ch *types.Character, msg string) { out += msg }
	d.GameLoop = NewGameLoop()

	room := types.NewRoom(100, "Mud School", "Paintings adorn the walls.")
	ch := types.NewCharacter("Aldor")
	ch.Position = types.PosStanding
	CharToRoom(ch, room)
	ch.Comm.Set(types.CommBrief)

	d.doLook(ch, "auto")
	if strings.Contains(out, "Paintings") {
		t.Errorf("brief auto-look showed description: %q", out)
	}

	out = ""
	d.doLook(ch, "")
	if !strings.Contains(out, "Paintings") {
		t.Errorf("typed look should show description in brief mode: %q", out)
	}
}
