package game

import (
	"testing"

	"rotmud/pkg/magic"
	"rotmud/pkg/types"
)

func recallFixture() (*CommandDispatcher, *types.Character, *types.Room, *types.Room, *types.Room) {
	d := NewCommandDispatcher()
	d.Output = func(ch *types.Character, msg string) {}
	d.GameLoop = NewGameLoop()
	temple := types.NewRoom(RecallVnum, "Temple", "x")
	inn := types.NewRoom(5000, "Inn", "x")
	field := types.NewRoom(6000, "Field", "x")
	for _, r := range []*types.Room{temple, inn, field} {
		d.GameLoop.Rooms[r.Vnum] = r
	}
	d.Magic.RoomFinder = func(vnum int) *types.Room { return d.GameLoop.Rooms[vnum] }

	ch := types.NewCharacter("Test")
	ch.PCData = &types.PCData{Learned: map[string]int{}}
	ch.Position = types.PosStanding
	ch.Move = 100
	CharToRoom(ch, inn)
	return d, ch, temple, inn, field
}

// Word of recall (as a scroll calls it: spell.Func directly) goes to the
// bound room and ends combat; the recall command still goes to Midgaard.
func TestWordOfRecallUsesBoundRoom(t *testing.T) {
	d, ch, temple, inn, field := recallFixture()
	d.cmdBind(ch, "")
	if ch.PCData.Recall != inn.Vnum {
		t.Fatalf("bind set recall %d, want %d", ch.PCData.Recall, inn.Vnum)
	}

	mob := types.NewCharacter("goblin")
	CharFromRoom(ch)
	CharToRoom(ch, field)
	CharToRoom(mob, field)
	ch.Fighting, mob.Fighting = mob, ch

	d.Magic.Registry.FindBySlot(magic.SlotWordOfRecall).Func(ch, 30, ch)
	if ch.InRoom != inn {
		t.Fatalf("word of recall went to %v, want the bound inn", ch.InRoom.Name)
	}
	if mob.Fighting != nil {
		t.Error("opponent still fighting after word of recall")
	}

	CharFromRoom(ch)
	CharToRoom(ch, field)
	d.cmdRecall(ch, "")
	if ch.InRoom != temple {
		t.Errorf("recall command went to %v, want Midgaard temple", ch.InRoom.Name)
	}
}

func TestBindRefusedInNoRecallRoom(t *testing.T) {
	d, ch, _, inn, _ := recallFixture()
	inn.Flags.Set(types.RoomNoRecall)
	d.cmdBind(ch, "")
	if ch.PCData.Recall != 0 {
		t.Errorf("bound in a no_recall room: %d", ch.PCData.Recall)
	}
}
