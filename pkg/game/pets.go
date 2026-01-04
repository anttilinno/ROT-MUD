package game

import (
	"fmt"
	"strings"

	"rotmud/pkg/types"
)

// Pet system constants
// These VNUMs match the C ROT implementation
const (
	MOBVnumCorpse  = 2 // Zombie from spell_resurrect
	MOBVnumAnimate = 3 // Animated body part from spell_animate
	MOBVnumDemon   = 4 // Demon from spell_conjure
)

// PetConfig holds configuration for creating a pet
type PetConfig struct {
	MobVnum     int    // Base mobile VNUM
	Name        string // Pet name/keywords
	ShortDesc   string // Short description
	LongDesc    string // Room description
	Description string // Full description
	Level       int    // Pet level
	MaxHit      int    // Max HP
	Alignment   int    // Alignment
	ArmorMult   int    // Armor multiplier (level/ArmorMult for AC)
	ExoticMult  int    // Exotic armor multiplier (level/ExoticMult for exotic AC)
}

// CreatePet creates a new pet mobile and sets it up as a follower
// Returns the created pet or nil on failure
func CreatePet(master *types.Character, config PetConfig) *types.Character {
	if master == nil {
		return nil
	}

	// Check if master already has a pet
	if master.Pet != nil {
		return nil
	}

	// Create the pet character
	pet := types.NewNPC(config.MobVnum, config.Name, config.Level)

	// Set descriptions
	pet.ShortDesc = config.ShortDesc
	pet.LongDesc = config.LongDesc
	pet.Desc = config.Description

	// Set as pet and charmed
	pet.Act.Set(types.ActPet)
	pet.AffectedBy.Set(types.AffCharm)

	// Set communication flags (can't communicate)
	pet.Comm.Set(types.CommNoTell)
	pet.Comm.Set(types.CommNoShout)
	pet.Comm.Set(types.CommNoChannels)

	// Set stats
	pet.Level = config.Level
	pet.MaxHit = config.MaxHit
	pet.Hit = pet.MaxHit
	pet.Alignment = config.Alignment

	// Set armor
	if config.ArmorMult > 0 {
		acValue := config.Level / config.ArmorMult
		pet.Armor[types.ACPierce] = acValue
		pet.Armor[types.ACBash] = acValue
		pet.Armor[types.ACSlash] = acValue
	}
	if config.ExoticMult > 0 {
		pet.Armor[types.ACExotic] = config.Level / config.ExoticMult
	}

	// Set up follower relationship
	pet.Master = master
	pet.Leader = master
	master.Pet = pet

	return pet
}

// AddFollower sets up a character as a follower (without making it a pet)
func AddFollower(ch *types.Character, master *types.Character) {
	if ch == nil || master == nil {
		return
	}

	// Stop following anyone else
	if ch.Master != nil {
		ch.Master = nil
	}

	ch.Master = master
	ch.Leader = master
}

// StopFollower removes a character from following their master
func StopFollower(ch *types.Character) {
	if ch == nil || ch.Master == nil {
		return
	}

	// If this was a pet, clear the pet reference
	if ch.Master.Pet == ch {
		ch.Master.Pet = nil
	}

	ch.Master = nil
	ch.Leader = nil
}

// DismissPet removes and destroys a character's pet
// This is called when a player logs out, dies, or dismisses their pet
func DismissPet(master *types.Character) {
	if master == nil || master.Pet == nil {
		return
	}

	pet := master.Pet

	// Remove from room
	if pet.InRoom != nil {
		pet.InRoom.RemovePerson(pet)
		pet.InRoom = nil
	}

	// Clear relationships
	pet.Master = nil
	pet.Leader = nil
	master.Pet = nil
}

// CreateZombiePet creates a zombie from a corpse (spell_resurrect)
// casterLevel is the caster's level, corpseLevel is the level of the original mob
func CreateZombiePet(master *types.Character, casterLevel int, corpseLevel int, corpseShortDesc string) *types.Character {
	// Calculate zombie level: average of caster and corpse level
	level := (casterLevel + corpseLevel) / 2
	if level < 1 {
		level = 1
	}
	if level > 109 {
		level = 109
	}

	// Transform corpse description to zombie
	shortDesc := strings.ReplaceAll(corpseShortDesc, "corpse", "zombie")
	name := "zombie " + strings.ReplaceAll(shortDesc, "the ", "")

	// Build long description from short desc
	longDesc := shortDesc + " standing here.\r\n"

	// Build full description with master's mark
	description := fmt.Sprintf("%s\r\n{GThe mark of %s is on its forehead.{x\r\n",
		shortDesc, master.Name)

	config := PetConfig{
		MobVnum:     MOBVnumCorpse,
		Name:        name,
		ShortDesc:   shortDesc,
		LongDesc:    longDesc,
		Description: description,
		Level:       level,
		MaxHit:      level * 15,
		Alignment:   master.Alignment,
		ArmorMult:   6, // level/6 for pierce/bash/slash
		ExoticMult:  8, // level/8 for exotic
	}

	return CreatePet(master, config)
}

