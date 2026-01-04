package game

import (
	"testing"

	"rotmud/pkg/types"
)

func TestCanStackObjects(t *testing.T) {
	t.Run("Same vnum and type can stack", func(t *testing.T) {
		obj1 := types.NewObject(100, "a gold coin", types.ItemTypeTreasure)
		obj2 := types.NewObject(100, "a gold coin", types.ItemTypeTreasure)

		if !CanStackObjects(obj1, obj2) {
			t.Error("identical objects should be stackable")
		}
	})

	t.Run("Different vnums cannot stack", func(t *testing.T) {
		obj1 := types.NewObject(100, "a gold coin", types.ItemTypeTreasure)
		obj2 := types.NewObject(101, "a silver coin", types.ItemTypeTreasure)

		if CanStackObjects(obj1, obj2) {
			t.Error("different vnum objects should not stack")
		}
	})

	t.Run("Different item types cannot stack", func(t *testing.T) {
		obj1 := types.NewObject(100, "a sword", types.ItemTypeWeapon)
		obj2 := types.NewObject(100, "a sword", types.ItemTypeArmor)

		if CanStackObjects(obj1, obj2) {
			t.Error("different item types should not stack")
		}
	})

	t.Run("Containers cannot stack", func(t *testing.T) {
		obj1 := types.NewObject(100, "a bag", types.ItemTypeContainer)
		obj2 := types.NewObject(100, "a bag", types.ItemTypeContainer)

		if CanStackObjects(obj1, obj2) {
			t.Error("containers should not stack")
		}
	})

	t.Run("Corpses cannot stack", func(t *testing.T) {
		obj1 := types.NewObject(100, "corpse of a goblin", types.ItemTypeCorpseNPC)
		obj2 := types.NewObject(100, "corpse of a goblin", types.ItemTypeCorpseNPC)

		if CanStackObjects(obj1, obj2) {
			t.Error("corpses should not stack")
		}
	})

	t.Run("Enchanted objects cannot stack", func(t *testing.T) {
		obj1 := types.NewObject(100, "a sword", types.ItemTypeWeapon)
		obj2 := types.NewObject(100, "a sword", types.ItemTypeWeapon)
		obj1.Enchanted = true

		if CanStackObjects(obj1, obj2) {
			t.Error("enchanted objects should not stack")
		}
	})

	t.Run("Objects with affects cannot stack", func(t *testing.T) {
		obj1 := types.NewObject(100, "a sword", types.ItemTypeWeapon)
		obj2 := types.NewObject(100, "a sword", types.ItemTypeWeapon)
		obj1.Affects.Add(&types.Affect{
			Type:     "hitroll",
			Modifier: 2,
		})

		if CanStackObjects(obj1, obj2) {
			t.Error("objects with affects should not stack")
		}
	})

	t.Run("Objects with different conditions cannot stack", func(t *testing.T) {
		obj1 := types.NewObject(100, "a sword", types.ItemTypeWeapon)
		obj2 := types.NewObject(100, "a sword", types.ItemTypeWeapon)
		obj1.Condition = 100
		obj2.Condition = 50

		if CanStackObjects(obj1, obj2) {
			t.Error("objects with different conditions should not stack")
		}
	})

	t.Run("Quest items cannot stack", func(t *testing.T) {
		obj1 := types.NewObject(100, "a quest item", types.ItemTypeTreasure)
		obj2 := types.NewObject(100, "a quest item", types.ItemTypeTreasure)
		obj1.ExtraFlags.Set(types.ItemQuest)
		obj2.ExtraFlags.Set(types.ItemQuest)

		if CanStackObjects(obj1, obj2) {
			t.Error("quest items should not stack")
		}
	})

	t.Run("Objects with timers cannot stack", func(t *testing.T) {
		obj1 := types.NewObject(100, "a sword", types.ItemTypeWeapon)
		obj2 := types.NewObject(100, "a sword", types.ItemTypeWeapon)
		obj1.Timer = 10 // Has a timer

		if CanStackObjects(obj1, obj2) {
			t.Error("objects with timers should not stack")
		}
	})

	t.Run("Objects with owner cannot stack", func(t *testing.T) {
		obj1 := types.NewObject(100, "a sword", types.ItemTypeWeapon)
		obj2 := types.NewObject(100, "a sword", types.ItemTypeWeapon)
		obj1.Owner = "Player1"

		if CanStackObjects(obj1, obj2) {
			t.Error("objects with owner should not stack")
		}
	})

	t.Run("Nil objects cannot stack", func(t *testing.T) {
		obj1 := types.NewObject(100, "a sword", types.ItemTypeWeapon)

		if CanStackObjects(obj1, nil) {
			t.Error("nil object should not stack")
		}
		if CanStackObjects(nil, obj1) {
			t.Error("nil object should not stack")
		}
	})
}

