package traits

import "sync"

// capBitsCeiling is the fixed number of distinct capability bits the registry
// can assign (D-06). It matches the bit width of CapBits (4 words * 64 bits).
const capBitsCeiling = 256

// CapBits is a fixed 256-bit set used to pack interned capability bits.
//
// It is a value-type array (not a heap slice) so that membership tests are
// O(1) and zero-allocation — the foundation for the zero-alloc capability
// query path (SC#4).
type CapBits [4]uint64

// Has reports whether bit is set. O(1) and zero-allocation (value receiver).
func (b CapBits) Has(bit int) bool {
	return b[bit>>6]&(1<<(uint(bit)&63)) != 0
}

// Set sets bit on the receiver (pointer receiver for mutation).
func (b *CapBits) Set(bit int) {
	b[bit>>6] |= 1 << (uint(bit) & 63)
}

// capRegistry interns capability strings to stable bit indices.
//
// Invariants:
//   - Determinism: bits are assigned in registration (first-sight) order, so
//     Resolve output is reproducible across runs for the same input order.
//   - 256-bit ceiling: at most capBitsCeiling distinct bits are ever assigned
//     (D-06). Once full, internCapability returns a false overflow signal
//     rather than panicking or growing unboundedly — forward-looking defense
//     for P3, when the registry accepts TOML-sourced strings (threat T-02-01).
//
// Concurrency: capRegistry and capNextBit are package-level mutable globals
// shared across all TraitSets. capMu guards every read and write so that a
// concurrent intern (during world/player loading) and a concurrent lookup
// (in-game HasCapability) cannot trigger a Go "concurrent map read and map
// write" data race. An uncontended RWMutex read does not allocate, so the
// zero-alloc query path (SC#4) is preserved.
var (
	capMu       sync.RWMutex
	capRegistry = map[string]int{}
	// capNextBit is the next bit index to assign. It only increases; never reused.
	capNextBit int
)

// internCapability returns the stable bit for key, assigning a new bit on first
// sight. The bool reports success: it is false (with a zero bit) only when the
// 256-bit ceiling is reached and key has not been registered yet. No panic.
func internCapability(key string) (int, bool) {
	// Fast path: already registered. Read lock only.
	capMu.RLock()
	if bit, ok := capRegistry[key]; ok {
		capMu.RUnlock()
		return bit, true
	}
	capMu.RUnlock()

	// Slow path: assign a new bit under the write lock, re-checking in case
	// another goroutine registered the key between the RUnlock and Lock.
	capMu.Lock()
	defer capMu.Unlock()
	if bit, ok := capRegistry[key]; ok {
		return bit, true
	}
	if capNextBit >= capBitsCeiling {
		return 0, false
	}
	bit := capNextBit
	capRegistry[key] = bit
	capNextBit++
	return bit, true
}

// lookupCapability returns the bit for an already-registered key without
// inserting. The bool reports whether key was registered. This is the
// non-allocating read path used by the zero-alloc query layer (Plan 02).
func lookupCapability(key string) (int, bool) {
	capMu.RLock()
	bit, ok := capRegistry[key]
	capMu.RUnlock()
	return bit, ok
}
