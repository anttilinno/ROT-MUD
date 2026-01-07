package game

import (
	"testing"

	"rotmud/pkg/types"
)

func TestCreatePet(t *testing.T) {
	master := types.NewCharacter("TestMaster")
	master.Level = 30
	master.Alignment = 500 // Good alignment

	config := PetConfig{
		MobVnum:     MOBVnumDemon,
		Name:        "test demon",
		ShortDesc:   "a test demon",
		LongDesc:    "A test demon stands here.\r\n",
		Description: "It's a test demon.\r\n",
		Level:       15,
		MaxHit:      150,
		Alignment:   -500,
		ArmorMult:   2,
		ExoticMult:  3,
	}

	pet := CreatePet(master, config)
	if pet == nil {
		t.Fatal("CreatePet returned nil")
	}

	// Verify pet properties
	if pet.Name != "test demon" {
		t.Errorf("Expected name 'test demon', got %q", pet.Name)
	}
	if pet.Level != 15 {
		t.Errorf("Expected level 15, got %d", pet.Level)
	}
	if pet.MaxHit != 150 {
		t.Errorf("Expected max hit 150, got %d", pet.MaxHit)
	}
	if pet.Hit != 150 {
		t.Errorf("Expected hit 150, got %d", pet.Hit)
	}
	if pet.Alignment != -500 {
		t.Errorf("Expected alignment -500, got %d", pet.Alignment)
	}

	// Verify pet flags
	if !pet.Act.Has(types.ActPet) {
		t.Error("Pet should have ActPet flag")
	}
	if !pet.Act.Has(types.ActNPC) {
		t.Error("Pet should have ActNPC flag")
	}
	if !pet.AffectedBy.Has(types.AffCharm) {
		t.Error("Pet should have AffCharm flag")
	}

	// Verify follower relationship
	if pet.Master != master {
		t.Error("Pet master should be set")
	}
	if pet.Leader != master {
		t.Error("Pet leader should be set")
	}
	if master.Pet != pet {
		t.Error("Master's pet should be set")
	}

	// Verify armor
	expectedAC := 15 / 2 // level / ArmorMult
	if pet.Armor[types.ACPierce] != expectedAC {
		t.Errorf("Expected AC pierce %d, got %d", expectedAC, pet.Armor[types.ACPierce])
	}
}

func TestCreateZombiePet(t *testing.T) {
	master := types.NewCharacter("Necromancer")
	master.Level = 40
	master.Alignment = -500 // Evil alignment

	corpseShortDesc := "the corpse of a goblin"

	pet := CreateZombiePet(master, master.Level, 20, corpseShortDesc)
	if pet == nil {
		t.Fatal("CreateZombiePet returned nil")
	}

	// Zombie level should be average of caster (40) and corpse (20) = 30
	if pet.Level != 30 {
		t.Errorf("Expected zombie level 30, got %d", pet.Level)
	}

	// Zombie max HP should be level * 15
	expectedHP := 30 * 15
	if pet.MaxHit != expectedHP {
		t.Errorf("Expected max hit %d, got %d", expectedHP, pet.MaxHit)
	}

	// Short desc should have "zombie" instead of "corpse"
	if pet.ShortDesc != "the zombie of a goblin" {
		t.Errorf("Expected short desc 'the zombie of a goblin', got %q", pet.ShortDesc)
	}
}

func TestCreateDemonPet(t *testing.T) {
	master := types.NewCharacter("Warlock")
	master.Level = 35
	master.Alignment = -750 // Evil alignment

	pet := CreateDemonPet(master)
	if pet == nil {
		t.Fatal("CreateDemonPet returned nil")
	}

	// Demon level should match master
	if pet.Level != 35 {
		t.Errorf("Expected demon level 35, got %d", pet.Level)
	}

	// Demon max HP should be level * 30
	expectedHP := 35 * 30
	if pet.MaxHit != expectedHP {
		t.Errorf("Expected max hit %d, got %d", expectedHP, pet.MaxHit)
	}

	// Demon inherits master's alignment
	if pet.Alignment != -750 {
		t.Errorf("Expected alignment -750, got %d", pet.Alignment)
	}
}

func TestCreateAnimateFollower(t *testing.T) {
	master := types.NewCharacter("Necromancer")
	master.Level = 30
	master.Alignment = -500

	bodyPart := types.NewObject(15, "a severed head", types.ItemTypeTrash)
	bodyPart.Name = "head severed"
	bodyPart.LongDesc = "A severed head is lying here."

	follower := CreateAnimateFollower(master, bodyPart)
	if follower == nil {
		t.Fatal("CreateAnimateFollower returned nil")
	}

	// Animate level should be master level / 2
	expectedLevel := master.Level / 2
	if follower.Level != expectedLevel {
		t.Errorf("Expected level %d, got %d", expectedLevel, follower.Level)
	}

	// Follower should have body part's name
	if follower.Name != bodyPart.Name {
		t.Errorf("Expected name %q, got %q", bodyPart.Name, follower.Name)
	}

	// KEY DIFFERENCE: animate does NOT set master.Pet
	if master.Pet != nil {
		t.Error("CreateAnimateFollower should NOT set master.Pet")
	}

	// But follower should still have master reference
	if follower.Master != master {
		t.Error("Follower should have master set")
	}
	if follower.Leader != master {
		t.Error("Follower should have leader set")
	}

	// Follower should have AffCharm but NOT ActPet
	if !follower.AffectedBy.Has(types.AffCharm) {
		t.Error("Follower should have AffCharm flag")
	}
	if follower.Act.Has(types.ActPet) {
		t.Error("Animated follower should NOT have ActPet flag")
	}
}

