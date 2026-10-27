package ai

import (
	"testing"

	"rotmud/pkg/magic"
	"rotmud/pkg/types"
)

// TestCastSpellTargetsVictim checks a mob's defensive spell lands on the
// chosen victim, not the caster (the mud school adept buffed herself).
func TestCastSpellTargetsVictim(t *testing.T) {
	a := NewAISystem()
	a.Magic = magic.NewMagicSystem()

	room := types.NewRoom(3700, "Cage Room", "x")
	adept := types.NewCharacter("adept cleric")
	adept.Act.Set(types.ActNPC)
	adept.Level, adept.Mana, adept.Position = 60, 1000, types.PosStanding
	player := types.NewCharacter("Aldor")
	player.Position = types.PosStanding
	for _, c := range []*types.Character{adept, player} {
		room.AddPerson(c)
		c.InRoom = room
	}

	// NPC casts fizzle 1% of the time; a few tries keeps this deterministic enough.
	for i := 0; i < 5 && !magic.IsAffectedBy(player, "armor"); i++ {
		a.castSpellWrapper(adept, "armor", player)
	}
	if !magic.IsAffectedBy(player, "armor") {
		t.Error("armor did not land on the victim")
	}
	if magic.IsAffectedBy(adept, "armor") {
		t.Error("armor landed on the caster")
	}
}