func TestStackObjects(t *testing.T) {
	t.Run("Empty list returns nil", func(t *testing.T) {
		stacks := StackObjects(nil)
		if stacks != nil {
			t.Error("expected nil for empty list")
		}
	})

	t.Run("Single object returns single stack", func(t *testing.T) {
		obj := types.NewObject(100, "a gold coin", types.ItemTypeTreasure)
		stacks := StackObjects([]*types.Object{obj})

		if len(stacks) != 1 {
			t.Fatalf("expected 1 stack, got %d", len(stacks))
		}
		if stacks[0].Count != 1 {
			t.Errorf("expected count 1, got %d", stacks[0].Count)
		}
	})

	t.Run("Identical objects stack together", func(t *testing.T) {
		obj1 := types.NewObject(100, "a gold coin", types.ItemTypeTreasure)
		obj2 := types.NewObject(100, "a gold coin", types.ItemTypeTreasure)
		obj3 := types.NewObject(100, "a gold coin", types.ItemTypeTreasure)

		stacks := StackObjects([]*types.Object{obj1, obj2, obj3})

		if len(stacks) != 1 {
			t.Fatalf("expected 1 stack, got %d", len(stacks))
		}
		if stacks[0].Count != 3 {
			t.Errorf("expected count 3, got %d", stacks[0].Count)
		}
	})

	t.Run("Different objects create separate stacks", func(t *testing.T) {
		coin1 := types.NewObject(100, "a gold coin", types.ItemTypeTreasure)
		coin2 := types.NewObject(100, "a gold coin", types.ItemTypeTreasure)
		sword := types.NewObject(200, "a sword", types.ItemTypeWeapon)

		stacks := StackObjects([]*types.Object{coin1, sword, coin2})

		if len(stacks) != 2 {
			t.Fatalf("expected 2 stacks, got %d", len(stacks))
		}

		// Find the coin stack
		var coinStack, swordStack *ObjectStack
		for i := range stacks {
			if stacks[i].Object.Vnum == 100 {
				coinStack = &stacks[i]
			} else {
				swordStack = &stacks[i]
			}
		}

		if coinStack == nil || coinStack.Count != 2 {
			t.Error("expected 2 coins in coin stack")
		}
		if swordStack == nil || swordStack.Count != 1 {
			t.Error("expected 1 sword in sword stack")
		}
	})
}

