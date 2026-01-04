package shops

import (
	"testing"

	"rotmud/pkg/types"
)

func TestNewShop(t *testing.T) {
	shop := NewShop(3001)

	if shop.KeeperVnum != 3001 {
		t.Errorf("expected KeeperVnum 3001, got %d", shop.KeeperVnum)
	}
	if shop.ProfitBuy != 100 {
		t.Errorf("expected ProfitBuy 100, got %d", shop.ProfitBuy)
	}
	if shop.ProfitSell != 100 {
		t.Errorf("expected ProfitSell 100, got %d", shop.ProfitSell)
	}
	if shop.OpenHour != 0 {
		t.Errorf("expected OpenHour 0, got %d", shop.OpenHour)
	}
	if shop.CloseHour != 24 {
		t.Errorf("expected CloseHour 24, got %d", shop.CloseHour)
	}
}

func TestShopBuysType(t *testing.T) {
	shop := NewShop(3001)

	// Empty buy list = buys nothing
	if shop.BuysType(types.ItemTypeWeapon) {
		t.Error("shop with empty buy list should not buy anything")
	}

	// Add weapon type
	shop.BuyTypes = []int{int(types.ItemTypeWeapon)}
	if !shop.BuysType(types.ItemTypeWeapon) {
		t.Error("shop should buy weapon type")
	}
	if shop.BuysType(types.ItemTypeArmor) {
		t.Error("shop should not buy armor type")
	}

	// Type 0 means buy all
	shop.BuyTypes = []int{0}
	if !shop.BuysType(types.ItemTypeWeapon) {
		t.Error("shop with type 0 should buy weapons")
	}
	if !shop.BuysType(types.ItemTypeArmor) {
		t.Error("shop with type 0 should buy armor")
	}
}

func TestShopIsOpen(t *testing.T) {
	shop := NewShop(3001)

	// Default is 24/7
	for hour := 0; hour < 24; hour++ {
		if !shop.IsOpen(hour) {
			t.Errorf("default shop should be open at hour %d", hour)
		}
	}

	// 9 AM to 5 PM
	shop.OpenHour = 9
	shop.CloseHour = 17

	if shop.IsOpen(8) {
		t.Error("shop should be closed at 8 AM")
	}
	if !shop.IsOpen(9) {
		t.Error("shop should be open at 9 AM")
	}
	if !shop.IsOpen(12) {
		t.Error("shop should be open at noon")
	}
	if !shop.IsOpen(16) {
		t.Error("shop should be open at 4 PM")
	}
	if shop.IsOpen(17) {
		t.Error("shop should be closed at 5 PM")
	}
	if shop.IsOpen(20) {
		t.Error("shop should be closed at 8 PM")
	}

	// Wraps midnight (10 PM to 6 AM)
	shop.OpenHour = 22
	shop.CloseHour = 6

	if !shop.IsOpen(22) {
		t.Error("shop should be open at 10 PM")
	}
	if !shop.IsOpen(0) {
		t.Error("shop should be open at midnight")
	}
	if !shop.IsOpen(5) {
		t.Error("shop should be open at 5 AM")
	}
	if shop.IsOpen(6) {
		t.Error("shop should be closed at 6 AM")
	}
	if shop.IsOpen(12) {
		t.Error("shop should be closed at noon")
	}
}

func TestShopGetBuyPrice(t *testing.T) {
	shop := NewShop(3001)
	shop.ProfitBuy = 50 // Shop pays 50% of value

	obj := &types.Object{
		Cost: 100,
	}

	// No character (no haggle)
	price := shop.GetBuyPrice(obj, nil)
	if price != 50 {
		t.Errorf("expected price 50, got %d", price)
	}

	// Character without haggle
	ch := &types.Character{
		Name: "Test",
	}
	price = shop.GetBuyPrice(obj, ch)
	if price != 50 {
		t.Errorf("expected price 50 without haggle, got %d", price)
	}

	// Character with haggle skill
	ch.PCData = &types.PCData{
		Learned: map[string]int{
			"haggle": 100, // Max haggle = 20% bonus
		},
	}
	price = shop.GetBuyPrice(obj, ch)
	// 50 * (100 + 20) / 100 = 60
	if price != 60 {
		t.Errorf("expected price 60 with max haggle, got %d", price)
	}
}

func TestShopGetSellPrice(t *testing.T) {
	shop := NewShop(3001)
	shop.ProfitSell = 150 // Shop charges 150% of value

	obj := &types.Object{
		Cost: 100,
	}

	// No character (no haggle)
	price := shop.GetSellPrice(obj, nil)
	if price != 150 {
		t.Errorf("expected price 150, got %d", price)
	}

	// Character without haggle
	ch := &types.Character{
		Name: "Test",
	}
	price = shop.GetSellPrice(obj, ch)
	if price != 150 {
		t.Errorf("expected price 150 without haggle, got %d", price)
	}

	// Character with haggle skill (reduces price paid)
	ch.PCData = &types.PCData{
		Learned: map[string]int{
			"haggle": 100, // Max haggle = 10% discount
		},
	}
	price = shop.GetSellPrice(obj, ch)
	// 150 * (100 - 10) / 100 = 135
	if price != 135 {
		t.Errorf("expected price 135 with max haggle, got %d", price)
	}
}

