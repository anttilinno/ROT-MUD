package combat

import (
	"math/rand"
)

// Dice rolls a number of dice with a given size
// e.g., Dice(2, 6) rolls 2d6
func Dice(number, size int) int {
	if number < 1 || size < 1 {
		return 0
	}

	total := 0
	for i := 0; i < number; i++ {
		total += rand.Intn(size) + 1
	}
	return total
}

// NumberRange returns a random number in the range [low, high]
func NumberRange(low, high int) int {
	if low >= high {
		return low
	}
	return low + rand.Intn(high-low+1)
}

// NumberPercent returns a random number from 1 to 100
func NumberPercent() int {
	return rand.Intn(100) + 1
}

// NumberBits returns a random number with the given number of bits
func NumberBits(bits int) int {
	if bits <= 0 {
		return 0
	}
	return rand.Intn(1 << bits)
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