func TestFindStackedObjectsInInventory(t *testing.T) {
	t.Run("Find all matching objects", func(t *testing.T) {
		ch := types.NewCharacter("Test")

		coin1 := types.NewObject(100, "a gold coin", types.ItemTypeTreasure)
		coin1.Name = "coin gold"
		coin2 := types.NewObject(100, "a gold coin", types.ItemTypeTreasure)
		coin2.Name = "coin gold"
		sword := types.NewObject(200, "a sword", types.ItemTypeWeapon)
		sword.Name = "sword"

		ch.AddInventory(coin1)
		ch.AddInventory(coin2)
		ch.AddInventory(sword)

		coins := FindStackedObjectsInInventory(ch, "coin", 0)
		if len(coins) != 2 {
			t.Errorf("expected 2 coins, got %d", len(coins))
		}

		swords := FindStackedObjectsInInventory(ch, "sword", 0)
		if len(swords) != 1 {
			t.Errorf("expected 1 sword, got %d", len(swords))
		}
	})

	t.Run("Limit number of objects returned", func(t *testing.T) {
		ch := types.NewCharacter("Test")

		for i := 0; i < 5; i++ {
			coin := types.NewObject(100, "a gold coin", types.ItemTypeTreasure)
			coin.Name = "coin gold"
			ch.AddInventory(coin)
		}

		coins := FindStackedObjectsInInventory(ch, "coin", 3)
		if len(coins) != 3 {
			t.Errorf("expected 3 coins, got %d", len(coins))
		}
	})

	t.Run("Nil character returns nil", func(t *testing.T) {
		result := FindStackedObjectsInInventory(nil, "coin", 0)
		if result != nil {
			t.Error("expected nil for nil character")
		}
	})
}

func TestDropQuantity(t *testing.T) {
	t.Run("Drop multiple items", func(t *testing.T) {
		d := NewCommandDispatcher()
		var output string
		d.Output = func(ch *types.Character, msg string) {
			output += msg
		}

		ch := types.NewCharacter("Test")
		ch.Position = types.PosStanding
		room := types.NewRoom(3001, "Test Room", "A test room.")
		CharToRoom(ch, room)

		// Add 5 coins to inventory
		for i := 0; i < 5; i++ {
			coin := types.NewObject(100, "a gold coin", types.ItemTypeTreasure)
			coin.Name = "coin gold"
			coin.WearFlags.Set(types.WearTake)
			ObjToChar(coin, ch)
		}

		d.Dispatch(Command{Character: ch, Input: "drop 3 coin"})

		if len(ch.Inventory) != 2 {
			t.Errorf("expected 2 coins left in inventory, got %d", len(ch.Inventory))
		}
		if len(room.Objects) != 3 {
			t.Errorf("expected 3 coins in room, got %d", len(room.Objects))
		}
		if !contains(output, "You drop 3") {
			t.Errorf("expected 'You drop 3' message, got '%s'", output)
		}
	})

	t.Run("Drop single item without quantity", func(t *testing.T) {
		d := NewCommandDispatcher()
		var output string
		d.Output = func(ch *types.Character, msg string) {
			output += msg
		}

		ch := types.NewCharacter("Test")
		ch.Position = types.PosStanding
		room := types.NewRoom(3001, "Test Room", "A test room.")
		CharToRoom(ch, room)

		coin := types.NewObject(100, "a gold coin", types.ItemTypeTreasure)
		coin.Name = "coin gold"
		ObjToChar(coin, ch)

		d.Dispatch(Command{Character: ch, Input: "drop coin"})

		if len(ch.Inventory) != 0 {
			t.Errorf("expected empty inventory, got %d items", len(ch.Inventory))
		}
		if len(room.Objects) != 1 {
			t.Errorf("expected 1 coin in room, got %d", len(room.Objects))
		}
	})
}

func TestGetQuantity(t *testing.T) {
	t.Run("Get multiple items", func(t *testing.T) {
		d := NewCommandDispatcher()
		var output string
		d.Output = func(ch *types.Character, msg string) {
			output += msg
		}

		ch := types.NewCharacter("Test")
		ch.Position = types.PosStanding
		room := types.NewRoom(3001, "Test Room", "A test room.")
		CharToRoom(ch, room)

		// Add 5 coins to room
		for i := 0; i < 5; i++ {
			coin := types.NewObject(100, "a gold coin", types.ItemTypeTreasure)
			coin.Name = "coin gold"
			coin.WearFlags.Set(types.WearTake)
			ObjToRoom(coin, room)
		}

		d.Dispatch(Command{Character: ch, Input: "get 3 coin"})

		if len(ch.Inventory) != 3 {
			t.Errorf("expected 3 coins in inventory, got %d", len(ch.Inventory))
		}
		if len(room.Objects) != 2 {
			t.Errorf("expected 2 coins left in room, got %d", len(room.Objects))
		}
		if !contains(output, "You get 3") {
			t.Errorf("expected 'You get 3' message, got '%s'", output)
		}
	})
}

