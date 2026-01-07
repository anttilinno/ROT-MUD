package types

import "time"

// Character represents a player or NPC
// Based on CHAR_DATA from merc.h:1585-1720
type Character struct {
	// Identity
	Name      string // Character name
	ShortDesc string // Short description (for NPCs)
	LongDesc  string // Long description (when seen in room)
	Desc      string // Full description (examine)

	// Core attributes
	Level     int  // Character level
	Class     int  // Class index
	Race      int  // Race index
	Sex       Sex  // Sex
	Alignment int  // Alignment (-1000 to 1000)
	Size      Size // Character size

	// Flags
	Act        ActFlags    // Action/behavior flags
	AffectedBy AffectFlags // Active affect flags (computed from Affected)
	ShieldedBy ShieldFlags // Active shield flags
	Comm       CommFlags   // Communication flags
	PlayerAct  PlayerFlags // Player-specific toggle flags (auto actions)
	Imm        ImmFlags    // Immunities
	Res        ImmFlags    // Resistances
	Vuln       ImmFlags    // Vulnerabilities

	// Vital stats
	Hit     int // Current HP
	MaxHit  int // Maximum HP
	Mana    int // Current mana
	MaxMana int // Maximum mana
	Move    int // Current movement
	MaxMove int // Maximum movement

	// Base stats (permanent + modifier)
	PermStats [MaxStats]int // Base stats (str, int, wis, dex, con)
	ModStats  [MaxStats]int // Stat modifiers from equipment/spells

	// Combat
	HitRoll  int        // Bonus to hit
	DamRoll  int        // Bonus to damage
	Armor    [4]int     // AC by type (pierce, bash, slash, exotic)
	Wimpy    int        // HP threshold for auto-flee
	Position Position   // Current position
	Fighting *Character // Current opponent

	// Damage output (for NPCs)
	Damage  [3]int     // dice number, dice size, bonus
	DamType DamageType // Default damage type

	// Location
	InRoom    *Room // Current room
	WasInRoom *Room // Previous room (for return after combat)

	// Relationships
	Master *Character // Who this character follows
	Leader *Character // Party leader
	Pet    *Character // Pet/charmed follower
	Reply  *Character // Last person who sent tell

	// Inventory and equipment
	Inventory []*Object           // Carried items
	Equipment [WearLocMax]*Object // Equipped items
	Carrying  *Object             // Object being held/wielded (legacy)

	// Affects
	Affected AffectList // Active affects

	// Money
	Gold     int // Gold coins
	Silver   int // Silver coins
	Platinum int // Platinum coins

	// Experience
	Exp   int // Experience points
	Trust int // Trust level (for immortals)

	// Training and Practice
	Train    int // Training sessions available
	Practice int // Practice sessions available

	// Timers
	Timer  int       // Idle timer
	Wait   int       // Command delay (lag)
	Daze   int       // Stun timer
	Logon  time.Time // Login time
	Played int       // Total time played (seconds)

	// NPC-specific
	MobVnum    int      // Mob template vnum (for NPCs)
	StartPos   Position // Starting position
	DefaultPos Position // Default position
	Special    string   // Special behavior function name (e.g. "spec_cast_mage")

	// Tracking (for track skill)
	TrackFrom [MaxTrack]int // Room vnums character came from
	TrackTo   [MaxTrack]int // Room vnums character went to

	// Player-specific
	Prompt     string      // Custom prompt
	Prefix     string      // Command prefix (immortal feature)
	Lines      int         // Lines per page for scrolling (0 = no paging)
	PCData     *PCData     // Player-specific data (nil for NPCs)
	Descriptor *Descriptor // Network connection (nil for NPCs)
	Deleted    bool        // Character has been deleted (don't save on disconnect)
}

