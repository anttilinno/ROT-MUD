package magic

import (
	"rotmud/pkg/types"
)

// AddAffect applies an affect to a character with proper stat modification
func AddAffect(ch *types.Character, af *types.Affect) {
	// Check for existing affect of same type
	existing := ch.Affected.FindByType(af.Type)
	if existing != nil {
		// Stack duration (up to a limit)
		existing.Duration += af.Duration
		if existing.Duration > 100 {
			existing.Duration = 100
		}
		// Update level if stronger
		if af.Level > existing.Level {
			// Reverse old modifier, apply new
			reverseModifier(ch, existing)
			existing.Level = af.Level
			existing.Modifier = af.Modifier
			existing.Location = af.Location
			applyModifier(ch, existing)
		}
		return
	}

	// Apply the modifier
	applyModifier(ch, af)

	// Add to affect list (the character's method handles AffectedBy)
	ch.AddAffect(af)
}

// RemoveAffect removes an affect from a character
func RemoveAffect(ch *types.Character, af *types.Affect) {
	// Reverse the modifier
	reverseModifier(ch, af)

	// Remove from list (the character's method handles AffectedBy)
	ch.RemoveAffect(af)
}

// RemoveAffectByType removes all affects of a specific type
func RemoveAffectByType(ch *types.Character, affectType string) {
	for _, af := range ch.Affected.All() {
		if af.Type == affectType {
			RemoveAffect(ch, af)
		}
	}
}

// IsAffectedBy checks if a character has an affect of a specific type
func IsAffectedBy(ch *types.Character, affectType string) bool {
	return ch.Affected.HasType(affectType)
}

// GetAffect returns the first affect of a specific type, or nil
func GetAffect(ch *types.Character, affectType string) *types.Affect {
	return ch.Affected.FindByType(affectType)
}

// ApplyModifier applies an affect's stat modifier to a character
// This is the exported version for use with object affects
func ApplyModifier(ch *types.Character, af *types.Affect) {
	applyModifier(ch, af)
}

// ReverseModifier reverses an affect's stat modifier from a character
// This is the exported version for use with object affects
func ReverseModifier(ch *types.Character, af *types.Affect) {
	reverseModifier(ch, af)
}

// applyModifier applies an affect's stat modifier
func applyModifier(ch *types.Character, af *types.Affect) {
	mod := af.Modifier

	switch af.Location {
	case types.ApplyStr:
		ch.ModStats[types.StatStr] += mod
	case types.ApplyDex:
		ch.ModStats[types.StatDex] += mod
	case types.ApplyInt:
		ch.ModStats[types.StatInt] += mod
	case types.ApplyWis:
		ch.ModStats[types.StatWis] += mod
	case types.ApplyCon:
		ch.ModStats[types.StatCon] += mod
	case types.ApplyHit:
		ch.MaxHit += mod
	case types.ApplyMana:
		ch.MaxMana += mod
	case types.ApplyMove:
		ch.MaxMove += mod
	case types.ApplyAC:
		ch.Armor[types.ACPierce] += mod
		ch.Armor[types.ACBash] += mod
		ch.Armor[types.ACSlash] += mod
		ch.Armor[types.ACExotic] += mod
	case types.ApplyHitroll:
		ch.HitRoll += mod
	case types.ApplyDamroll:
		ch.DamRoll += mod
	case types.ApplySaves:
		// ch.Saving += mod (if we add this field)
	}
}

// reverseModifier reverses an affect's stat modifier
func reverseModifier(ch *types.Character, af *types.Affect) {
	mod := -af.Modifier

	switch af.Location {
	case types.ApplyStr:
		ch.ModStats[types.StatStr] += mod
	case types.ApplyDex:
		ch.ModStats[types.StatDex] += mod
	case types.ApplyInt:
		ch.ModStats[types.StatInt] += mod
	case types.ApplyWis:
		ch.ModStats[types.StatWis] += mod
	case types.ApplyCon:
		ch.ModStats[types.StatCon] += mod
	case types.ApplyHit:
		ch.MaxHit += mod
	case types.ApplyMana:
		ch.MaxMana += mod
	case types.ApplyMove:
		ch.MaxMove += mod
	case types.ApplyAC:
		ch.Armor[types.ACPierce] += mod
		ch.Armor[types.ACBash] += mod
		ch.Armor[types.ACSlash] += mod
		ch.Armor[types.ACExotic] += mod
	case types.ApplyHitroll:
		ch.HitRoll += mod
	case types.ApplyDamroll:
		ch.DamRoll += mod
	case types.ApplySaves:
		// ch.Saving += mod
	}
}

// AffectTick processes affect decay for a character
// Returns a list of wear-off messages for expired affects
func AffectTick(ch *types.Character, registry *SpellRegistry) []string {
	var messages []string

	expired := ch.Affected.TickAll()
	for _, af := range expired {
		// Reverse the modifier for expired affects
		reverseModifier(ch, af)

		// Recalculate AffectedBy and ShieldedBy flags
		ch.AffectedBy = ch.Affected.GetBitVector()
		ch.ShieldedBy = ch.Affected.GetShieldVector()

		// Get wear-off message from spell
		if registry != nil {
			if spell := registry.FindByName(af.Type); spell != nil && spell.WearOff != "" {
				messages = append(messages, spell.WearOff)
			}
		}
	}

	return messages
}

// NewAffect creates a new affect (convenience wrapper)
func NewAffect(affectType string, level, duration int, location types.ApplyType, modifier int) *types.Affect {
	return &types.Affect{
		Type:     affectType,
		Level:    level,
		Duration: duration,
		Location: location,
		Modifier: modifier,
	}
}

// NewAffectWithBit creates an affect that sets an affect bit
func NewAffectWithBit(affectType string, level, duration int, bit types.AffectFlags) *types.Affect {
	return &types.Affect{
		Type:      affectType,
		Level:     level,
		Duration:  duration,
		Location:  types.ApplyNone,
		BitVector: bit,
	}
}

// NewAffectWithShield creates an affect that sets a shield bit
func NewAffectWithShield(affectType string, level, duration int, shield types.ShieldFlags) *types.Affect {
	return &types.Affect{
		Type:         affectType,
		Level:        level,
		Duration:     duration,
		Location:     types.ApplyNone,
		ShieldVector: shield,
	}
}
