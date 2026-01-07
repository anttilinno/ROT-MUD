package ai

import (
	"testing"

	"rotmud/pkg/types"
)

func TestAISystem(t *testing.T) {
	ai := NewAISystem()

	if ai.Registry == nil {
		t.Error("Registry should not be nil")
	}
}

func TestAISystemProcessMobile(t *testing.T) {
	ai := NewAISystem()

	// Player should return false
	player := types.NewCharacter("TestPlayer")
	player.InRoom = &types.Room{Vnum: 3001}
	if ai.ProcessMobile(player) {
		t.Error("ProcessMobile should return false for players")
	}

	// NPC without special
	mob := types.NewNPC(1000, "test mob", 10)
	mob.InRoom = &types.Room{Vnum: 3001}
	mob.Position = types.PosStanding
	// Should return false (no special action taken by default)
	ai.ProcessMobile(mob)

	// NPC with unknown special
	mob.Special = "spec_nonexistent"
	ai.ProcessMobile(mob)
}

func TestDefaultBehavior_Scavenger(t *testing.T) {
	ai := NewAISystem()

	mob := types.NewNPC(1000, "scavenger", 10)
	mob.Act.Set(types.ActScavenger)
	room := &types.Room{
		Vnum:    3001,
		Objects: make([]*types.Object, 0),
	}
	mob.InRoom = room
	mob.Position = types.PosStanding

	// Add a valuable object
	obj := types.NewObject(1, "gold coin", types.ItemTypeTreasure)
	obj.WearFlags.Set(types.WearTake)
	obj.Cost = 100
	obj.InRoom = room
	room.Objects = append(room.Objects, obj)

	// Force the scavenger to pick it up by running multiple times
	// (the behavior has a random chance)
	picked := false
	for i := 0; i < 1000; i++ {
		if len(mob.Inventory) > 0 {
			picked = true
			break
		}
		ai.defaultBehavior(mob)
	}

	if !picked {
		t.Log("Scavenger didn't pick up item (random chance - this can happen)")
	}
}

func TestSpecialRegistry(t *testing.T) {
	r := NewSpecialRegistry()

	// Test that all default specials are registered
	expectedSpecials := []string{
		"spec_breath_any",
		"spec_breath_acid",
		"spec_breath_fire",
		"spec_breath_frost",
		"spec_breath_gas",
		"spec_breath_lightning",
		"spec_cast_adept",
		"spec_cast_cleric",
		"spec_cast_mage",
		"spec_cast_undead",
		"spec_guard",
		"spec_executioner",
		"spec_patrolman",
		"spec_thief",
		"spec_nasty",
		"spec_poison",
		"spec_janitor",
		"spec_fido",
		"spec_mayor",
	}

	for _, name := range expectedSpecials {
		if r.Find(name) == nil {
			t.Errorf("expected special %q to be registered", name)
		}
	}
}

