package traits

import (
	"testing"

	"rotmud/pkg/types"
)

func TestCompose(t *testing.T) {
	t.Run("preserves left-to-right source order across every kind slice", func(t *testing.T) {
		race := TraitSet{
			Vulnerabilities: []Vulnerability{{DamageType: types.DamFire, Magnitude: 10}},
			Resistances:     []Resistance{{DamageType: types.DamFire, Magnitude: 1}},
			Immunities:      []Immunity{{DamageType: types.DamFire, Magnitude: 100}},
			Modifiers:       []StatModifier{{Stat: types.StatStr, Delta: 1}},
			Capabilities:    []Capability{{Key: "race-cap"}},
			Hooks:           []BehaviorHook{{Event: OnDeath, Script: "race.lua"}},
		}
		class := TraitSet{
			Vulnerabilities: []Vulnerability{{DamageType: types.DamFire, Magnitude: 20}},
			Resistances:     []Resistance{{DamageType: types.DamFire, Magnitude: 2}},
			Immunities:      []Immunity{{DamageType: types.DamFire, Magnitude: 50}},
			Modifiers:       []StatModifier{{Stat: types.StatStr, Delta: 2}},
			Capabilities:    []Capability{{Key: "class-cap"}},
			Hooks:           []BehaviorHook{{Event: OnLevelUp, Script: "class.lua"}},
		}

		got := Compose(race, class)

		if len(got.Vulnerabilities) != 2 || got.Vulnerabilities[0].Magnitude != 10 || got.Vulnerabilities[1].Magnitude != 20 {
			t.Errorf("Vulnerabilities order = %+v, expected [race(10), class(20)]", got.Vulnerabilities)
		}
		if len(got.Resistances) != 2 || got.Resistances[0].Magnitude != 1 || got.Resistances[1].Magnitude != 2 {
			t.Errorf("Resistances order = %+v, expected [race(1), class(2)]", got.Resistances)
		}
		if len(got.Immunities) != 2 || got.Immunities[0].Magnitude != 100 || got.Immunities[1].Magnitude != 50 {
			t.Errorf("Immunities order = %+v, expected [race(100), class(50)]", got.Immunities)
		}
		if len(got.Modifiers) != 2 || got.Modifiers[0].Delta != 1 || got.Modifiers[1].Delta != 2 {
			t.Errorf("Modifiers order = %+v, expected [race(1), class(2)]", got.Modifiers)
		}
		if len(got.Capabilities) != 2 || got.Capabilities[0].Key != "race-cap" || got.Capabilities[1].Key != "class-cap" {
			t.Errorf("Capabilities order = %+v, expected [race-cap, class-cap]", got.Capabilities)
		}
		if len(got.Hooks) != 2 || got.Hooks[0].Script != "race.lua" || got.Hooks[1].Script != "class.lua" {
			t.Errorf("Hooks order = %+v, expected [race.lua, class.lua]", got.Hooks)
		}
	})

	t.Run("composing N sets preserves left-to-right order", func(t *testing.T) {
		a := TraitSet{Capabilities: []Capability{{Key: "a"}}}
		b := TraitSet{Capabilities: []Capability{{Key: "b"}}}
		c := TraitSet{Capabilities: []Capability{{Key: "c"}}}

		got := Compose(a, b, c)
		want := []string{"a", "b", "c"}
		if len(got.Capabilities) != len(want) {
			t.Fatalf("Capabilities len = %d, expected %d", len(got.Capabilities), len(want))
		}
		for i, key := range want {
			if got.Capabilities[i].Key != key {
				t.Errorf("Capabilities[%d] = %q, expected %q", i, got.Capabilities[i].Key, key)
			}
		}
	})

	t.Run("empty/zero TraitSet composes cleanly without nil-panic", func(t *testing.T) {
		got := Compose(TraitSet{}, TraitSet{})
		if len(got.Vulnerabilities) != 0 || len(got.Capabilities) != 0 || len(got.Hooks) != 0 {
			t.Errorf("composing empty sets should yield empty slices, got %+v", got)
		}
	})

	t.Run("Compose of zero arguments yields empty TraitSet", func(t *testing.T) {
		got := Compose()
		if len(got.Vulnerabilities) != 0 || len(got.Capabilities) != 0 {
			t.Errorf("Compose() with no args should be empty, got %+v", got)
		}
	})
}

func TestMerge(t *testing.T) {
	t.Run("Merge into empty set is identity", func(t *testing.T) {
		var ts TraitSet
		other := TraitSet{
			Capabilities: []Capability{{Key: "flight"}},
			Modifiers:    []StatModifier{{Stat: types.StatStr, Delta: 3}},
		}
		ts.Merge(other)
		if len(ts.Capabilities) != 1 || ts.Capabilities[0].Key != "flight" {
			t.Errorf("Merge into empty = %+v, expected flight capability", ts.Capabilities)
		}
		if len(ts.Modifiers) != 1 || ts.Modifiers[0].Delta != 3 {
			t.Errorf("Merge into empty = %+v, expected +3 str modifier", ts.Modifiers)
		}
	})

	t.Run("Merge appends in place preserving order", func(t *testing.T) {
		ts := TraitSet{Capabilities: []Capability{{Key: "first"}}}
		ts.Merge(TraitSet{Capabilities: []Capability{{Key: "second"}}})
		if len(ts.Capabilities) != 2 || ts.Capabilities[0].Key != "first" || ts.Capabilities[1].Key != "second" {
			t.Errorf("Merge order = %+v, expected [first, second]", ts.Capabilities)
		}
	})

	t.Run("Merge of empty other is identity", func(t *testing.T) {
		ts := TraitSet{Capabilities: []Capability{{Key: "only"}}}
		ts.Merge(TraitSet{})
		if len(ts.Capabilities) != 1 || ts.Capabilities[0].Key != "only" {
			t.Errorf("Merge of empty other changed receiver: %+v", ts.Capabilities)
		}
	})
}

func TestTraitSetCacheFields(t *testing.T) {
	t.Run("fresh Compose result is unresolved", func(t *testing.T) {
		got := Compose(TraitSet{Capabilities: []Capability{{Key: "x"}}})
		if got.resolved {
			t.Error("fresh Compose result should be unresolved (resolved == false)")
		}
	})
}
