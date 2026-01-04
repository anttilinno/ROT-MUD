package shops

import (
	"fmt"
	"strings"

	"rotmud/pkg/types"
)

// ShopHandler handles shop-related commands
type ShopHandler struct {
	Registry    *ShopRegistry
	World       WorldInterface
	ObjectIndex map[int]*types.Object // Vnum -> object template
	Output      func(ch *types.Character, msg string)
}

// WorldInterface provides access to world data
type WorldInterface interface {
	GetCurrentHour() int
	GetObjectTemplate(vnum int) *types.Object
	CloneObject(template *types.Object) *types.Object
}

// NewShopHandler creates a new shop handler
func NewShopHandler(registry *ShopRegistry, world WorldInterface) *ShopHandler {
	return &ShopHandler{
		Registry: registry,
		World:    world,
	}
}

// findKeeperInRoom finds a shopkeeper in the character's room
func (h *ShopHandler) findKeeperInRoom(ch *types.Character) (*types.Character, *Shop) {
	if ch.InRoom == nil {
		return nil, nil
	}

	for _, mob := range ch.InRoom.People {
		if mob == ch || !mob.IsNPC() {
			continue
		}

		shop := h.Registry.GetByMob(mob)
		if shop != nil {
			return mob, shop
		}
	}
	return nil, nil
}

// sendToChar sends a message to a character
func (h *ShopHandler) sendToChar(ch *types.Character, format string, args ...interface{}) {
	if h.Output != nil {
		h.Output(ch, fmt.Sprintf(format, args...))
	}
}

// DoBuy handles the 'buy' command
func (h *ShopHandler) DoBuy(ch *types.Character, argument string) {
	keeper, shop := h.findKeeperInRoom(ch)
	if keeper == nil {
		h.sendToChar(ch, "You can't do that here.\r\n")
		return
	}

	// Check if shop is open
	hour := h.World.GetCurrentHour()
	if !shop.IsOpen(hour) {
		h.sendToChar(ch, "%s says 'Sorry, we're closed. Come back later.'\r\n", keeper.ShortDesc)
		return
	}

	if argument == "" {
		h.sendToChar(ch, "Buy what?\r\n")
		return
	}

	// Find the item in keeper's inventory
	var obj *types.Object
	for _, item := range keeper.Inventory {
		if matchesKeyword(item.Name, argument) {
			obj = item
			break
		}
	}

	if obj == nil {
		h.sendToChar(ch, "%s says 'I don't have that item.'\r\n", keeper.ShortDesc)
		return
	}

	// Calculate price
	price := shop.GetSellPrice(obj, ch)

	// Check if player can afford it
	totalGold := ch.Gold + ch.Silver/100
	if totalGold < price {
		h.sendToChar(ch, "You can't afford it. It costs %d gold.\r\n", price)
		return
	}

	// Check weight
	if getCarryWeight(ch)+obj.Weight > maxCarryWeight(ch) {
		h.sendToChar(ch, "You can't carry that much weight.\r\n")
		return
	}

	// Check item count
	if getCarryCount(ch) >= maxCarryCount(ch) {
		h.sendToChar(ch, "You can't carry that many items.\r\n")
		return
	}

	// Check level
	if obj.Level > ch.Level {
		h.sendToChar(ch, "You're not experienced enough to use that.\r\n")
		return
	}

	// Deduct money
	ch.Gold -= price
	if ch.Gold < 0 {
		// Use silver to make up the difference
		silverNeeded := (-ch.Gold * 100)
		ch.Silver -= silverNeeded
		ch.Gold = 0
	}

	// Clone the object for the buyer (don't remove from shop inventory)
	newObj := h.World.CloneObject(obj)
	if newObj == nil {
		h.sendToChar(ch, "Something went wrong.\r\n")
		return
	}

	// Add to player's inventory
	newObj.CarriedBy = ch
	ch.Inventory = append(ch.Inventory, newObj)

	h.sendToChar(ch, "You buy %s for %d gold.\r\n", obj.ShortDesc, price)

	// Notify the room
	if ch.InRoom != nil {
		for _, other := range ch.InRoom.People {
			if other != ch && other.Descriptor != nil {
				h.sendToChar(other, "%s buys %s.\r\n", ch.Name, obj.ShortDesc)
			}
		}
	}
}

