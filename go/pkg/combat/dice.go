package combat

import (
	"math/rand"
)

// defaultRand is an unexported package-level RNG hook. When non-nil, every
// dice-rolling function in this package routes through it instead of the
// global math/rand source. nil means fall through to global math/rand
// (production default).
//
// Intended for deterministic tests (see pkg/golden). Do not call SetRand
// from production code — a non-nil defaultRand serializes all rolls through
// a single unsynchronised source and is not goroutine-safe.
var defaultRand *rand.Rand

// SetRand installs a deterministic RNG source for Dice, NumberRange,
// NumberPercent, and NumberBits. Returns a restore closure that reinstates
// the previous source (including nil to restore the global fallback).
// Intended for tests; call the returned closure from t.Cleanup.
//
// Passing nil restores the global math/rand source.
func SetRand(r *rand.Rand) func() {
	prev := defaultRand
	defaultRand = r
	return func() { defaultRand = prev }
}

// randIntn routes to defaultRand when set, else falls through to the
// package math/rand source. All rollers in this file MUST call randIntn
// (not the global rand package directly) to respect the SetRand test hook.
func randIntn(n int) int {
	if defaultRand != nil {
		return defaultRand.Intn(n)
	}
	return rand.Intn(n)
}

// Dice rolls a number of dice with a given size
// e.g., Dice(2, 6) rolls 2d6
func Dice(number, size int) int {
	if number < 1 || size < 1 {
		return 0
	}

	total := 0
	for i := 0; i < number; i++ {
		total += randIntn(size) + 1
	}
	return total
}

// NumberRange returns a random number in the range [low, high]
func NumberRange(low, high int) int {
	if low >= high {
		return low
	}
	return low + randIntn(high-low+1)
}

// NumberPercent returns a random number from 1 to 100
func NumberPercent() int {
	return randIntn(100) + 1
}

// NumberBits returns a random number with the given number of bits
func NumberBits(bits int) int {
	if bits <= 0 {
		return 0
	}
	return randIntn(1 << bits)
}

// Interpolate linearly interpolates between two values based on level
// level 0 = low, level 32 = high
func Interpolate(level, low, high int) int {
	return low + level*(high-low)/32
}

// Max returns the larger of two integers
func Max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Min returns the smaller of two integers
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Clamp constrains a value between min and max
func Clamp(val, min, max int) int {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}
