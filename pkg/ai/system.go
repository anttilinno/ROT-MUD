package ai

import (
	"rotmud/pkg/combat"
	"rotmud/pkg/magic"
	"rotmud/pkg/types"
)

// AISystem manages NPC behavior and special functions
type AISystem struct {
	Registry *SpecialRegistry
	Magic    *magic.MagicSystem
	Output   func(ch *types.Character, msg string)

	// Callbacks for game actions
	ActToRoom   func(msg string, ch, victim *types.Character, output func(ch *types.Character, msg string))
	StartCombat func(ch, victim *types.Character)
	MoveChar    func(ch *types.Character, dir types.Direction)
}

// NewAISystem creates a new AI system
func NewAISystem() *AISystem {
	return &AISystem{
		Registry: NewSpecialRegistry(),
	}
}

// ProcessMobile runs AI for a single NPC
// Returns true if the mob took a special action
func (a *AISystem) ProcessMobile(ch *types.Character) bool {
	// Only process NPCs
	if !ch.IsNPC() {
		return false
	}

	// Check for special function
	if ch.Special != "" {
		specFunc := a.Registry.Find(ch.Special)
		if specFunc != nil {
			ctx := a.createContext()
			if specFunc(ch, ctx) {
				return true // Special function did something
			}
		}
	}

	// Default NPC behavior if no special or special didn't fire
	return a.defaultBehavior(ch)
}

// ProcessAllMobiles runs AI for all NPCs in the character list
func (a *AISystem) ProcessAllMobiles(characters []*types.Character) {
	for _, ch := range characters {
		if ch.IsNPC() {
			a.ProcessMobile(ch)
		}
	}
}

// createContext builds a SpecialContext with all the callbacks
func (a *AISystem) createContext() *SpecialContext {
	return &SpecialContext{
		Magic:       a.Magic,
		Output:      a.Output,
		ActToRoom:   a.ActToRoom,
		StartCombat: a.StartCombat,
		MoveChar:    a.MoveChar,
		CastSpell:   a.castSpellWrapper,
	}
}

// castSpellWrapper wraps the magic system's Cast method
func (a *AISystem) castSpellWrapper(ch *types.Character, spellName string, victim *types.Character) bool {
	if a.Magic == nil {
		return false
	}

	// Create a simple target finder that just returns the victim
	findTarget := func(caster *types.Character, name string, offensive bool) interface{} {
		return victim
	}

	return a.Magic.Cast(ch, spellName, "", findTarget)
}

// defaultBehavior implements standard NPC AI behaviors
func (a *AISystem) defaultBehavior(ch *types.Character) bool {
	if !combat.IsAwake(ch) {
		return false
	}

	// Scavenger behavior - pick up items
	if ch.Act.Has(types.ActScavenger) && ch.InRoom != nil && len(ch.InRoom.Objects) > 0 {
		if combat.NumberBits(6) == 0 {
			var bestObj *types.Object
			maxCost := 1

			for _, obj := range ch.InRoom.Objects {
				if obj.WearFlags.Has(types.WearTake) && obj.Cost > maxCost {
					bestObj = obj
					maxCost = obj.Cost
				}
			}

			if bestObj != nil {
				// Pick up the object
				for i, obj := range ch.InRoom.Objects {
					if obj == bestObj {
						ch.InRoom.Objects = append(ch.InRoom.Objects[:i], ch.InRoom.Objects[i+1:]...)
						ch.AddInventory(bestObj)

						if a.ActToRoom != nil {
							a.ActToRoom("$n picks up "+bestObj.ShortDesc+".", ch, nil, a.Output)
						}
						return true
					}
				}
			}
		}
	}

	// Aggressive behavior - attack players
	if ch.Act.Has(types.ActAggressive) && ch.Fighting == nil && ch.InRoom != nil {
		// Find a victim
		for _, victim := range ch.InRoom.People {
			if victim.IsNPC() {
				continue
			}
			if !combat.CanSee(ch, victim) {
				continue
			}
			if victim.Level >= types.LevelImmortal {
				continue
			}

			// Random chance to attack
			if combat.NumberBits(1) == 0 {
				if a.StartCombat != nil {
					a.StartCombat(ch, victim)
				}
				return true
			}
		}
	}

	// Wander behavior - move randomly
	if !ch.Act.Has(types.ActSentinel) && ch.Fighting == nil && ch.InRoom != nil {
		if combat.NumberBits(4) == 0 {
			// Pick a random exit
			exits := ch.InRoom.ExitDirections()
			if len(exits) > 0 {
				dir := exits[combat.NumberRange(0, len(exits)-1)]
				exit := ch.InRoom.GetExit(dir)

				// Check if we can move there
				if exit != nil && exit.ToRoom != nil && !exit.IsClosed() {
					// Check stay_area flag
					if ch.Act.Has(types.ActStayArea) {
						// Would need area info to check this properly
						// For now, just allow the move
					}

					if a.MoveChar != nil {
						a.MoveChar(ch, dir)
						return true
					}
				}
			}
		}
	}

	return false
}