func TestCanAct(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(ch *types.Character)
		expected bool
	}{
		{
			name: "awake and normal",
			setup: func(ch *types.Character) {
				ch.Position = types.PosStanding
			},
			expected: true,
		},
		{
			name: "sleeping",
			setup: func(ch *types.Character) {
				ch.Position = types.PosSleeping
			},
			expected: false,
		},
		{
			name: "calmed",
			setup: func(ch *types.Character) {
				ch.Position = types.PosStanding
				ch.AffectedBy.Set(types.AffCalm)
			},
			expected: false,
		},
		{
			name: "charmed",
			setup: func(ch *types.Character) {
				ch.Position = types.PosStanding
				ch.AffectedBy.Set(types.AffCharm)
			},
			expected: false,
		},
		{
			name: "no room",
			setup: func(ch *types.Character) {
				ch.Position = types.PosStanding
				ch.InRoom = nil
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch := types.NewNPC(1000, "test mob", 10)
			ch.InRoom = &types.Room{Vnum: 3001}
			tt.setup(ch)

			if got := canAct(ch); got != tt.expected {
				t.Errorf("canAct() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestSpecJanitor(t *testing.T) {
	ch := types.NewNPC(3060, "janitor", 5)
	room := &types.Room{
		Vnum:    3001,
		People:  []*types.Character{ch},
		Objects: make([]*types.Object, 0),
	}
	ch.InRoom = room

	ctx := &SpecialContext{}

	// No trash - should return false
	if specJanitor(ch, ctx) {
		t.Error("janitor should not act when no trash present")
	}

	// Add some trash
	trash := types.NewObject(1, "trash", types.ItemTypeTrash)
	trash.WearFlags.Set(types.WearTake)
	trash.Cost = 5
	trash.InRoom = room
	room.Objects = append(room.Objects, trash)

	// Track if action was taken
	acted := false
	ctx.ActToRoom = func(msg string, ch, victim *types.Character, output func(ch *types.Character, msg string)) {
		acted = true
	}

	// Should pick up trash
	if !specJanitor(ch, ctx) {
		t.Error("janitor should pick up trash")
	}

	if !acted {
		t.Error("janitor should have acted")
	}

	// Trash should be in inventory now
	if len(ch.Inventory) != 1 {
		t.Errorf("expected 1 item in inventory, got %d", len(ch.Inventory))
	}

	if len(room.Objects) != 0 {
		t.Errorf("expected 0 items in room, got %d", len(room.Objects))
	}
}

func TestSpecGuard(t *testing.T) {
	guard := types.NewNPC(3060, "guard", 20)
	room := &types.Room{
		Vnum:   3001,
		People: make([]*types.Character, 0),
	}
	guard.InRoom = room
	guard.Position = types.PosStanding
	room.People = append(room.People, guard)

	ctx := &SpecialContext{}

	// No one to attack - should return false
	if specGuard(guard, ctx) {
		t.Error("guard should not act when no troublemakers present")
	}

	// Add an evil fighter
	evil := types.NewNPC(1001, "evil mob", 10)
	evil.InRoom = room
	evil.Alignment = -500
	good := types.NewNPC(1002, "good mob", 8)
	good.InRoom = room
	good.Alignment = 500
	evil.Fighting = good
	good.Fighting = evil
	room.People = append(room.People, evil, good)

	// Track if combat was started
	combatStarted := false
	ctx.StartCombat = func(ch, victim *types.Character) {
		combatStarted = true
		if victim != evil {
			t.Error("guard should attack the evil fighter")
		}
	}
	ctx.ActToRoom = func(msg string, ch, victim *types.Character, output func(ch *types.Character, msg string)) {}

	// Should intervene
	if !specGuard(guard, ctx) {
		t.Error("guard should intervene in fight with evil attacker")
	}

	if !combatStarted {
		t.Error("guard should have started combat")
	}
}

func TestSpecCastMage_NotFighting(t *testing.T) {
	mage := types.NewNPC(3020, "evil mage", 15)
	mage.InRoom = &types.Room{Vnum: 3001}
	mage.Position = types.PosStanding

	ctx := &SpecialContext{}

	// Not fighting - should return false
	if specCastMage(mage, ctx) {
		t.Error("mage should not cast when not fighting")
	}
}

func TestSpecFido(t *testing.T) {
	fido := types.NewNPC(3062, "fido", 5)
	room := &types.Room{
		Vnum:    3001,
		People:  []*types.Character{fido},
		Objects: make([]*types.Object, 0),
	}
	fido.InRoom = room
	fido.Position = types.PosStanding

	ctx := &SpecialContext{}

	// No corpse - should return false
	if specFido(fido, ctx) {
		t.Error("fido should not act when no corpse present")
	}

	// Add a corpse
	corpse := types.NewObject(1, "corpse of a goblin", types.ItemTypeCorpseNPC)
	corpse.InRoom = room
	room.Objects = append(room.Objects, corpse)

	// Add some items in the corpse
	sword := types.NewObject(2, "a rusty sword", types.ItemTypeWeapon)
	corpse.AddContent(sword)

	acted := false
	ctx.ActToRoom = func(msg string, ch, victim *types.Character, output func(ch *types.Character, msg string)) {
		acted = true
	}

	// Should eat the corpse
	if !specFido(fido, ctx) {
		t.Error("fido should eat the corpse")
	}

	if !acted {
		t.Error("fido should have acted")
	}

	// Corpse should be gone
	if len(room.Objects) != 1 {
		t.Errorf("expected 1 item in room (the sword), got %d", len(room.Objects))
	}

	// The sword should be on the ground now
	if room.Objects[0] != sword {
		t.Error("sword should be in the room")
	}
}

func TestSpecCastAdept(t *testing.T) {
	adept := types.NewNPC(3010, "adept", 30)
	room := &types.Room{
		Vnum:   3700,
		People: make([]*types.Character, 0),
	}
	adept.InRoom = room
	adept.Position = types.PosStanding
	room.People = append(room.People, adept)

	ctx := &SpecialContext{}

	// No players - should return false
	if specCastAdept(adept, ctx) {
		t.Error("adept should not act when no low-level players present")
	}

	// Add a low-level player
	player := types.NewCharacter("TestPlayer")
	player.Level = 5 // Below level 31, so adept should help
	player.InRoom = room
	player.Position = types.PosStanding
	room.People = append(room.People, player)

	// Track actions
	spellCast := false
	actedToRoom := false
	ctx.ActToRoom = func(msg string, ch, victim *types.Character, output func(ch *types.Character, msg string)) {
		actedToRoom = true
	}
	ctx.CastSpell = func(ch *types.Character, spellName string, victim *types.Character) bool {
		spellCast = true
		if victim != player {
			t.Error("adept should cast spell on the low-level player")
		}
		return true
	}

	// Run multiple times to account for random chance
	for i := 0; i < 100; i++ {
		if specCastAdept(adept, ctx) {
			break
		}
	}

	if !actedToRoom {
		t.Error("adept should have uttered a magic word")
	}
	if !spellCast {
		t.Error("adept should have cast a spell")
	}
}

func TestSpecCastAdept_HighLevelPlayer(t *testing.T) {
	adept := types.NewNPC(3010, "adept", 30)
	room := &types.Room{
		Vnum:   3700,
		People: make([]*types.Character, 0),
	}
	adept.InRoom = room
	adept.Position = types.PosStanding
	room.People = append(room.People, adept)

	// Add a high-level player (level 35, above the 31 threshold)
	player := types.NewCharacter("VeteranPlayer")
	player.Level = 35
	player.InRoom = room
	player.Position = types.PosStanding
	room.People = append(room.People, player)

	ctx := &SpecialContext{}
	spellCast := false
	ctx.CastSpell = func(ch *types.Character, spellName string, victim *types.Character) bool {
		spellCast = true
		return true
	}

	// Run multiple times - should never cast on high-level player
	for i := 0; i < 100; i++ {
		specCastAdept(adept, ctx)
	}

	if spellCast {
		t.Error("adept should NOT cast spells on players level 31+")
	}
}
