// Package shops implements the shop and economy system for the ROT MUD.
//
// This package handles NPC shopkeepers, buying/selling items, and
// price calculations including haggle skill bonuses.
//
// # Shop Configuration
//
// Shops are configured in mobile TOML files:
//
//	[[mobiles]]
//	vnum = 3010
//	keywords = ["shopkeeper"]
//	short_desc = "the shopkeeper"
//	[mobiles.shop]
//	buy_types = ["weapon", "armor"]
//	profit_buy = 50      # Pays 50% of base price
//	profit_sell = 150    # Sells at 150% of base price
//	open_hour = 6
//	close_hour = 22
//
// # Shop Commands
//
// Players interact with shops using:
//
//   - list: Show items for sale
//   - buy <item>: Purchase an item
//   - sell <item>: Sell an item to the shop
//   - value <item>: Get a price quote
//
// # Price Calculations
//
// Buy price (player pays):
//
//	base = object.Cost * shop.ProfitSell / 100
//	final = base * (100 - haggleBonus/2) / 100
//
// Sell price (player receives):
//
//	base = object.Cost * shop.ProfitBuy / 100
//	final = base * (100 + haggleBonus) / 100
//
// The haggle skill provides up to 20% bonus on transactions.
//
// # Shop Hours
//
// Shops can have operating hours. Outside these hours:
//
//   - "Sorry, we're closed. Come back later."
//
// Set open_hour = 0 and close_hour = 24 for 24/7 operation.
//
// # Item Types
//
// Shops only buy specific item types listed in buy_types.
// Common configurations:
//
//   - Weapons shop: ["weapon"]
//   - Armor shop: ["armor"]
//   - General store: ["weapon", "armor", "light", "container"]
//   - Magic shop: ["scroll", "potion", "wand", "staff"]
//
// # Usage Example
//
//	registry := shops.NewShopRegistry()
//
//	// Register a shop
//	shop := shops.NewShop(3010)
//	shop.BuyTypes = []int{int(types.ItemTypeWeapon)}
//	shop.ProfitBuy = 50
//	shop.ProfitSell = 150
//	registry.Register(shop)
//
//	// Create handler
//	handler := shops.NewShopHandler(registry)
//	handler.Output = sendToPlayer
//
//	// Process commands
//	handler.DoList(player, "")
//	handler.DoBuy(player, "sword")
package shops
