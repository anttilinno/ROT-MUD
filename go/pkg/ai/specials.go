// Package ai implements NPC special behaviors
// Ported from special.c in the original C codebase
package ai

import (
	"rotmud/pkg/combat"
	"rotmud/pkg/magic"
	"rotmud/pkg/types"
)

// SpecialFunc is the signature for all special behavior functions
// Returns true if the mob did something (should skip normal AI)
type SpecialFunc func(ch *types.Character, ctx *SpecialContext) bool

// SpecialContext provides dependencies for special functions
type SpecialContext struct {
	Magic            *magic.MagicSystem
	Output           func(ch *types.Character, msg string)
	ActToRoom        func(msg string, ch, victim *types.Character, output func(ch *types.Character, msg string))
	StartCombat      func(ch, victim *types.Character)
	CastSpell        func(ch *types.Character, spellName string, victim *types.Character) bool
	FindPlayerInRoom func(ch *types.Character) *types.Character
	MoveChar         func(ch *types.Character, dir types.Direction)
}

// SpecialEntry maps a name to a special function
type SpecialEntry struct {
	Name string
	Func SpecialFunc
}

// SpecialRegistry holds all registered special functions
type SpecialRegistry struct {
	specials map[string]SpecialFunc
}

// NewSpecialRegistry creates a new registry with default specials
func NewSpecialRegistry() *SpecialRegistry {
	r := &SpecialRegistry{
		specials: make(map[string]SpecialFunc),
	}
	r.registerDefaults()
	return r
}

// Register adds a special function
func (r *SpecialRegistry) Register(name string, fn SpecialFunc) {
	r.specials[name] = fn
}

// Find looks up a special function by name
func (r *SpecialRegistry) Find(name string) SpecialFunc {
	return r.specials[name]
}

// registerDefaults adds all the standard special functions
func (r *SpecialRegistry) registerDefaults() {
	// Dragon breath attacks
	r.Register("spec_breath_any", specBreathAny)
	r.Register("spec_breath_acid", specBreathAcid)
	r.Register("spec_breath_fire", specBreathFire)
	r.Register("spec_breath_frost", specBreathFrost)
	r.Register("spec_breath_gas", specBreathGas)
	r.Register("spec_breath_lightning", specBreathLightning)

	// Casting mobs
	r.Register("spec_cast_adept", specCastAdept)
	r.Register("spec_cast_cleric", specCastCleric)
	r.Register("spec_cast_mage", specCastMage)
	r.Register("spec_cast_undead", specCastUndead)

	// Guard and law enforcement
	r.Register("spec_guard", specGuard)
	r.Register("spec_executioner", specExecutioner)
	r.Register("spec_patrolman", specPatrolman)

	// Thief and nasty mobs
	r.Register("spec_thief", specThief)
	r.Register("spec_nasty", specNasty)
	r.Register("spec_poison", specPoison)

	// Utility mobs
	r.Register("spec_janitor", specJanitor)
	r.Register("spec_fido", specFido)

	// Fun mobs
	r.Register("spec_mayor", specMayor)
}

// Helper to check if mob can act
func canAct(ch *types.Character) bool {
	if !combat.IsAwake(ch) {
		return false
	}
	if ch.IsAffected(types.AffCalm) {
		return false
	}
	if ch.IsAffected(types.AffCharm) {
		return false
	}
	if ch.InRoom == nil {
		return false
	}
	return true
}

// Dragon breath helper - casts a breath spell
func dragonBreath(ch *types.Character, spellName string, ctx *SpecialContext) bool {
	if ch.Position != types.PosFighting {
		return false
	}

	// Find a random target fighting us
	var victim *types.Character
	for _, vch := range ch.InRoom.People {
		if vch.Fighting == ch && combat.NumberBits(3) == 0 {
			victim = vch
			break
		}
	}

	if victim == nil {
		return false
	}

	if ch.Daze > 0 {
		return false
	}

	// Cast the breath weapon spell
	if ctx.CastSpell != nil {
		return ctx.CastSpell(ch, spellName, victim)
	}

	return false
}

// === Dragon Breath Specials ===

func specBreathAny(ch *types.Character, ctx *SpecialContext) bool {
	if ch.Position != types.PosFighting {
		return false
	}

	switch combat.NumberBits(3) {
	case 0:
		return specBreathFire(ch, ctx)
	case 1, 2:
		return specBreathLightning(ch, ctx)
	case 3:
		return specBreathGas(ch, ctx)
	case 4:
		return specBreathAcid(ch, ctx)
	default:
		return specBreathFrost(ch, ctx)
	}
}

