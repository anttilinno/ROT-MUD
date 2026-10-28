package game

import (
	"testing"

	"rotmud/pkg/types"
)

// TestGetCoinsFromCorpseGoesToPurse: coins looted from any corpse are added to
// the purse, not carried as an inventory item.
func TestGetCoinsFromCorpseGoesToPurse(t *testing.T) {
	d := NewCommandDispatcher()
	d.Output = func(ch *types.Character, msg string) {}
	d.GameLoop = NewGameLoop()

	room := types.NewRoom(3700, "Room", "x")
	ch := types.NewCharacter("Test")
	ch.Position = types.PosStanding
	CharToRoom(ch, room)

	corpse := types.NewObject(10, "the corpse of the blob", types.ItemTypeCorpseNPC)
	corpse.Name = "corpse blob"
	coins := types.NewObject(0, "1g 1s 6c in coins", types.ItemTypeMoney)
	coins.Name = "coins money"
	coins.Cost = 116
	corpse.AddContent(coins)
	ObjToRoom(corpse, room)

	d.cmdGet(ch, "all corpse")

	if ch.Coin != 116 {
		t.Errorf("coin = %d, want 116", ch.Coin)
	}
	if len(ch.Inventory) != 0 {
		t.Errorf("coins carried as an item: %d inventory objects", len(ch.Inventory))
	}

	// Coins lying on the floor can be picked up too.
	floor := types.NewObject(0, "2s in coins", types.ItemTypeMoney)
	floor.Name = "coins money"
	floor.Cost = 20
	floor.WearFlags.Set(types.WearTake)
	ObjToRoom(floor, room)
	d.cmdGet(ch, "coins")
	if ch.Coin != 136 {
		t.Errorf("coin = %d after floor pickup, want 136", ch.Coin)
	}
}
