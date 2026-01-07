package magic

import (
	"rotmud/pkg/types"
)

// TargetType defines what a spell can target
type TargetType int

const (
	TargetIgnore      TargetType = iota // No target needed (area effects)
	TargetCharOffense                   // Offensive spell on character
	TargetCharDefense                   // Defensive spell on character
	TargetCharSelf                      // Self-only spell
	TargetObjInv                        // Object in inventory
	TargetObjCharDef                    // Object on defensive character
	TargetObjCharOff                    // Object on offensive character
)

// SpellFunc is the function signature for spell implementations
type SpellFunc func(caster *types.Character, level int, target interface{}) bool

// Spell defines a magical spell
type Spell struct {
	Name        string         // Spell name
	Slot        int            // Unique spell slot number
	Target      TargetType     // Target type
	MinPosition types.Position // Minimum position to cast
	ManaCost    int            // Base mana cost
	Beats       int            // Lag after casting (in pulses)
	NounDamage  string         // Damage noun for combat messages
	WearOff     string         // Message when affect wears off
	WearOffObj  string         // Message when object affect wears off
	MsgSelf     string         // Message to caster when cast on self
	MsgVictim   string         // Message to victim when cast on them
	MsgRoom     string         // Message to room (unused for now)
	Func        SpellFunc      // Spell function
	Levels      map[string]int // Level required by class (class name -> level)
}

// NewSpell creates a new spell with basic configuration
func NewSpell(name string, slot int, target TargetType, mana int, fn SpellFunc) *Spell {
	return &Spell{
		Name:        name,
		Slot:        slot,
		Target:      target,
		MinPosition: types.PosStanding,
		ManaCost:    mana,
		Beats:       12, // Default: 3 seconds lag (12 pulses at 250ms each)
		Func:        fn,
		Levels:      make(map[string]int),
	}
}

// SetLevels sets the class level requirements
func (s *Spell) SetLevels(levels map[string]int) *Spell {
	s.Levels = levels
	return s
}

// SetDamageNoun sets the damage noun for combat messages
func (s *Spell) SetDamageNoun(noun string) *Spell {
	s.NounDamage = noun
	return s
}

// SetWearOff sets the wear-off message
func (s *Spell) SetWearOff(msg string) *Spell {
	s.WearOff = msg
	return s
}

// SetMessages sets the effect messages for the spell
// selfMsg: shown to caster when casting on self (e.g., "You feel lighter.")
// victimMsg: shown to victim when cast on them by another (e.g., "You feel lighter.")
func (s *Spell) SetMessages(selfMsg, victimMsg string) *Spell {
	s.MsgSelf = selfMsg
	s.MsgVictim = victimMsg
	return s
}

// CanCast checks if a character can cast this spell
func (s *Spell) CanCast(ch *types.Character) bool {
	// Check position
	if ch.Position < s.MinPosition {
		return false
	}

	// Check mana
	if ch.Mana < s.ManaCost {
		return false
	}

	// NPCs can cast any spell at their level
	if ch.IsNPC() {
		return true
	}

	// Check class level requirement using class index
	reqLevel := s.GetClassLevel(ch.Class)
	if reqLevel == 0 {
		return false // Class can't learn this spell
	}
	return ch.Level >= reqLevel
}

// GetClassLevel returns the level requirement for a class index
func (s *Spell) GetClassLevel(classIndex int) int {
	// Map class index to name and look up
	className := types.ClassName(classIndex)
	if reqLevel, ok := s.Levels[className]; ok {
		return reqLevel
	}
	return 0 // Can't cast
}

// GetManaCost returns the mana cost adjusted for level
func (s *Spell) GetManaCost(ch *types.Character, level int) int {
	// Base mana cost, could be modified by skills, equipment, etc.
	return s.ManaCost
}

// SpellRegistry holds all registered spells
type SpellRegistry struct {
	byName map[string]*Spell
	bySlot map[int]*Spell
}

// NewSpellRegistry creates a new spell registry
func NewSpellRegistry() *SpellRegistry {
	return &SpellRegistry{
		byName: make(map[string]*Spell),
		bySlot: make(map[int]*Spell),
	}
}

// Register adds a spell to the registry
func (r *SpellRegistry) Register(spell *Spell) {
	r.byName[spell.Name] = spell
	r.bySlot[spell.Slot] = spell
}

// FindByName finds a spell by exact name match
func (r *SpellRegistry) FindByName(name string) *Spell {
	if spell, ok := r.byName[name]; ok {
		return spell
	}
	return nil
}

// FindByPrefix finds a spell by name prefix (for command parsing)
func (r *SpellRegistry) FindByPrefix(name string) *Spell {
	// Exact match first
	if spell, ok := r.byName[name]; ok {
		return spell
	}

	// Prefix match - the input must be a prefix of the spell name
	// If multiple spells match, prefer the one where input matches more of the name
	// e.g., "detect invis" should match "detect invis" not "detect evil"
	var bestMatch *Spell
	for spellName, spell := range r.byName {
		if len(name) <= len(spellName) && spellName[:len(name)] == name {
			// This spell matches - check if it's better than current best
			// Prefer spells where the input covers more of the spell name
			if bestMatch == nil {
				bestMatch = spell
			} else {
				// If current match would leave fewer unmatched characters, prefer it
				// e.g., "detect invis" matching "detect invis" (0 left) vs "detect evil" (would need "detect e" to match)
				currentUnmatched := len(spellName) - len(name)
				bestUnmatched := len(bestMatch.Name) - len(name)
				if currentUnmatched < bestUnmatched {
					bestMatch = spell
				}
			}
		}
	}
	return bestMatch
}

// FindBySlot finds a spell by slot number
func (r *SpellRegistry) FindBySlot(slot int) *Spell {
	return r.bySlot[slot]
}

// All returns all registered spells
func (r *SpellRegistry) All() []*Spell {
	spells := make([]*Spell, 0, len(r.byName))
	for _, spell := range r.byName {
		spells = append(spells, spell)
	}
	return spells
}
