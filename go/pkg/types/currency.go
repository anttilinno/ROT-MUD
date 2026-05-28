package types

import (
	"fmt"
	"strings"
)

// Currency model — D&D-style 4-tier denominations.
//
// Internal storage on the Character (Coin, BankCoin) is always a single
// int64 number of copper pieces. The denominations below are only used at
// I/O boundaries — parsing player input ("deposit 5 gold") and rendering
// for display ("5p 3g 2s 1c").
//
// Object.Cost in TOML is interpreted as copper pieces (no data edit required;
// the historical "gold" numbers in area files are now read as copper).

const (
	CopperPerSilver   int64 = 10
	SilverPerGold     int64 = 10
	GoldPerPlatinum   int64 = 10
	CopperPerGold     int64 = CopperPerSilver * SilverPerGold     // 100
	CopperPerPlatinum int64 = CopperPerGold * GoldPerPlatinum     // 1000
)

// CoinAmount is a denominated coin breakdown. Useful for parsing user
// input and emitting structured displays. Internally everything is copper.
type CoinAmount struct {
	Platinum int64
	Gold     int64
	Silver   int64
	Copper   int64
}

// ToCopper collapses a denominated amount to a single copper total.
func (a CoinAmount) ToCopper() int64 {
	return a.Copper +
		a.Silver*CopperPerSilver +
		a.Gold*CopperPerGold +
		a.Platinum*CopperPerPlatinum
}

// CoinFromCopper splits a copper total into the largest-denomination
// breakdown (platinum-first, copper-last remainder).
func CoinFromCopper(copper int64) CoinAmount {
	if copper <= 0 {
		return CoinAmount{}
	}
	pp := copper / CopperPerPlatinum
	copper -= pp * CopperPerPlatinum
	gp := copper / CopperPerGold
	copper -= gp * CopperPerGold
	sp := copper / CopperPerSilver
	copper -= sp * CopperPerSilver
	return CoinAmount{Platinum: pp, Gold: gp, Silver: sp, Copper: copper}
}

// FormatCoin renders a copper total as a compact denominated string,
// omitting any denomination that is zero. Returns "0c" for empty.
//
//	FormatCoin(5034) -> "5p 0g 3s 4c" -> "5p 3s 4c"  (zeros stripped)
func FormatCoin(copper int64) string {
	if copper <= 0 {
		return "0c"
	}
	a := CoinFromCopper(copper)
	var parts []string
	if a.Platinum > 0 {
		parts = append(parts, fmt.Sprintf("%dp", a.Platinum))
	}
	if a.Gold > 0 {
		parts = append(parts, fmt.Sprintf("%dg", a.Gold))
	}
	if a.Silver > 0 {
		parts = append(parts, fmt.Sprintf("%ds", a.Silver))
	}
	if a.Copper > 0 {
		parts = append(parts, fmt.Sprintf("%dc", a.Copper))
	}
	return strings.Join(parts, " ")
}

// FormatCoinLong renders a copper total as a long-form string for score
// sheets and prompts ("5 platinum, 3 silver, 4 copper").
func FormatCoinLong(copper int64) string {
	if copper <= 0 {
		return "no coins"
	}
	a := CoinFromCopper(copper)
	var parts []string
	if a.Platinum > 0 {
		parts = append(parts, fmt.Sprintf("%d platinum", a.Platinum))
	}
	if a.Gold > 0 {
		parts = append(parts, fmt.Sprintf("%d gold", a.Gold))
	}
	if a.Silver > 0 {
		parts = append(parts, fmt.Sprintf("%d silver", a.Silver))
	}
	if a.Copper > 0 {
		parts = append(parts, fmt.Sprintf("%d copper", a.Copper))
	}
	return strings.Join(parts, ", ")
}

// ParseCoinDenom returns the copper-per-unit multiplier for a denomination
// name (case-insensitive, accepts short and long forms). Returns 0 for
// unknown denominations.
func ParseCoinDenom(name string) int64 {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "cp", "copper", "coppers":
		return 1
	case "sp", "silver", "silvers":
		return CopperPerSilver
	case "gp", "gold", "golds":
		return CopperPerGold
	case "pp", "plat", "platinum", "platinums":
		return CopperPerPlatinum
	}
	return 0
}