// PCData contains player-specific data
// Based on PC_DATA from merc.h:1727-1762
type PCData struct {
	Password string // Hashed password
	Title    string // Player title
	Bamfin   string // Immortal enter message
	Bamfout  string // Immortal leave message
	WhoDesc  string // Custom who description

	// Tracking
	LastNote    time.Time // Last note read
	LastIdea    time.Time // Last idea read
	LastNews    time.Time // Last news read
	LastChanges time.Time // Last changes read
	LastPenalty time.Time // Last penalty read
	LastLevel   int       // Level at last login

	// Stats
	PermHit  int // Permanent max HP
	PermMana int // Permanent max mana
	PermMove int // Permanent max move
	TrueSex  Sex // Original sex (before magical changes)

	// Skills and groups
	Learned    map[string]int  // Skill name -> proficiency
	GroupKnown map[string]bool // Skill groups known

	// Social
	Clan    int   // Clan ID (0 = no clan)
	Invited int   // Clan invitation (0 = none)
	Deity   int   // Deity ID (0 = no deity)
	Tier    int   // Character tier/remort level
	Classes []int // Multi-class support (primary, secondary, etc.)

	// Conditions
	Condition [4]int // drunk, full, thirst, hunger

	// Command aliases
	Aliases map[string]string // alias -> substitution

	// Communication
	TellBuffer    []string    // Buffered tells for replay
	QuestProgress map[int]int // Quest ID -> progress count
	ForgetList    []string    // Players to ignore (max 10)

	// Other
	Recall          int // Recall room vnum
	SavedRoom       int // Last room vnum when player quit (0 = use recall)
	Points          int // Creation points spent
	OverspentPoints int // Creation points overspent (affects XP per level)
	Security        int // Builder security level
	BankGold        int // Gold stored in bank

	// Dupe tracking (alternate character names owned by same player)
	Dupes []string // List of alternate character names

	// OLC Editor state
	EditMode int         // What we're editing (0=none, 1=area, 2=room, 3=mob, 4=obj)
	EditVnum int         // Vnum of what we're editing
	EditData interface{} // The data being edited
}

// NewCharacter creates a new character
func NewCharacter(name string) *Character {
	return &Character{
		Name:      name,
		Level:     1,
		Position:  PosStanding,
		Alignment: 0,
		Size:      SizeMedium,
		Sex:       SexNeutral,
		Hit:       20,
		MaxHit:    20,
		Mana:      100,
		MaxMana:   100,
		Move:      100,
		MaxMove:   100,
		Inventory: make([]*Object, 0),
	}
}

// NewNPC creates a new NPC
func NewNPC(vnum int, name string, level int) *Character {
	ch := NewCharacter(name)
	ch.MobVnum = vnum
	ch.Level = level
	ch.Act.Set(ActNPC)
	return ch
}

// IsNPC returns true if this is an NPC
func (ch *Character) IsNPC() bool {
	return ch.Act.Has(ActNPC)
}

// IsPlayer returns true if this is a player
func (ch *Character) IsPlayer() bool {
	return !ch.IsNPC()
}

// IsImmortal returns true if this is an immortal
func (ch *Character) IsImmortal() bool {
	return ch.Level >= LevelImmortal
}

// GetStat returns the effective stat value (permanent + modifier)
func (ch *Character) GetStat(stat int) int {
	if stat < 0 || stat >= MaxStats {
		return 0
	}
	return ch.PermStats[stat] + ch.ModStats[stat]
}

// HitPercent returns current HP as a percentage
func (ch *Character) HitPercent() int {
	if ch.MaxHit == 0 {
		return 0
	}
	return (ch.Hit * 100) / ch.MaxHit
}

// ManaPercent returns current mana as a percentage
func (ch *Character) ManaPercent() int {
	if ch.MaxMana == 0 {
		return 100
	}
	return (ch.Mana * 100) / ch.MaxMana
}

// MovePercent returns current movement as a percentage
func (ch *Character) MovePercent() int {
	if ch.MaxMove == 0 {
		return 100
	}
	return (ch.Move * 100) / ch.MaxMove
}

// InCombat returns true if the character is fighting
func (ch *Character) InCombat() bool {
	return ch.Fighting != nil
}

// IsShielded returns true if the character has the specified shield flag
func (ch *Character) IsShielded(flag ShieldFlags) bool {
	return ch.ShieldedBy.Has(flag)
}

// CanAct returns true if the character can take actions
func (ch *Character) CanAct() bool {
	return ch.Position >= PosResting
}

// CanSee returns true if the character can see (not blind, not in dark)
func (ch *Character) CanSee() bool {
	if ch.IsAffected(AffBlind) {
		return false
	}
	// Add more checks for darkness, etc.
	return true
}

// IsAffected returns true if the character has the given affect flag
func (ch *Character) IsAffected(flag AffectFlags) bool {
	return ch.AffectedBy.Has(flag)
}

// AddAffect adds an affect to the character and updates AffectedBy/ShieldedBy flags
func (ch *Character) AddAffect(aff *Affect) {
	ch.Affected.Add(aff)
	ch.AffectedBy |= aff.BitVector
	ch.ShieldedBy |= aff.ShieldVector
}

// RemoveAffect removes an affect and updates AffectedBy/ShieldedBy flags
func (ch *Character) RemoveAffect(aff *Affect) {
	ch.Affected.Remove(aff)
	// Recalculate flags from remaining affects
	ch.AffectedBy = ch.Affected.GetBitVector()
	ch.ShieldedBy = ch.Affected.GetShieldVector()
}

