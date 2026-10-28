package game

import (
	"strings"
	"testing"

	"rotmud/pkg/types"
)

func TestLookShowsExtraDescriptions(t *testing.T) {
	d := NewCommandDispatcher()
	var out string
	d.Output = func(ch *types.Character, msg string) { out += msg }

	room := types.NewRoom(1103, "Entrance to the Shire", "x")
	room.ExtraDescriptions = []*types.ExtraDescription{{Keywords: []string{"plaque"}, Description: "Bywater, 2 leagues."}}
	sign := types.NewObject(1102, "a sign", types.ItemTypeFurniture)
	sign.Name = "sign"
	sign.LongDesc = "A sign demands your attention."
	sign.ExtraDescriptions = []*types.ExtraDescription{{Keywords: []string{"sign"}, Description: "Welcome to the Shire."}}
	room.AddObject(sign)
	ch := types.NewCharacter("Test")
	ch.Position = types.PosStanding
	CharToRoom(ch, room)

	for _, cmd := range []func(*types.Character, string){d.cmdLook, d.cmdExamine} {
		out = ""
		cmd(ch, "sign")
		if !strings.Contains(out, "Welcome to the Shire.") {
			t.Errorf("sign text not shown: %q", out)
		}
	}
	out = ""
	d.cmdLook(ch, "plaque")
	if !strings.Contains(out, "Bywater, 2 leagues.") {
		t.Errorf("room extra description not shown: %q", out)
	}
}