// DoSell handles the 'sell' command
func (h *ShopHandler) DoSell(ch *types.Character, argument string) {
	keeper, shop := h.findKeeperInRoom(ch)
	if keeper == nil {
		h.sendToChar(ch, "You can't do that here.\r\n")
		return
	}

	// Check if shop is open
	hour := h.World.GetCurrentHour()
	if !shop.IsOpen(hour) {
		h.sendToChar(ch, "%s says 'Sorry, we're closed. Come back later.'\r\n", keeper.ShortDesc)
		return
	}

	if argument == "" {
		h.sendToChar(ch, "Sell what?\r\n")
		return
	}

	// Find the item in player's inventory
	var obj *types.Object
	var objIndex int
	for i, item := range ch.Inventory {
		if matchesKeyword(item.Name, argument) {
			obj = item
			objIndex = i
			break
		}
	}

	if obj == nil {
		h.sendToChar(ch, "You don't have that.\r\n")
		return
	}

	// Check if item can be sold
	if !obj.WearFlags.Has(types.WearTake) {
		h.sendToChar(ch, "You can't sell that.\r\n")
		return
	}

	// Check if shop buys this type
	if !shop.BuysType(obj.ItemType) {
		h.sendToChar(ch, "%s says 'I don't buy that kind of thing.'\r\n", keeper.ShortDesc)
		return
	}

	// Calculate price
	price := shop.GetBuyPrice(obj, ch)

	if price <= 0 {
		h.sendToChar(ch, "%s says 'That's worthless to me.'\r\n", keeper.ShortDesc)
		return
	}

	// Remove from player's inventory
	ch.Inventory = append(ch.Inventory[:objIndex], ch.Inventory[objIndex+1:]...)
	obj.CarriedBy = nil

	// Pay the player
	ch.Gold += price

	h.sendToChar(ch, "You sell %s for %d gold.\r\n", obj.ShortDesc, price)

	// Notify the room
	if ch.InRoom != nil {
		for _, other := range ch.InRoom.People {
			if other != ch && other.Descriptor != nil {
				h.sendToChar(other, "%s sells %s.\r\n", ch.Name, obj.ShortDesc)
			}
		}
	}
}

// DoList handles the 'list' command
func (h *ShopHandler) DoList(ch *types.Character, argument string) {
	keeper, shop := h.findKeeperInRoom(ch)
	if keeper == nil {
		h.sendToChar(ch, "You can't do that here.\r\n")
		return
	}

	// Check if shop is open
	hour := h.World.GetCurrentHour()
	if !shop.IsOpen(hour) {
		h.sendToChar(ch, "%s says 'Sorry, we're closed. Come back later.'\r\n", keeper.ShortDesc)
		return
	}

	if len(keeper.Inventory) == 0 {
		h.sendToChar(ch, "%s says 'I have nothing for sale right now.'\r\n", keeper.ShortDesc)
		return
	}

	var sb strings.Builder
	sb.WriteString("[Lvl Price Qty] Item\r\n")

	// Group items by vnum and count quantities
	itemCounts := make(map[int]int)
	itemObjects := make(map[int]*types.Object)
	for _, obj := range keeper.Inventory {
		itemCounts[obj.Vnum]++
		if itemObjects[obj.Vnum] == nil {
			itemObjects[obj.Vnum] = obj
		}
	}

	for vnum, obj := range itemObjects {
		price := shop.GetSellPrice(obj, ch)
		qty := itemCounts[vnum]

		// Filter by argument if provided
		if argument != "" && !matchesKeyword(obj.Name, argument) {
			continue
		}

		sb.WriteString(fmt.Sprintf("[%3d %5d %3d] %s\r\n",
			obj.Level, price, qty, obj.ShortDesc))
	}

	h.sendToChar(ch, "%s", sb.String())
}

// DoValue handles the 'value' command
func (h *ShopHandler) DoValue(ch *types.Character, argument string) {
	keeper, shop := h.findKeeperInRoom(ch)
	if keeper == nil {
		h.sendToChar(ch, "You can't do that here.\r\n")
		return
	}

	if argument == "" {
		h.sendToChar(ch, "Value what?\r\n")
		return
	}

	// Find the item in player's inventory
	var obj *types.Object
	for _, item := range ch.Inventory {
		if matchesKeyword(item.Name, argument) {
			obj = item
			break
		}
	}

	if obj == nil {
		h.sendToChar(ch, "You don't have that.\r\n")
		return
	}

	// Check if shop buys this type
	if !shop.BuysType(obj.ItemType) {
		h.sendToChar(ch, "%s says 'I don't buy that kind of thing.'\r\n", keeper.ShortDesc)
		return
	}

	// Calculate price
	price := shop.GetBuyPrice(obj, ch)

	if price <= 0 {
		h.sendToChar(ch, "%s says 'That's worthless to me.'\r\n", keeper.ShortDesc)
		return
	}

	h.sendToChar(ch, "%s says 'I'll give you %d gold for %s.'\r\n",
		keeper.ShortDesc, price, obj.ShortDesc)
}

// Helper functions

func matchesKeyword(keywords, target string) bool {
	if target == "" {
		return false
	}
	target = strings.ToLower(target)
	for _, kw := range strings.Fields(strings.ToLower(keywords)) {
		if strings.HasPrefix(kw, target) {
			return true
		}
	}
	return false
}

func getCarryWeight(ch *types.Character) int {
	total := 0
	for _, obj := range ch.Inventory {
		total += obj.TotalWeight()
	}
	for _, obj := range ch.Equipment {
		if obj != nil {
			total += obj.TotalWeight()
		}
	}
	return total
}

func maxCarryWeight(ch *types.Character) int {
	// Base on strength
	str := ch.GetStat(types.StatStr)
	return 50 + str*10 + ch.Level*5
}

func getCarryCount(ch *types.Character) int {
	return len(ch.Inventory)
}

func maxCarryCount(ch *types.Character) int {
	// Base on dexterity
	dex := ch.GetStat(types.StatDex)
	return 15 + dex/2 + ch.Level/3
}
