package types

import "testing"

func TestAffect(t *testing.T) {
	t.Run("NewAffect creates affect with correct values", func(t *testing.T) {
		aff := NewAffect("sanctuary", 50, 10, ApplyNone, 0, AffSanctuary)

		if aff.Type != "sanctuary" {
			t.Errorf("expected type 'sanctuary', got '%s'", aff.Type)
		}
		if aff.Level != 50 {
			t.Errorf("expected level 50, got %d", aff.Level)
		}
		if aff.Duration != 10 {
			t.Errorf("expected duration 10, got %d", aff.Duration)
		}
		if aff.Location != ApplyNone {
			t.Errorf("expected location ApplyNone, got %d", aff.Location)
		}
		if aff.Modifier != 0 {
			t.Errorf("expected modifier 0, got %d", aff.Modifier)
		}
		if aff.BitVector != AffSanctuary {
			t.Errorf("expected bitvector AffSanctuary, got %d", aff.BitVector)
		}
	})

	t.Run("Affect with stat modifier", func(t *testing.T) {
		aff := NewAffect("giant strength", 30, 24, ApplyStr, 3, 0)

		if aff.Type != "giant strength" {
			t.Errorf("expected type 'giant strength', got '%s'", aff.Type)
		}
		if aff.Location != ApplyStr {
			t.Errorf("expected location ApplyStr, got %d", aff.Location)
		}
		if aff.Modifier != 3 {
			t.Errorf("expected modifier 3, got %d", aff.Modifier)
		}
	})

	t.Run("IsExpired returns true when duration is 0", func(t *testing.T) {
		aff := NewAffect("test", 10, 0, ApplyNone, 0, 0)
		if !aff.IsExpired() {
			t.Error("expected affect with duration 0 to be expired")
		}
	})

	t.Run("IsExpired returns false when duration > 0", func(t *testing.T) {
		aff := NewAffect("test", 10, 5, ApplyNone, 0, 0)
		if aff.IsExpired() {
			t.Error("expected affect with duration 5 to not be expired")
		}
	})

	t.Run("Tick decrements duration", func(t *testing.T) {
		aff := NewAffect("test", 10, 5, ApplyNone, 0, 0)
		aff.Tick()
		if aff.Duration != 4 {
			t.Errorf("expected duration 4 after tick, got %d", aff.Duration)
		}
	})

	t.Run("Tick does not go below 0", func(t *testing.T) {
		aff := NewAffect("test", 10, 0, ApplyNone, 0, 0)
		aff.Tick()
		if aff.Duration != 0 {
			t.Errorf("expected duration to stay at 0, got %d", aff.Duration)
		}
	})

	t.Run("Permanent affects have duration -1", func(t *testing.T) {
		aff := NewAffect("permanent", 10, -1, ApplyNone, 0, 0)
		if aff.IsExpired() {
			t.Error("permanent affect should not be expired")
		}
		aff.Tick()
		if aff.Duration != -1 {
			t.Errorf("permanent affect duration should stay -1, got %d", aff.Duration)
		}
	})
}

func TestAffectList(t *testing.T) {
	t.Run("Add appends affect to list", func(t *testing.T) {
		list := &AffectList{}
		aff := NewAffect("test", 10, 5, ApplyNone, 0, 0)
		list.Add(aff)

		if list.Len() != 1 {
			t.Errorf("expected length 1, got %d", list.Len())
		}
	})

	t.Run("FindByType returns correct affect", func(t *testing.T) {
		list := &AffectList{}
		list.Add(NewAffect("armor", 10, 5, ApplyAC, -20, 0))
		list.Add(NewAffect("bless", 10, 5, ApplyHitroll, 2, 0))

		found := list.FindByType("armor")
		if found == nil {
			t.Fatal("expected to find armor affect")
		}
		if found.Type != "armor" {
			t.Errorf("expected type 'armor', got '%s'", found.Type)
		}
	})

	t.Run("FindByType returns nil when not found", func(t *testing.T) {
		list := &AffectList{}
		list.Add(NewAffect("armor", 10, 5, ApplyAC, -20, 0))

		found := list.FindByType("sanctuary")
		if found != nil {
			t.Error("expected nil for non-existent affect")
		}
	})

	t.Run("HasType returns true when affect exists", func(t *testing.T) {
		list := &AffectList{}
		list.Add(NewAffect("armor", 10, 5, ApplyAC, -20, 0))

		if !list.HasType("armor") {
			t.Error("expected HasType to return true for armor")
		}
	})

	t.Run("Remove removes affect from list", func(t *testing.T) {
		list := &AffectList{}
		aff := NewAffect("armor", 10, 5, ApplyAC, -20, 0)
		list.Add(aff)
		list.Remove(aff)

		if list.Len() != 0 {
			t.Errorf("expected length 0 after remove, got %d", list.Len())
		}
	})

	t.Run("RemoveByType removes all affects of type", func(t *testing.T) {
		list := &AffectList{}
		list.Add(NewAffect("armor", 10, 5, ApplyAC, -20, 0))
		list.Add(NewAffect("bless", 10, 5, ApplyHitroll, 2, 0))
		list.Add(NewAffect("armor", 20, 10, ApplyAC, -30, 0)) // second armor

		list.RemoveByType("armor")

		if list.Len() != 1 {
			t.Errorf("expected length 1 after removing armor affects, got %d", list.Len())
		}
		if list.HasType("armor") {
			t.Error("expected no armor affects after RemoveByType")
		}
	})
}
