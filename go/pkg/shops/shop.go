package shops

import (
	"rotmud/pkg/types"
)

// Shop represents a shopkeeper's shop configuration
type Shop struct {
	KeeperVnum int   // Mob vnum of the shopkeeper
	BuyTypes   []int // Item types the shop buys
	ProfitBuy  int   // Profit margin when buying (100 = base price)
	ProfitSell int   // Profit margin when selling (100 = base price)
	OpenHour   int   // Hour shop opens (0-23)
	CloseHour  int   // Hour shop closes (0-23)
}

// NewShop creates a new shop with default values
func NewShop(keeperVnum int) *Shop {
	return &Shop{
		KeeperVnum: keeperVnum,
		BuyTypes:   make([]int, 0),
		ProfitBuy:  100, // Buy at 100% of base price
		ProfitSell: 100, // Sell at 100% of base price
		OpenHour:   0,   // Always open
		CloseHour:  24,
	}
}

// BuysType checks if shop buys a specific item type
func (s *Shop) BuysType(itemType types.ItemType) bool {
	if len(s.BuyTypes) == 0 {
		return false // No types specified = buy nothing
	}
	for _, t := range s.BuyTypes {
		if t == int(itemType) || t == 0 { // 0 means buy all
			return true
		}
	}
	return false
}

// IsOpen checks if the shop is open at the given hour
func (s *Shop) IsOpen(hour int) bool {
	if s.OpenHour == 0 && s.CloseHour == 24 {
		return true // 24/7
	}
	if s.OpenHour < s.CloseHour {
		return hour >= s.OpenHour && hour < s.CloseHour
	}
	// Wraps midnight (e.g., 22-6)
	return hour >= s.OpenHour || hour < s.CloseHour
}

// GetBuyPrice calculates price shopkeeper pays to buy an item
func (s *Shop) GetBuyPrice(obj *types.Object, ch *types.Character) int {
	if obj == nil {
		return 0
	}

	// Base price
	price := obj.Cost

	// Apply shop profit margin (lower = shop pays less)
	price = price * s.ProfitBuy / 100

	// Apply haggle skill bonus
	haggleBonus := getHaggleBonus(ch)
	price = price * (100 + haggleBonus) / 100

	// Minimum price of 1
	if price < 1 && obj.Cost > 0 {
		price = 1
	}

	return price
}

// GetSellPrice calculates price customer pays to buy an item
func (s *Shop) GetSellPrice(obj *types.Object, ch *types.Character) int {
	if obj == nil {
		return 0
	}

	// Base price
	price := obj.Cost

	// Apply shop profit margin (higher = customer pays more)
	price = price * s.ProfitSell / 100

	// Apply haggle skill bonus (reduces price for good hagglers)
	haggleBonus := getHaggleBonus(ch)
	price = price * (100 - haggleBonus/2) / 100

	// Minimum price of 1
	if price < 1 && obj.Cost > 0 {
		price = 1
	}

	return price
}

// getHaggleBonus returns haggle skill bonus (0-20)
func getHaggleBonus(ch *types.Character) int {
	if ch == nil || ch.IsNPC() || ch.PCData == nil || ch.PCData.Learned == nil {
		return 0
	}

	haggle := ch.PCData.Learned["haggle"]
	if haggle <= 0 {
		return 0
	}

	// Cap bonus at 20%
	bonus := haggle / 5
	if bonus > 20 {
		bonus = 20
	}
	return bonus
}

// ShopRegistry manages all shops
type ShopRegistry struct {
	byKeeperVnum map[int]*Shop
}

// NewShopRegistry creates a new shop registry
func NewShopRegistry() *ShopRegistry {
	return &ShopRegistry{
		byKeeperVnum: make(map[int]*Shop),
	}
}

// Register adds a shop to the registry
func (r *ShopRegistry) Register(shop *Shop) {
	r.byKeeperVnum[shop.KeeperVnum] = shop
}

// GetByKeeper finds a shop by keeper vnum
func (r *ShopRegistry) GetByKeeper(keeperVnum int) *Shop {
	return r.byKeeperVnum[keeperVnum]
}

// GetByMob finds a shop for a mob
func (r *ShopRegistry) GetByMob(mob *types.Character) *Shop {
	if mob == nil || !mob.IsNPC() {
		return nil
	}
	return r.byKeeperVnum[mob.MobVnum]
}

// Count returns number of registered shops
func (r *ShopRegistry) Count() int {
	return len(r.byKeeperVnum)
}
