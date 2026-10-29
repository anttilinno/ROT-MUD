package combat

import (
	"testing"

	"rotmud/pkg/types"
)

// ROT check_assist: an assist_vnum mob joins its kin's fight (half the
// rounds), an unflagged mob stays out, and an autoassisting groupmate
// joins its leader's fight every round.
func TestCheckAssist(t *testing.T) {
	c := NewCombatSystem()
	room := types.NewRoom(1, "Test", "Test room")
	put := func(ch *types.Character) *types.Character {
		ch.InRoom = room
		room.AddPerson(ch)
		ch.Hit, ch.MaxHit = 10000, 10000
		return ch
	}
	player := put(types.NewCharacter("Player"))
	player.Level = 10
	mate := put(types.NewCharacter("Mate"))
	mate.Level = 10
	mate.PlayerAct.Set(types.PlrAutoAssist)
	mob := put(types.NewNPC(100, "guard", 10))
	kin := put(types.NewNPC(100, "guard", 10))
	kin.Off.Set(types.OffAssistVnum)
	other := put(types.NewNPC(200, "rat", 10))

	joined := 0
	for range 50 {
		for _, ch := range room.People {
			ch.Fighting, ch.Position = nil, types.PosStanding
		}
		mob.Fighting = player
		c.checkAssist(mob, player)
		if kin.Fighting == player {
			joined++
		}
		if other.Fighting != nil || mate.Fighting != nil {
			t.Fatal("a mob without assist flags, or a player, joined the mob's side")
		}
	}
	if joined == 0 || joined == 50 {
		t.Errorf("assist_vnum mob joined %d of 50 rounds, want about half", joined)
	}

	for _, ch := range room.People {
		ch.Fighting, ch.Position = nil, types.PosStanding
	}
	mate.Leader = player
	player.Fighting = mob
	c.checkAssist(player, mob)
	if mate.Fighting != mob {
		t.Error("autoassisting groupmate did not join the leader's fight")
	}
}