func specBreathAcid(ch *types.Character, ctx *SpecialContext) bool {
	return dragonBreath(ch, "acid breath", ctx)
}

func specBreathFire(ch *types.Character, ctx *SpecialContext) bool {
	return dragonBreath(ch, "fire breath", ctx)
}

func specBreathFrost(ch *types.Character, ctx *SpecialContext) bool {
	return dragonBreath(ch, "frost breath", ctx)
}

func specBreathGas(ch *types.Character, ctx *SpecialContext) bool {
	return dragonBreath(ch, "gas breath", ctx)
}

func specBreathLightning(ch *types.Character, ctx *SpecialContext) bool {
	return dragonBreath(ch, "lightning breath", ctx)
}

// === Casting Specials ===

// specCastAdept - helpful healer that buffs low level players
func specCastAdept(ch *types.Character, ctx *SpecialContext) bool {
	if !canAct(ch) {
		return false
	}

	// Find a low-level player in the room
	var victim *types.Character
	for _, vch := range ch.InRoom.People {
		if vch != ch && !vch.IsNPC() && vch.Level < 31 &&
			combat.CanSee(ch, vch) && combat.NumberBits(1) == 0 {
			victim = vch
			break
		}
	}

	if victim == nil {
		return false
	}

	var spell string
	var word string

	switch combat.NumberBits(4) {
	case 0:
		word = "abrazak"
		spell = "armor"
	case 1:
		word = "fido"
		spell = "bless"
	case 2:
		word = "judicandus noselacri"
		spell = "cure blindness"
	case 3:
		word = "judicandus dies"
		spell = "cure light"
	case 4:
		word = "judicandus sausabru"
		spell = "cure poison"
	case 5:
		word = "candusima"
		spell = "refresh"
	case 6:
		word = "judicandus eugzagz"
		spell = "cure disease"
	case 7:
		word = "judicandus eqtuyp"
		spell = "sanctuary"
	case 8:
		word = "qwerty"
		spell = "shield"
	case 9:
		word = "saspell"
		spell = "heal"
	default:
		return false
	}

	// Say the magic word
	if ctx.ActToRoom != nil {
		ctx.ActToRoom("$n utters the word '"+word+"'.", ch, nil, ctx.Output)
	}

	// Cast the spell
	if ctx.CastSpell != nil {
		return ctx.CastSpell(ch, spell, victim)
	}

	return true
}

// specCastCleric - offensive cleric mob
func specCastCleric(ch *types.Character, ctx *SpecialContext) bool {
	if ch.Position != types.PosFighting {
		return false
	}

	// Find a random target fighting us
	var victim *types.Character
	for _, vch := range ch.InRoom.People {
		if vch.Fighting == ch && combat.NumberBits(2) == 0 {
			victim = vch
			break
		}
	}

	if victim == nil || ch.Daze > 0 {
		return false
	}

	// Pick a spell based on level
	var spell string
	for {
		var minLevel int
		switch combat.NumberBits(4) {
		case 0:
			minLevel = 0
			spell = "blindness"
		case 1:
			minLevel = 3
			spell = "cause serious"
		case 2:
			minLevel = 7
			spell = "earthquake"
		case 3:
			minLevel = 9
			spell = "cause critical"
		case 4:
			minLevel = 10
			spell = "dispel evil"
		case 5, 6:
			minLevel = 12
			spell = "curse"
		case 7:
			minLevel = 13
			spell = "flamestrike"
		case 8, 9, 10:
			minLevel = 15
			spell = "harm"
		case 11:
			minLevel = 15
			spell = "plague"
		default:
			minLevel = 16
			spell = "dispel magic"
		}

		if ch.Level >= minLevel {
			break
		}
	}

	if ctx.CastSpell != nil {
		return ctx.CastSpell(ch, spell, victim)
	}

	return false
}