// IsGood returns true if alignment is good (>= 350)
func (ch *Character) IsGood() bool {
	return ch.Alignment >= 350
}

// IsEvil returns true if alignment is evil (<= -350)
func (ch *Character) IsEvil() bool {
	return ch.Alignment <= -350
}

// IsNeutral returns true if alignment is neutral
func (ch *Character) IsNeutral() bool {
	return !ch.IsGood() && !ch.IsEvil()
}

// AlignmentString returns a description of the character's alignment
func (ch *Character) AlignmentString() string {
	switch {
	case ch.Alignment >= 900:
		return "angelic"
	case ch.Alignment >= 700:
		return "saintly"
	case ch.Alignment >= 350:
		return "good"
	case ch.Alignment >= 100:
		return "kind"
	case ch.Alignment > -100:
		return "neutral"
	case ch.Alignment > -350:
		return "mean"
	case ch.Alignment > -700:
		return "evil"
	case ch.Alignment > -900:
		return "demonic"
	default:
		return "satanic"
	}
}

// Equipment management

// GetEquipment returns the object in the given wear slot
func (ch *Character) GetEquipment(loc WearLocation) *Object {
	if loc < 0 || loc >= WearLocMax {
		return nil
	}
	return ch.Equipment[loc]
}

// Equip puts an object in an equipment slot
func (ch *Character) Equip(obj *Object, loc WearLocation) {
	if loc < 0 || loc >= WearLocMax {
		return
	}
	ch.Equipment[loc] = obj
	obj.WearLoc = loc
	obj.CarriedBy = ch
}

// Unequip removes an object from an equipment slot
func (ch *Character) Unequip(loc WearLocation) *Object {
	if loc < 0 || loc >= WearLocMax {
		return nil
	}
	obj := ch.Equipment[loc]
	if obj != nil {
		obj.WearLoc = WearLocNone
		ch.Equipment[loc] = nil
	}
	return obj
}

// Inventory management

// AddInventory adds an object to inventory
func (ch *Character) AddInventory(obj *Object) {
	obj.CarriedBy = ch
	obj.InRoom = nil
	obj.InObject = nil
	ch.Inventory = append(ch.Inventory, obj)
}

// RemoveInventory removes an object from inventory
func (ch *Character) RemoveInventory(obj *Object) {
	for i, item := range ch.Inventory {
		if item == obj {
			ch.Inventory = append(ch.Inventory[:i], ch.Inventory[i+1:]...)
			obj.CarriedBy = nil
			return
		}
	}
}

// CarryWeight returns the total weight being carried
func (ch *Character) CarryWeight() int {
	total := 0
	for _, obj := range ch.Inventory {
		total += obj.TotalWeight()
	}
	for _, obj := range ch.Equipment {
		if obj != nil {
			total += obj.TotalWeight()
		}
	}
	return total
}

// CarryCount returns the number of items being carried
func (ch *Character) CarryCount() int {
	count := len(ch.Inventory)
	for _, obj := range ch.Equipment {
		if obj != nil {
			count++
		}
	}
	return count
}

// RecordTrack records a movement from one room to another for tracking purposes
func (ch *Character) RecordTrack(fromVnum, toVnum int) {
	// Shift all tracks down
	for i := MaxTrack - 1; i > 0; i-- {
		ch.TrackFrom[i] = ch.TrackFrom[i-1]
		ch.TrackTo[i] = ch.TrackTo[i-1]
	}
	// Record new track at position 0
	ch.TrackFrom[0] = fromVnum
	ch.TrackTo[0] = toVnum
}

// Player penalty flag helpers

// HasPenalty returns true if the player has the specified penalty flag
func (ch *Character) HasPenalty(flag PlayerFlags) bool {
	return ch.PlayerAct.Has(flag)
}

// AddPenalty adds a penalty flag to the player
func (ch *Character) AddPenalty(flag PlayerFlags) {
	ch.PlayerAct.Set(flag)
}

// RemovePenalty removes a penalty flag from the player
func (ch *Character) RemovePenalty(flag PlayerFlags) {
	ch.PlayerAct.Remove(flag)
}

// IsKiller returns true if the player is marked as a killer
func (ch *Character) IsKiller() bool {
	return ch.HasPenalty(PlrKiller)
}

// IsThief returns true if the player is marked as a thief
func (ch *Character) IsThief() bool {
	return ch.HasPenalty(PlrThief)
}

// IsFrozen returns true if the player is frozen
func (ch *Character) IsFrozen() bool {
	return ch.HasPenalty(PlrFrozen)
}
