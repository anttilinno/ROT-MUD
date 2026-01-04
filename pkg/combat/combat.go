package combat

import (
	"rotmud/pkg/types"
)

// OutputFunc is a callback for sending messages
type OutputFunc func(ch *types.Character, msg string)

// RoomFinderFunc is a callback to find a room by vnum
type RoomFinderFunc func(vnum int) *types.Room

// CharMoverFunc is a callback to safely move a character to a room
type CharMoverFunc func(ch *types.Character, room *types.Room)

// SkillGetterFunc is a callback to get a character's skill level (0-100)
type SkillGetterFunc func(ch *types.Character, skillName string) int

// OnLevelUpFunc is a callback when a character levels up
type OnLevelUpFunc func(ch *types.Character, oldLevel, newLevel int)

// OnDamageFunc is a callback when damage is dealt (for metrics)
type OnDamageFunc func(damage int)

// OnKillFunc is a callback when a character is killed
type OnKillFunc func(killer, victim *types.Character)

// Constants for recall rooms (matching C source merc.h)
const (
	RoomVnumTemple  = 3001 // Default recall point
	RoomVnumTempleB = 3365 // Alternate recall point
)

// CombatSystem manages all combat operations
type CombatSystem struct {
	Output      OutputFunc
	RoomFinder  RoomFinderFunc  // For finding recall room on death
	CharMover   CharMoverFunc   // For moving characters to rooms
	SkillGetter SkillGetterFunc // For checking skill levels
	OnLevelUp   OnLevelUpFunc   // Called when a character levels up
	OnDamage    OnDamageFunc    // Called when damage is dealt (for metrics)
	OnKill      OnKillFunc      // Called when a character is killed (for quests)
}

// GetSkill returns a character's skill level, falling back to level-based if no getter
func (c *CombatSystem) GetSkill(ch *types.Character, skillName string) int {
	if c.SkillGetter != nil {
		return c.SkillGetter(ch, skillName)
	}
	// Fallback: level-based skill
	return 20 + ch.Level*2
}

// NewCombatSystem creates a new combat system
func NewCombatSystem() *CombatSystem {
	return &CombatSystem{}
}

// SetFighting starts combat between attacker and victim
func SetFighting(ch, victim *types.Character) {
	if ch.Fighting != nil {
		return // Already fighting
	}

	ch.Fighting = victim
	ch.Position = types.PosFighting
}

// StopFighting ends combat for a character
func StopFighting(ch *types.Character, allInRoom bool) {
	if allInRoom && ch.InRoom != nil {
		for _, person := range ch.InRoom.People {
			if person.Fighting == ch {
				person.Fighting = nil
				if person.Position == types.PosFighting {
					person.Position = types.PosStanding
				}
			}
		}
	}

	ch.Fighting = nil
	if ch.Position == types.PosFighting {
		ch.Position = types.PosStanding
	}
}

// IsAwake returns true if the character is awake
func IsAwake(ch *types.Character) bool {
	return ch.Position > types.PosSleeping
}

// IsSafe checks if combat is not allowed between two characters
func IsSafe(ch, victim *types.Character) bool {
	// Can't attack yourself
	if ch == victim {
		return true
	}

	// Safe rooms
	if ch.InRoom != nil && ch.InRoom.IsSafe() {
		return true
	}

	// Immortals are safe
	if victim.IsImmortal() {
		return true
	}

	return false
}

// CanSee returns true if ch can see victim
func CanSee(ch, victim *types.Character) bool {
	// Can always see yourself
	if ch == victim {
		return true
	}

	// Blind characters can't see
	if ch.IsAffected(types.AffBlind) {
		return false
	}

	// Check invisibility
	if victim.IsAffected(types.AffInvisible) && !ch.IsAffected(types.AffDetectInvis) {
		return false
	}

	// Check hidden
	if victim.IsAffected(types.AffHide) && !ch.IsAffected(types.AffDetectHidden) {
		return false
	}

	return true
}

// GetAC returns the character's armor class for a damage type
func GetAC(ch *types.Character, damType types.DamageType) int {
	var ac int
	switch damType {
	case types.DamPierce:
		ac = ch.Armor[types.ACPierce]
	case types.DamBash:
		ac = ch.Armor[types.ACBash]
	case types.DamSlash:
		ac = ch.Armor[types.ACSlash]
	default:
		ac = ch.Armor[types.ACExotic]
	}

	// Dexterity bonus
	ac -= (ch.GetStat(types.StatDex) - 15) * 2

	return ac
}

// strTable defines hit/dam bonuses per strength score (index 0-25)
// Based on ROM 2.4 str_app table: {tohit, todam, carry, wield}
var strTable = [][2]int{
	{-5, -4}, // 0
	{-5, -4}, // 1
	{-3, -2}, // 2
	{-3, -1}, // 3
	{-2, -1}, // 4
	{-2, -1}, // 5
	{-1, 0},  // 6
	{-1, 0},  // 7
	{0, 0},   // 8
	{0, 0},   // 9
	{0, 0},   // 10
	{0, 0},   // 11
	{0, 0},   // 12
	{0, 0},   // 13
	{0, 1},   // 14
	{1, 1},   // 15
	{1, 2},   // 16
	{2, 3},   // 17
	{2, 3},   // 18
	{3, 4},   // 19
	{3, 5},   // 20
	{4, 6},   // 21
	{4, 6},   // 22
	{5, 7},   // 23
	{5, 8},   // 24
	{6, 9},   // 25
}