// specCastMage - offensive mage mob
func specCastMage(ch *types.Character, ctx *SpecialContext) bool {
	if ch.Position != types.PosFighting {
		return false
	}

	// Find a random target fighting us
	var victim *types.Character
	for _, vch := range ch.InRoom.People {
		if vch.Fighting == ch && combat.NumberBits(2) == 0 {
			victim = vch
			break
		}
	}

	if victim == nil || ch.Daze > 0 {
		return false
	}

	// Pick a spell based on level
	var spell string
	for {
		var minLevel int
		switch combat.NumberBits(4) {
		case 0:
			minLevel = 0
			spell = "blindness"
		case 1:
			minLevel = 3
			spell = "chill touch"
		case 2:
			minLevel = 7
			spell = "weaken"
		case 3:
			minLevel = 8
			spell = "teleport"
		case 4:
			minLevel = 11
			spell = "colour spray"
		case 5:
			minLevel = 12
			spell = "change sex"
		case 6:
			minLevel = 13
			spell = "energy drain"
		case 7, 8, 9:
			minLevel = 15
			spell = "fireball"
		case 10:
			minLevel = 20
			spell = "plague"
		default:
			minLevel = 20
			spell = "acid blast"
		}

		if ch.Level >= minLevel {
			break
		}
	}

	if ctx.CastSpell != nil {
		return ctx.CastSpell(ch, spell, victim)
	}

	return false
}

// specCastUndead - undead caster mob
func specCastUndead(ch *types.Character, ctx *SpecialContext) bool {
	if ch.Position != types.PosFighting {
		return false
	}

	// Find a random target fighting us
	var victim *types.Character
	for _, vch := range ch.InRoom.People {
		if vch.Fighting == ch && combat.NumberBits(2) == 0 {
			victim = vch
			break
		}
	}

	if victim == nil || ch.Daze > 0 {
		return false
	}

	// Pick a spell based on level
	var spell string
	for {
		var minLevel int
		switch combat.NumberBits(4) {
		case 0:
			minLevel = 0
			spell = "curse"
		case 1:
			minLevel = 3
			spell = "weaken"
		case 2:
			minLevel = 6
			spell = "chill touch"
		case 3:
			minLevel = 9
			spell = "blindness"
		case 4:
			minLevel = 12
			spell = "poison"
		case 5:
			minLevel = 15
			spell = "energy drain"
		case 6, 7:
			minLevel = 18
			spell = "harm"
		default:
			minLevel = 20
			spell = "plague"
		}

		if ch.Level >= minLevel {
			break
		}
	}

	if ctx.CastSpell != nil {
		return ctx.CastSpell(ch, spell, victim)
	}

	return false
}

// === Guard and Law Enforcement ===

// specGuard - attacks evil players and protects the innocent
func specGuard(ch *types.Character, ctx *SpecialContext) bool {
	if !canAct(ch) || ch.Fighting != nil {
		return false
	}

	var target *types.Character
	var maxEvil = 300

	for _, victim := range ch.InRoom.People {
		// Look for "TWIT" flagged players (troublemakers)
		if !victim.IsNPC() && victim.Act.Has(types.ActPet) {
			// Using ActPet as a standin for PLR_TWIT flag
			if combat.CanSee(ch, victim) {
				target = victim
				break
			}
		}

		// Or attack very evil fighters
		if victim.Fighting != nil && victim.Fighting != ch && victim.Alignment < maxEvil {
			maxEvil = victim.Alignment
			target = victim
		}
	}

	if target == nil {
		return false
	}

	// Yell and attack
	if ctx.ActToRoom != nil {
		ctx.ActToRoom("$n screams 'PROTECT THE INNOCENT!! BANZAI!!'", ch, nil, ctx.Output)
	}

	if ctx.StartCombat != nil {
		ctx.StartCombat(ch, target)
	}

	return true
}

// specExecutioner - hunts down troublemakers
func specExecutioner(ch *types.Character, ctx *SpecialContext) bool {
	if !canAct(ch) || ch.Fighting != nil {
		return false
	}

	// Find troublemakers
	var victim *types.Character
	for _, vch := range ch.InRoom.People {
		if !vch.IsNPC() && vch.Act.Has(types.ActPet) && combat.CanSee(ch, vch) {
			victim = vch
			break
		}
	}

	if victim == nil {
		return false
	}

	if ctx.ActToRoom != nil {
		ctx.ActToRoom("$n screams 'PROTECT THE INNOCENT! MORE BLOOD!!!'", ch, nil, ctx.Output)
	}

	if ctx.StartCombat != nil {
		ctx.StartCombat(ch, victim)
	}

	return true
}