// CreateAnimateFollower creates an animated follower from a body part (spell_animate)
// Body parts are objects with VNUMs 12-17 (brains, guts, hearts, etc.)
// NOTE: Unlike CreateZombiePet and CreateDemonPet, this does NOT set master.Pet.
// Animated body parts are followers only, not true pets. The C code only calls
// add_follower() without setting ch->pet.
func CreateAnimateFollower(master *types.Character, bodyPart *types.Object) *types.Character {
	if master == nil || bodyPart == nil {
		return nil
	}

	// Use the body part's descriptions
	name := bodyPart.Name
	shortDesc := bodyPart.ShortDesc
	longDesc := bodyPart.LongDesc

	// Modify long desc to show it floating
	if len(longDesc) > 12 {
		// Remove trailing newlines and "lying here" text
		longDesc = strings.TrimSuffix(longDesc, "\r\n")
		longDesc = strings.TrimSuffix(longDesc, "\n")
		// Find the last period or end
		if idx := strings.LastIndex(longDesc, " lying here"); idx > 0 {
			longDesc = longDesc[:idx]
		}
		if idx := strings.LastIndex(longDesc, " is here"); idx > 0 {
			longDesc = longDesc[:idx]
		}
		longDesc = longDesc + " floating here.\r\n"
	} else {
		longDesc = shortDesc + " floating here.\r\n"
	}

	// Description with master's mark
	description := fmt.Sprintf("%s\r\n{GIt's branded with the mark of %s.{x\r\n",
		bodyPart.LongDesc, master.Name)

	// Animate doesn't set a level based on anything - use master level / 2
	level := master.Level / 2
	if level < 1 {
		level = 1
	}

	// Create the follower character (NOT using CreatePet, which sets master.Pet)
	follower := types.NewNPC(MOBVnumAnimate, name, level)

	// Set descriptions
	follower.ShortDesc = shortDesc
	follower.LongDesc = longDesc
	follower.Desc = description

	// Set as charmed but NOT as a pet (key difference from CreatePet)
	follower.AffectedBy.Set(types.AffCharm)

	// Set communication flags (can't communicate)
	follower.Comm.Set(types.CommNoTell)
	follower.Comm.Set(types.CommNoShout)
	follower.Comm.Set(types.CommNoChannels)

	// Set stats
	follower.Level = level
	follower.MaxHit = level * 10
	follower.Hit = follower.MaxHit
	follower.Alignment = master.Alignment

	// Set armor
	acValue := level / 8
	follower.Armor[types.ACPierce] = acValue
	follower.Armor[types.ACBash] = acValue
	follower.Armor[types.ACSlash] = acValue
	follower.Armor[types.ACExotic] = level / 10

	// Set up follower relationship (but NOT master.Pet)
	follower.Master = master
	follower.Leader = master

	return follower
}

// CreateDemonPet creates a demon follower (spell_conjure)
func CreateDemonPet(master *types.Character) *types.Character {
	// Demon uses master's level directly
	level := master.Level
	if level < 1 {
		level = 1
	}

	// Standard demon descriptions
	name := "demon"
	shortDesc := "a demon"
	longDesc := "A demon from the lower planes stands here.\r\n"
	description := fmt.Sprintf("A demon from the lower planes.\r\n{GThe mark of %s is on its forehead.{x\r\n",
		master.Name)

	config := PetConfig{
		MobVnum:     MOBVnumDemon,
		Name:        name,
		ShortDesc:   shortDesc,
		LongDesc:    longDesc,
		Description: description,
		Level:       level,
		MaxHit:      level * 30, // Demons are tougher
		Alignment:   master.Alignment,
		ArmorMult:   2, // level/2 - better armor
		ExoticMult:  3, // level/3 - better exotic armor
	}

	return CreatePet(master, config)
}

// IsPet returns true if the character is a pet
func IsPet(ch *types.Character) bool {
	if ch == nil {
		return false
	}
	return ch.Act.Has(types.ActPet)
}

// GetPetMaster returns the master of a pet, or nil if not a pet
func GetPetMaster(ch *types.Character) *types.Character {
	if ch == nil || !IsPet(ch) {
		return nil
	}
	return ch.Master
}

// PetFollowMaster makes a pet follow its master to a new room
// This should be called after the master moves
func PetFollowMaster(pet *types.Character, newRoom *types.Room) {
	if pet == nil || newRoom == nil {
		return
	}

	// Remove from old room
	if pet.InRoom != nil {
		pet.InRoom.RemovePerson(pet)
	}

	// Add to new room
	pet.InRoom = newRoom
	newRoom.AddPerson(pet)
}