func TestShopGetBuyPriceMinimum(t *testing.T) {
	shop := NewShop(3001)
	shop.ProfitBuy = 1 // Shop pays almost nothing

	obj := &types.Object{
		Cost: 1, // Very cheap item
	}

	price := shop.GetBuyPrice(obj, nil)
	// Minimum price is 1
	if price < 1 {
		t.Errorf("expected minimum price of 1, got %d", price)
	}
}

func TestShopRegistryBasic(t *testing.T) {
	registry := NewShopRegistry()

	shop1 := NewShop(3001)
	shop2 := NewShop(3002)

	registry.Register(shop1)
	registry.Register(shop2)

	if registry.Count() != 2 {
		t.Errorf("expected count 2, got %d", registry.Count())
	}

	found := registry.GetByKeeper(3001)
	if found != shop1 {
		t.Error("expected to find shop1 by keeper vnum")
	}

	found = registry.GetByKeeper(9999)
	if found != nil {
		t.Error("expected nil for unknown keeper")
	}
}

func TestShopRegistryGetByMob(t *testing.T) {
	registry := NewShopRegistry()
	shop := NewShop(3001)
	registry.Register(shop)

	// NPC with matching vnum
	mob := &types.Character{
		Name:    "Shopkeeper",
		MobVnum: 3001,
		Act:     types.ActNPC,
	}

	found := registry.GetByMob(mob)
	if found != shop {
		t.Error("expected to find shop by mob")
	}

	// Player (not NPC)
	player := &types.Character{
		Name: "Player",
	}
	found = registry.GetByMob(player)
	if found != nil {
		t.Error("expected nil for non-NPC")
	}

	// NPC without shop
	otherMob := &types.Character{
		Name:    "Guard",
		MobVnum: 9999,
		Act:     types.ActNPC,
	}
	found = registry.GetByMob(otherMob)
	if found != nil {
		t.Error("expected nil for NPC without shop")
	}
}

func TestGetHaggleBonus(t *testing.T) {
	// Nil character
	bonus := getHaggleBonus(nil)
	if bonus != 0 {
		t.Errorf("expected 0 for nil character, got %d", bonus)
	}

	// NPC
	npc := &types.Character{
		Name: "NPC",
		Act:  types.ActNPC,
	}
	bonus = getHaggleBonus(npc)
	if bonus != 0 {
		t.Errorf("expected 0 for NPC, got %d", bonus)
	}

	// Player without PCData
	player := &types.Character{
		Name: "Player",
	}
	bonus = getHaggleBonus(player)
	if bonus != 0 {
		t.Errorf("expected 0 for player without PCData, got %d", bonus)
	}

	// Player with haggle skill
	player.PCData = &types.PCData{
		Learned: map[string]int{
			"haggle": 50, // 50% haggle = 10% bonus
		},
	}
	bonus = getHaggleBonus(player)
	if bonus != 10 {
		t.Errorf("expected bonus 10, got %d", bonus)
	}

	// Player with max haggle
	player.PCData.Learned["haggle"] = 100
	bonus = getHaggleBonus(player)
	if bonus != 20 {
		t.Errorf("expected max bonus 20, got %d", bonus)
	}

	// Over 100 haggle still caps at 20
	player.PCData.Learned["haggle"] = 150
	bonus = getHaggleBonus(player)
	if bonus != 20 {
		t.Errorf("expected capped bonus 20, got %d", bonus)
	}
}

func TestMatchesKeyword(t *testing.T) {
	tests := []struct {
		keywords string
		target   string
		expected bool
	}{
		{"sword long steel", "sword", true},
		{"sword long steel", "swo", true},
		{"sword long steel", "steel", true},
		{"sword long steel", "axe", false},
		{"sword long steel", "SWORD", true}, // Case insensitive
		{"sword long steel", "", false},
		{"", "sword", false},
	}

	for _, tc := range tests {
		result := matchesKeyword(tc.keywords, tc.target)
		if result != tc.expected {
			t.Errorf("matchesKeyword(%q, %q) = %v, expected %v",
				tc.keywords, tc.target, result, tc.expected)
		}
	}
}

func TestCarryFunctions(t *testing.T) {
	ch := &types.Character{
		Name:  "Test",
		Level: 10,
	}
	ch.PermStats[types.StatStr] = 18
	ch.PermStats[types.StatDex] = 16

	// Max carry weight = 50 + 18*10 + 10*5 = 280
	maxWeight := maxCarryWeight(ch)
	if maxWeight != 280 {
		t.Errorf("expected maxCarryWeight 280, got %d", maxWeight)
	}

	// Max carry count = 15 + 16/2 + 10/3 = 15 + 8 + 3 = 26
	maxCount := maxCarryCount(ch)
	if maxCount != 26 {
		t.Errorf("expected maxCarryCount 26, got %d", maxCount)
	}

	// Empty inventory
	if getCarryCount(ch) != 0 {
		t.Error("expected 0 items in empty inventory")
	}
	if getCarryWeight(ch) != 0 {
		t.Error("expected 0 weight for empty inventory")
	}

	// Add some items
	ch.Inventory = []*types.Object{
		{Weight: 5},
		{Weight: 10},
	}
	if getCarryCount(ch) != 2 {
		t.Errorf("expected 2 items, got %d", getCarryCount(ch))
	}
	if getCarryWeight(ch) != 15 {
		t.Errorf("expected weight 15, got %d", getCarryWeight(ch))
	}
}