// specPatrolman - breaks up fights
func specPatrolman(ch *types.Character, ctx *SpecialContext) bool {
	if !canAct(ch) || ch.Fighting != nil {
		return false
	}

	// Look for a fight in the room
	var victim *types.Character
	for _, vch := range ch.InRoom.People {
		if vch == ch {
			continue
		}
		if vch.Fighting != nil {
			// Break it up by attacking the stronger fighter
			if vch.Level > vch.Fighting.Level {
				victim = vch
			} else {
				victim = vch.Fighting
			}
			break
		}
	}

	if victim == nil || (victim.IsNPC() && victim.MobVnum == ch.MobVnum) {
		return false
	}

	// Yell at them
	messages := []string{
		"All roit! All roit! break it up!",
		"Society's to blame, but what's a bloke to do?",
		"I'll teach you to disrupt the peace!",
		"Hey! Who started this fight?",
		"Hold it! POLICE!",
		"Settle down, you hooligans!",
	}

	msg := messages[combat.NumberRange(0, len(messages)-1)]
	if ctx.ActToRoom != nil {
		ctx.ActToRoom("$n yells '"+msg+"'", ch, nil, ctx.Output)
	}

	if ctx.StartCombat != nil {
		ctx.StartCombat(ch, victim)
	}

	return true
}

// === Thief and Nasty Mobs ===

// specThief - steals from players
func specThief(ch *types.Character, ctx *SpecialContext) bool {
	if ch.Position != types.PosStanding {
		return false
	}

	// Find a victim
	var victim *types.Character
	for _, vch := range ch.InRoom.People {
		if !vch.IsNPC() && vch.Level < types.LevelImmortal &&
			combat.NumberBits(5) == 0 && combat.CanSee(ch, vch) {
			victim = vch
			break
		}
	}

	if victim == nil || victim == ch.Fighting {
		return false
	}

	// Try to steal gold
	if combat.NumberPercent() < 50 {
		// Steal some gold
		gold := victim.Gold * combat.NumberRange(1, 20) / 100
		if gold > 0 {
			ch.Gold += gold / 2
			victim.Gold -= gold

			if ctx.Output != nil {
				ctx.Output(victim, "You feel lighter.\r\n")
			}
		}
		return true
	}

	// TODO: Try to steal an object

	return false
}

// specNasty - backstabs and steals
func specNasty(ch *types.Character, ctx *SpecialContext) bool {
	if !combat.IsAwake(ch) {
		return false
	}

	if ch.Position != types.PosFighting {
		// Not fighting - look for a target
		for _, victim := range ch.InRoom.People {
			if !victim.IsNPC() && victim.Level > ch.Level && victim.Level < ch.Level+10 {
				// Try to backstab
				if ctx.StartCombat != nil {
					ctx.StartCombat(ch, victim)
				}
				return true
			}
		}
		return false
	}

	// Fighting - try to steal and flee
	victim := ch.Fighting
	if victim == nil {
		return false
	}

	switch combat.NumberBits(2) {
	case 0:
		// Steal gold
		gold := victim.Gold / 10
		victim.Gold -= gold
		ch.Gold += gold

		if ctx.ActToRoom != nil {
			ctx.ActToRoom("$n rips apart your coin purse, spilling your gold!", ch, victim, ctx.Output)
		}
		return true

	case 1:
		// Flee
		combat.StopFighting(ch, true)
		// TODO: Actually move to a random exit
		return true
	}

	return false
}

// specPoison - bites and poisons during combat
func specPoison(ch *types.Character, ctx *SpecialContext) bool {
	if ch.Position != types.PosFighting {
		return false
	}

	victim := ch.Fighting
	if victim == nil {
		return false
	}

	// 2*level percent chance
	if combat.NumberPercent() > 2*ch.Level {
		return false
	}

	if ch.Daze > 0 {
		return false
	}

	if ctx.ActToRoom != nil {
		ctx.ActToRoom("$n bites $N!", ch, victim, ctx.Output)
	}

	if ctx.CastSpell != nil {
		ctx.CastSpell(ch, "poison", victim)
	}

	return true
}

// === Utility Mobs ===

// specJanitor - picks up trash
func specJanitor(ch *types.Character, ctx *SpecialContext) bool {
	if !combat.IsAwake(ch) || ch.InRoom == nil {
		return false
	}

	for _, obj := range ch.InRoom.Objects {
		if !obj.WearFlags.Has(types.WearTake) {
			continue
		}

		// Pick up drink containers, trash, or cheap items
		if obj.ItemType == types.ItemTypeDrinkCon ||
			obj.ItemType == types.ItemTypeTrash ||
			obj.Cost < 10 {

			if ctx.ActToRoom != nil {
				ctx.ActToRoom("$n picks up some trash.", ch, nil, ctx.Output)
			}

			// Move obj from room to inventory
			// Remove from room
			for i, o := range ch.InRoom.Objects {
				if o == obj {
					ch.InRoom.Objects = append(ch.InRoom.Objects[:i], ch.InRoom.Objects[i+1:]...)
					break
				}
			}
			// Add to inventory
			ch.AddInventory(obj)

			return true
		}
	}

	return false
}