func TestGiveQuantity(t *testing.T) {
	t.Run("Give multiple items", func(t *testing.T) {
		d := NewCommandDispatcher()
		d.GameLoop = NewGameLoop()
		var output string
		d.Output = func(ch *types.Character, msg string) {
			output += msg
		}

		giver := types.NewCharacter("Giver")
		giver.Position = types.PosStanding
		receiver := types.NewCharacter("Receiver")
		receiver.Position = types.PosStanding

		room := types.NewRoom(3001, "Test Room", "A test room.")
		CharToRoom(giver, room)
		CharToRoom(receiver, room)

		d.GameLoop.AddCharacter(giver)
		d.GameLoop.AddCharacter(receiver)

		// Add 5 coins to giver
		for i := 0; i < 5; i++ {
			coin := types.NewObject(100, "a gold coin", types.ItemTypeTreasure)
			coin.Name = "coin gold"
			ObjToChar(coin, giver)
		}

		d.Dispatch(Command{Character: giver, Input: "give 3 coin Receiver"})

		if len(giver.Inventory) != 2 {
			t.Errorf("expected 2 coins left with giver, got %d", len(giver.Inventory))
		}
		if len(receiver.Inventory) != 3 {
			t.Errorf("expected 3 coins with receiver, got %d", len(receiver.Inventory))
		}
		if !contains(output, "You give 3") {
			t.Errorf("expected 'You give 3' message, got '%s'", output)
		}
	})
}

func TestInventoryStackedDisplay(t *testing.T) {
	t.Run("Inventory shows stacked items with count", func(t *testing.T) {
		d := NewCommandDispatcher()
		var output string
		d.Output = func(ch *types.Character, msg string) {
			output += msg
		}

		ch := types.NewCharacter("Test")
		ch.Position = types.PosStanding
		ch.Comm.Set(types.CommCombine) // Enable combine mode

		// Add 5 identical coins
		for i := 0; i < 5; i++ {
			coin := types.NewObject(100, "a gold coin", types.ItemTypeTreasure)
			coin.Name = "coin gold"
			ObjToChar(coin, ch)
		}

		// Add a sword
		sword := types.NewObject(200, "a sword", types.ItemTypeWeapon)
		sword.Name = "sword"
		ObjToChar(sword, ch)

		d.Dispatch(Command{Character: ch, Input: "inventory"})

		// Should show "(5) a gold coin" not 5 separate lines
		if !contains(output, "( 5)") {
			t.Errorf("expected '( 5)' for stacked coins, got '%s'", output)
		}
		if !contains(output, "a gold coin") {
			t.Errorf("expected 'a gold coin' in output, got '%s'", output)
		}
		if !contains(output, "a sword") {
			t.Errorf("expected 'a sword' in output, got '%s'", output)
		}
	})

	t.Run("Inventory without combine shows individual items", func(t *testing.T) {
		d := NewCommandDispatcher()
		var output string
		d.Output = func(ch *types.Character, msg string) {
			output += msg
		}

		ch := types.NewCharacter("Test")
		ch.Position = types.PosStanding
		// Note: CommCombine is NOT set

		// Add 3 identical coins
		for i := 0; i < 3; i++ {
			coin := types.NewObject(100, "a gold coin", types.ItemTypeTreasure)
			coin.Name = "coin gold"
			ObjToChar(coin, ch)
		}

		d.Dispatch(Command{Character: ch, Input: "inventory"})

		// Should show each coin separately (no count prefix)
		// Count occurrences of "a gold coin" - should be 3
		count := 0
		for i := 0; i <= len(output)-len("a gold coin"); i++ {
			if output[i:i+len("a gold coin")] == "a gold coin" {
				count++
			}
		}
		if count != 3 {
			t.Errorf("expected 3 occurrences of 'a gold coin', got %d in '%s'", count, output)
		}
	})
}
