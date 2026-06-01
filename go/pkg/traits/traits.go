package traits

import "rotmud/pkg/types"

// TraitKind tags the six closed kinds of trait (TRAIT-01 / D-04).
type TraitKind int

const (
	KindVulnerability TraitKind = iota // Increases damage on a damage axis
	KindResistance                     // Decreases damage on a damage axis
	KindImmunity                       // Nullifies damage on a damage axis
	KindModifier                       // Signed delta on a stat index
	KindCapability                     // Named boolean capability flag
	KindHook                           // Lua behavior-hook reference
)

// String returns the trait-kind name, or "unknown" if out of range.
func (k TraitKind) String() string {
	names := []string{
		"vulnerability", "resistance", "immunity",
		"modifier", "capability", "hook",
	}
	if k >= 0 && int(k) < len(names) {
		return names[k]
	}
	return "unknown"
}

// HookEvent names the five points at which a [BehaviorHook] script may run.
// Lua execution is a later phase; this enum only tags the binding point.
type HookEvent int

const (
	OnBeforeDamage HookEvent = iota // Before damage is applied to a victim
	OnAfterDamage                   // After damage has been applied
	OnDeath                         // When a character dies
	OnSpellCast                     // When a spell is cast
	OnLevelUp                       // When a character gains a level
)

// String returns the hook-event name, or "unknown" if out of range.
func (e HookEvent) String() string {
	names := []string{
		"on_before_damage", "on_after_damage", "on_death",
		"on_spell_cast", "on_level_up",
	}
	if e >= 0 && int(e) < len(names) {
		return names[e]
	}
	return "unknown"
}

// Vulnerability increases damage taken on a damage axis.
type Vulnerability struct {
	DamageType types.DamageType // RIS axis (D-02: reuse existing enum)
	Magnitude  int              // numeric, summed across sources (D-01)
}

// Resistance decreases damage taken on a damage axis.
type Resistance struct {
	DamageType types.DamageType // RIS axis (D-02: reuse existing enum)
	Magnitude  int              // numeric, summed across sources (D-01)
}

// Immunity nullifies damage taken on a damage axis.
type Immunity struct {
	DamageType types.DamageType // RIS axis (D-02: reuse existing enum)
	Magnitude  int              // numeric, summed across sources (D-01)
}

// StatModifier applies a signed delta to a stat index.
type StatModifier struct {
	Stat  types.Stat // stat index (D-02: reuse types.Stat alias)
	Delta int        // signed delta, summed across sources in Resolve (Plan 02)
}

// Capability is a named boolean flag interned to a stable bit at Resolve (D-05).
type Capability struct {
	Key string // capability name, interned to a CapBits bit
}

// BehaviorHook binds a Lua script reference to a hook event.
type BehaviorHook struct {
	Event  HookEvent // when the script runs
	Script string    // script reference only; Lua execution is a later phase
}
