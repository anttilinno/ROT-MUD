package game

import (
	"strings"
	"testing"

	"rotmud/pkg/types"
)

func TestTrainNeedsTrainerNotGuildmaster(t *testing.T) {
	d := NewCommandDispatcher()
	var out string
	d.Output = func(ch *types.Character, msg string) { out += msg }
	d.GameLoop = NewGameLoop()

	room := types.NewRoom(3758, "Zump's Guild Room", "x")
	priest := types.NewCharacter("priest")
	priest.Act.Set(types.ActNPC)
	priest.Act.Set(types.ActPractice)
	ch := types.NewCharacter("Aldor")
	ch.PCData = &types.PCData{}
	ch.Position = types.PosStanding
	CharToRoom(priest, room)
	CharToRoom(ch, room)

	d.cmdTrain(ch, "str")
	if !strings.Contains(out, "can't do that here") {
		t.Errorf("guildmaster should not train stats, got %q", out)
	}
}