// GetHitroll returns the character's hit bonus
func GetHitroll(ch *types.Character) int {
	hr := ch.HitRoll

	// Strength bonus for melee (from str_app table)
	str := ch.GetStat(types.StatStr)
	if str < 0 {
		str = 0
	}
	if str > 25 {
		str = 25
	}
	hr += strTable[str][0]

	return hr
}

// GetDamroll returns the character's damage bonus
func GetDamroll(ch *types.Character) int {
	dr := ch.DamRoll

	// Strength bonus (from str_app table)
	str := ch.GetStat(types.StatStr)
	if str < 0 {
		str = 0
	}
	if str > 25 {
		str = 25
	}
	dr += strTable[str][1]

	return dr
}

// GetThac0 returns the character's THAC0 (to hit armor class 0)
// Lower is better
func GetThac0(ch *types.Character) int {
	var thac0_00, thac0_32 int

	if ch.IsNPC() {
		// NPCs use level-based thac0
		thac0_00 = 20
		thac0_32 = -4 // Default to thief-like

		// Adjust based on mob type (using act flags)
		if ch.Act.Has(types.ActWarrior) {
			thac0_32 = -10
		} else if ch.Act.Has(types.ActCleric) {
			thac0_32 = 2
		} else if ch.Act.Has(types.ActMage) {
			thac0_32 = 6
		}
	} else {
		// Players use class-based thac0
		// For now, use generic values (warrior-like for simplicity)
		thac0_00 = 20
		thac0_32 = -10
	}

	thac0 := Interpolate(ch.Level, thac0_00, thac0_32)

	// Cap very low values
	if thac0 < 0 {
		thac0 = thac0 / 2
	}
	if thac0 < -5 {
		thac0 = -5 + (thac0+5)/2
	}

	return thac0
}

// GetWeaponDamType returns the damage type for a character's attack
func GetWeaponDamType(ch *types.Character) types.DamageType {
	wield := ch.GetEquipment(types.WearLocWield)
	if wield != nil && wield.ItemType == types.ItemTypeWeapon {
		return wield.DamageType()
	}
	// Default unarmed damage type
	if ch.DamType != types.DamNone {
		return ch.DamType
	}
	return types.DamBash // Default to bash (fists)
}

// UpdatePosition sets a character's position based on HP
func UpdatePosition(ch *types.Character) {
	if ch.Hit > 0 {
		if ch.Position <= types.PosStunned {
			ch.Position = types.PosStanding
		}
		return
	}

	if ch.IsNPC() || ch.Hit <= -11 {
		ch.Position = types.PosDead
		return
	}

	if ch.Hit <= -6 {
		ch.Position = types.PosMortal
	} else if ch.Hit <= -3 {
		ch.Position = types.PosIncap
	} else {
		ch.Position = types.PosStunned
	}
}

// CheckImmune returns the immunity status for a damage type
type ImmunityResult int

const (
	ImmNormal ImmunityResult = iota
	ImmImmune
	ImmResistant
	ImmVulnerable
)

// CheckImmune determines if the victim is immune/resistant/vulnerable to a damage type
func CheckImmune(victim *types.Character, damType types.DamageType) ImmunityResult {
	var immFlag, resFlag, vulnFlag types.ImmFlags

	switch damType {
	case types.DamFire:
		immFlag, resFlag, vulnFlag = types.ImmFire, types.ImmFire, types.ImmFire
	case types.DamCold:
		immFlag, resFlag, vulnFlag = types.ImmCold, types.ImmCold, types.ImmCold
	case types.DamLightning:
		immFlag, resFlag, vulnFlag = types.ImmLightning, types.ImmLightning, types.ImmLightning
	case types.DamAcid:
		immFlag, resFlag, vulnFlag = types.ImmAcid, types.ImmAcid, types.ImmAcid
	case types.DamPoison:
		immFlag, resFlag, vulnFlag = types.ImmPoison, types.ImmPoison, types.ImmPoison
	case types.DamBash:
		immFlag, resFlag, vulnFlag = types.ImmBash, types.ImmBash, types.ImmBash
	case types.DamPierce:
		immFlag, resFlag, vulnFlag = types.ImmPierce, types.ImmPierce, types.ImmPierce
	case types.DamSlash:
		immFlag, resFlag, vulnFlag = types.ImmSlash, types.ImmSlash, types.ImmSlash
	default:
		return ImmNormal
	}

	if victim.Imm.Has(immFlag) {
		return ImmImmune
	}
	if victim.Res.Has(resFlag) {
		return ImmResistant
	}
	if victim.Vuln.Has(vulnFlag) {
		return ImmVulnerable
	}
	return ImmNormal
}