func TestMultipleAnimatedFollowers(t *testing.T) {
	master := types.NewCharacter("Necromancer")
	master.Level = 30

	bodyPart1 := types.NewObject(15, "a severed head", types.ItemTypeTrash)
	bodyPart1.Name = "head severed"
	bodyPart1.LongDesc = "A severed head is lying here."

	bodyPart2 := types.NewObject(12, "a brain", types.ItemTypeTrash)
	bodyPart2.Name = "brain"
	bodyPart2.LongDesc = "A brain is lying here."

	// Create first animated follower
	follower1 := CreateAnimateFollower(master, bodyPart1)
	if follower1 == nil {
		t.Fatal("First animate failed")
	}

	// Create second animated follower - should succeed since animate doesn't set Pet
	follower2 := CreateAnimateFollower(master, bodyPart2)
	if follower2 == nil {
		t.Fatal("Second animate should succeed - animate creates followers, not pets")
	}

	// Master should still have no pet
	if master.Pet != nil {
		t.Error("Master should have no pet after animating")
	}

	// Both followers should reference the master
	if follower1.Master != master {
		t.Error("Follower1 should have master set")
	}
	if follower2.Master != master {
		t.Error("Follower2 should have master set")
	}
}

func TestOnlyOnePet(t *testing.T) {
	master := types.NewCharacter("Master")
	master.Level = 30

	// Create first pet
	pet1 := CreateDemonPet(master)
	if pet1 == nil {
		t.Fatal("First pet creation failed")
	}

	// Try to create second pet - should fail
	pet2 := CreateDemonPet(master)
	if pet2 != nil {
		t.Error("Should not be able to create second pet")
	}

	// Master should still have the first pet
	if master.Pet != pet1 {
		t.Error("Master's pet reference corrupted")
	}
}

func TestDismissPet(t *testing.T) {
	master := types.NewCharacter("Master")
	master.Level = 30

	room := types.NewRoom(1000, "Test Room", "A test room.")

	pet := CreateDemonPet(master)
	if pet == nil {
		t.Fatal("Pet creation failed")
	}

	// Place pet in room
	pet.InRoom = room
	room.AddPerson(pet)

	// Dismiss the pet
	DismissPet(master)

	// Verify master no longer has a pet
	if master.Pet != nil {
		t.Error("Master should have no pet after dismiss")
	}

	// Verify pet is removed from room
	if pet.InRoom != nil {
		t.Error("Pet should be removed from room")
	}

	// Verify pet has no master
	if pet.Master != nil {
		t.Error("Pet should have no master")
	}
}

func TestPetHelpers(t *testing.T) {
	master := types.NewCharacter("Master")
	master.Level = 30

	pet := CreateDemonPet(master)
	if pet == nil {
		t.Fatal("Pet creation failed")
	}

	// Test IsPet
	if !IsPet(pet) {
		t.Error("IsPet should return true for pet")
	}

	// Test IsPet on master (should be false)
	if IsPet(master) {
		t.Error("IsPet should return false for non-pet")
	}

	// Test GetPetMaster
	petMaster := GetPetMaster(pet)
	if petMaster != master {
		t.Error("GetPetMaster should return master")
	}

	// Test GetPetMaster on non-pet
	if GetPetMaster(master) != nil {
		t.Error("GetPetMaster should return nil for non-pet")
	}
}

func TestPetFollowMaster(t *testing.T) {
	master := types.NewCharacter("Master")
	master.Level = 30

	room1 := types.NewRoom(1000, "Room 1", "First room.")
	room2 := types.NewRoom(1001, "Room 2", "Second room.")

	pet := CreateDemonPet(master)
	if pet == nil {
		t.Fatal("Pet creation failed")
	}

	// Place pet in room1
	pet.InRoom = room1
	room1.AddPerson(pet)

	// Move pet to room2
	PetFollowMaster(pet, room2)

	// Verify pet is in room2
	if pet.InRoom != room2 {
		t.Error("Pet should be in room2")
	}

	// Verify pet is in room2's people list
	found := false
	for _, p := range room2.People {
		if p == pet {
			found = true
			break
		}
	}
	if !found {
		t.Error("Pet should be in room2's people list")
	}
}