// specFido - eats corpses
func specFido(ch *types.Character, ctx *SpecialContext) bool {
	if !combat.IsAwake(ch) || ch.InRoom == nil {
		return false
	}

	for i, obj := range ch.InRoom.Objects {
		if obj.ItemType != types.ItemTypeCorpseNPC {
			continue
		}

		if ctx.ActToRoom != nil {
			ctx.ActToRoom("$n savagely devours a corpse.", ch, nil, ctx.Output)
		}

		// Drop contents to room
		for _, contained := range obj.Contents {
			obj.RemoveContent(contained)
			contained.InRoom = ch.InRoom
			ch.InRoom.Objects = append(ch.InRoom.Objects, contained)
		}

		// Remove the corpse
		ch.InRoom.Objects = append(ch.InRoom.Objects[:i], ch.InRoom.Objects[i+1:]...)

		return true
	}

	return false
}

// === Special Fun Mobs ===

// MayorState holds state for the mayor's patrol
type MayorState struct {
	Path   string
	Pos    int
	Moving bool
}

// Global mayor state (one mayor in the world)
var mayorState = &MayorState{
	Path:   "W3a3003b33000c111d0d111Oe333333Oe22c222112212111a1S.",
	Pos:    0,
	Moving: false,
}

// specMayor - follows a patrol path, opens/closes gates
func specMayor(ch *types.Character, ctx *SpecialContext) bool {
	// If fighting, cast spells
	if ch.Fighting != nil {
		return specCastMage(ch, ctx)
	}

	// Check if it's time to start moving
	// In a real implementation, this would check game time
	// For now, just patrol randomly
	if !mayorState.Moving {
		if combat.NumberBits(3) == 0 {
			mayorState.Moving = true
			mayorState.Pos = 0
		} else {
			return false
		}
	}

	if ch.Position < types.PosSleeping {
		return false
	}

	// Follow the path
	if mayorState.Pos >= len(mayorState.Path) {
		mayorState.Moving = false
		return false
	}

	action := mayorState.Path[mayorState.Pos]
	mayorState.Pos++

	switch action {
	case '0', '1', '2', '3':
		// Move in direction
		dir := types.Direction(action - '0')
		if ctx.MoveChar != nil {
			ctx.MoveChar(ch, dir)
		}
	case 'W':
		ch.Position = types.PosStanding
		if ctx.ActToRoom != nil {
			ctx.ActToRoom("$n awakens and groans loudly.", ch, nil, ctx.Output)
		}
	case 'S':
		ch.Position = types.PosSleeping
		if ctx.ActToRoom != nil {
			ctx.ActToRoom("$n lies down and falls asleep.", ch, nil, ctx.Output)
		}
	case 'a':
		if ctx.ActToRoom != nil {
			ctx.ActToRoom("$n says 'Hello Honey!'", ch, nil, ctx.Output)
		}
	case 'b':
		if ctx.ActToRoom != nil {
			ctx.ActToRoom("$n says 'What a view! I must do something about that dump!'", ch, nil, ctx.Output)
		}
	case 'c':
		if ctx.ActToRoom != nil {
			ctx.ActToRoom("$n says 'Vandals! Youngsters have no respect for anything!'", ch, nil, ctx.Output)
		}
	case 'd':
		if ctx.ActToRoom != nil {
			ctx.ActToRoom("$n says 'Good day, citizens!'", ch, nil, ctx.Output)
		}
	case 'e':
		if ctx.ActToRoom != nil {
			ctx.ActToRoom("$n says 'I hereby declare the city of Midgaard open!'", ch, nil, ctx.Output)
		}
	case 'E':
		if ctx.ActToRoom != nil {
			ctx.ActToRoom("$n says 'I hereby declare the city of Midgaard closed!'", ch, nil, ctx.Output)
		}
	case 'O':
		// Open gate - would need door handling
		if ctx.ActToRoom != nil {
			ctx.ActToRoom("$n opens the gate.", ch, nil, ctx.Output)
		}
	case 'C':
		// Close gate
		if ctx.ActToRoom != nil {
			ctx.ActToRoom("$n closes the gate.", ch, nil, ctx.Output)
		}
	case '.':
		mayorState.Moving = false
	}

	return false
}
