package game

import (
	"testing"

	"rotmud/pkg/types"
)

func TestSacrificeReward(t *testing.T) {
	sword := types.NewObject(3021, "a small sword", types.ItemTypeWeapon)
	sword.Cost = 250
	if got := SacrificeReward(sword); got != 1 {
		t.Errorf("level 0 sword: %d, want 1", got)
	}
	sword.Level = 20
	if got := SacrificeReward(sword); got != 60 {
		t.Errorf("level 20 sword: %d, want 60", got)
	}
	sword.Cost = 10
	if got := SacrificeReward(sword); got != 10 {
		t.Errorf("capped at value: %d, want 10", got)
	}
	corpse := types.NewObject(0, "the corpse of the blob", types.ItemTypeCorpseNPC)
	corpse.Level = 50
	if got := SacrificeReward(corpse); got != 150 {
		t.Errorf("level 50 corpse: %d, want 150 (no value cap)", got)
	}
}
