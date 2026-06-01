package traits

import "testing"

func TestCapBits(t *testing.T) {
	t.Run("Has/Set round-trip across word boundaries", func(t *testing.T) {
		var b CapBits
		set := []int{0, 63, 64, 255}
		for _, bit := range set {
			b.Set(bit)
		}
		for _, bit := range set {
			if !b.Has(bit) {
				t.Errorf("Has(%d) = false, expected true after Set", bit)
			}
		}
		// Unset neighbors of each boundary should be false.
		for _, bit := range []int{1, 62, 65, 254} {
			if b.Has(bit) {
				t.Errorf("Has(%d) = true, expected false (never set)", bit)
			}
		}
	})

	t.Run("bit 255 is the highest settable bit", func(t *testing.T) {
		var b CapBits
		b.Set(255)
		if !b.Has(255) {
			t.Errorf("Has(255) = false, expected true")
		}
		// Setting bit 255 must not bleed into any other bit.
		count := 0
		for bit := 0; bit < 256; bit++ {
			if b.Has(bit) {
				count++
			}
		}
		if count != 1 {
			t.Errorf("set bits = %d, expected 1 (only bit 255)", count)
		}
	})
}

func TestRegistry(t *testing.T) {
	// Each subtest uses its own registry to stay independent of package state.
	t.Run("interning same key returns same bit", func(t *testing.T) {
		resetRegistry()
		a, ok1 := internCapability("flight")
		b, ok2 := internCapability("flight")
		if !ok1 || !ok2 {
			t.Fatalf("intern reported overflow unexpectedly: ok1=%v ok2=%v", ok1, ok2)
		}
		if a != b {
			t.Errorf("internCapability(\"flight\") = %d then %d, expected stable bit", a, b)
		}
	})

	t.Run("distinct keys get distinct ascending bits in registration order", func(t *testing.T) {
		resetRegistry()
		keys := []string{"flight", "swim", "infravision", "detect"}
		for i, k := range keys {
			bit, ok := internCapability(k)
			if !ok {
				t.Fatalf("internCapability(%q) reported overflow", k)
			}
			if bit != i {
				t.Errorf("internCapability(%q) = %d, expected %d (registration order)", k, bit, i)
			}
		}
	})

	t.Run("lookupCapability does not insert and reports registration", func(t *testing.T) {
		resetRegistry()
		if _, ok := lookupCapability("flight"); ok {
			t.Error("lookupCapability(\"flight\") = ok before registration, expected not ok")
		}
		want, _ := internCapability("flight")
		got, ok := lookupCapability("flight")
		if !ok {
			t.Error("lookupCapability(\"flight\") = not ok after registration, expected ok")
		}
		if got != want {
			t.Errorf("lookupCapability(\"flight\") = %d, expected %d", got, want)
		}
		// A lookup miss must not have grown the registry.
		before := capNextBit
		if _, ok := lookupCapability("never-registered"); ok {
			t.Error("lookupCapability of unknown key reported ok")
		}
		if capNextBit != before {
			t.Errorf("capNextBit grew from %d to %d on lookup miss, expected no growth", before, capNextBit)
		}
	})
}

func TestCapabilityOverflow(t *testing.T) {
	t.Run("257th distinct key overflows without panic", func(t *testing.T) {
		resetRegistry()
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("intern panicked on overflow: %v", r)
			}
		}()
		for i := 0; i < capBitsCeiling; i++ {
			key := "cap-" + itoa(i)
			bit, ok := internCapability(key)
			if !ok {
				t.Fatalf("internCapability(%q) overflowed early at index %d", key, i)
			}
			if bit != i {
				t.Errorf("internCapability(%q) = %d, expected %d", key, bit, i)
			}
		}
		bit, ok := internCapability("cap-overflow")
		if ok {
			t.Errorf("internCapability past ceiling = (%d, true), expected (0, false)", bit)
		}
		if bit != 0 {
			t.Errorf("overflow bit = %d, expected 0", bit)
		}
	})
}

func TestCapabilityZeroAlloc(t *testing.T) {
	t.Run("CapBits.Has is zero-allocation on a registered bit", func(t *testing.T) {
		resetRegistry()
		bit, ok := internCapability("flight")
		if !ok {
			t.Fatalf("intern reported overflow unexpectedly")
		}
		var b CapBits
		b.Set(bit)
		allocs := testing.AllocsPerRun(1000, func() {
			_ = b.Has(bit)
		})
		if allocs != 0 {
			t.Errorf("CapBits.Has allocs = %v, expected 0", allocs)
		}
	})

	t.Run("lookupCapability is zero-allocation on a registered key", func(t *testing.T) {
		resetRegistry()
		if _, ok := internCapability("flight"); !ok {
			t.Fatalf("intern reported overflow unexpectedly")
		}
		allocs := testing.AllocsPerRun(1000, func() {
			_, _ = lookupCapability("flight")
		})
		if allocs != 0 {
			t.Errorf("lookupCapability allocs = %v, expected 0", allocs)
		}
	})
}

// resetRegistry clears the package-level intern state so each test starts from
// a known-empty registry (white-box test helper).
func resetRegistry() {
	capRegistry = map[string]int{}
	capNextBit = 0
}

// itoa is a tiny allocation-free-enough base-10 formatter for test key names.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
