package combat

import (
	"math/rand"
	"reflect"
	"testing"
)

// capture produces a reproducible fingerprint of the four rollers.
func capture() []int {
	out := make([]int, 0, 1+5+5+5)
	out = append(out, Dice(10, 6))
	for i := 0; i < 5; i++ {
		out = append(out, NumberPercent())
	}
	for i := 0; i < 5; i++ {
		out = append(out, NumberRange(1, 100))
	}
	for i := 0; i < 5; i++ {
		out = append(out, NumberBits(8))
	}
	return out
}

func TestSetRandDeterministic(t *testing.T) {
	restore := SetRand(rand.New(rand.NewSource(42)))
	first := capture()
	restore()

	restore = SetRand(rand.New(rand.NewSource(42)))
	second := capture()
	restore()

	if !reflect.DeepEqual(first, second) {
		t.Fatalf("expected identical sequences under same seed\nfirst:  %v\nsecond: %v", first, second)
	}
}

func TestSetRandRestore(t *testing.T) {
	// Snapshot current value (should be nil in a clean run).
	if defaultRand != nil {
		t.Fatalf("expected defaultRand nil at test start, got %v", defaultRand)
	}

	restore := SetRand(rand.New(rand.NewSource(1)))
	if defaultRand == nil {
		t.Fatal("expected defaultRand set after SetRand")
	}
	restore()
	if defaultRand != nil {
		t.Fatalf("expected defaultRand nil after restore, got %v", defaultRand)
	}
}

func TestDefaultRandFallsThroughToGlobal(t *testing.T) {
	// With no SetRand call, rollers still return values in expected ranges.
	if defaultRand != nil {
		t.Skip("defaultRand not nil at test start; another test leaked")
	}
	for i := 0; i < 100; i++ {
		if v := NumberPercent(); v < 1 || v > 100 {
			t.Fatalf("NumberPercent out of range: %d", v)
		}
		if v := NumberRange(10, 20); v < 10 || v > 20 {
			t.Fatalf("NumberRange out of range: %d", v)
		}
	}
}

func TestEdgeCasesPreserved(t *testing.T) {
	if Dice(0, 6) != 0 {
		t.Error("Dice(0,6) must return 0")
	}
	if Dice(2, 0) != 0 {
		t.Error("Dice(2,0) must return 0")
	}
	if NumberRange(5, 5) != 5 {
		t.Errorf("NumberRange(5,5) want 5 got %d", NumberRange(5, 5))
	}
	if NumberRange(10, 5) != 10 {
		t.Errorf("NumberRange(10,5) want 10 got %d", NumberRange(10, 5))
	}
	if NumberBits(0) != 0 {
		t.Error("NumberBits(0) must return 0")
	}
	if NumberBits(-1) != 0 {
		t.Error("NumberBits(-1) must return 0")
	}
}

func TestCombatSystemRandField(t *testing.T) {
	cs := NewCombatSystem()
	if cs.Rand != nil {
		t.Fatalf("expected zero-value nil Rand, got %v", cs.Rand)
	}
	cs.Rand = rand.New(rand.NewSource(7))
	if cs.Rand == nil {
		t.Fatal("assignment to cs.Rand did not stick")
	}
}

func TestCombatSystemRandFieldTypeByReflection(t *testing.T) {
	var cs CombatSystem
	ty := reflect.TypeOf(cs)
	f, ok := ty.FieldByName("Rand")
	if !ok {
		t.Fatal("CombatSystem has no Rand field")
	}
	want := "*rand.Rand"
	got := f.Type.String()
	if got != want {
		t.Fatalf("Rand field type: want %q got %q", want, got)
	}
}
