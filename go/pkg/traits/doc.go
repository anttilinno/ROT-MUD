// Package traits implements the data-driven trait type system for the ROT MUD.
//
// Traits are typed, parameterized annotations (vulnerabilities, resistances,
// immunities, stat modifiers, capability flags, behavior hooks) that compose
// additively into a resolved trait set queried by combat, magic, and skills.
// Content lives in data files; the engine queries the composed trait set
// instead of doing identity checks against hardcoded race/class constants.
//
// # Trait Kinds
//
// Each trait is one of six closed kinds tagged by [TraitKind]:
//
//   - [Vulnerability]: increases damage taken on a [rotmud/pkg/types.DamageType] axis
//   - [Resistance]: decreases damage taken on a damage axis
//   - [Immunity]: nullifies damage on a damage axis
//   - [StatModifier]: a signed delta applied to a [rotmud/pkg/types.Stat] index
//   - [Capability]: a named boolean flag interned to a stable bit
//   - [BehaviorHook]: a Lua script reference bound to a [HookEvent]
//
// # Behavior Hooks
//
// A [BehaviorHook] names one of five [HookEvent] points at which a script runs.
// This package stores the script reference only — Lua VM execution is a later
// phase and no script is loaded or evaluated here.
//
// # Capabilities
//
// Capability strings intern to stable bits in a deterministic package-level
// registry, packed into a fixed 256-bit [CapBits] value so capability presence
// tests are O(1) and zero-allocation.
package traits
