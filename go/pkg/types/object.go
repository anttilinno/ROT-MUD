package types

// Object represents an in-game object/item
// Based on OBJ_DATA from merc.h:1837-1869
type Object struct {
	Vnum       int          // Virtual number (template ID)
	Name       string       // Keywords for targeting
	ShortDesc  string       // Short description (in inventory)
	LongDesc   string       // Long description (on ground)
	ItemType   ItemType     // Type of item (weapon, armor, etc.)
	ExtraFlags ItemFlags    // Extra flags (glow, hum, magic, etc.)
	WearFlags  WearFlags    // Where it can be worn/wielded
	WearLoc    WearLocation // Current wear location (if equipped)
	Weight     int          // Weight in pounds
	Cost       int          // Base value in gold
	Level      int          // Minimum level to use
	Condition  int          // Condition (0-100)
	Material   string       // Material (e.g., "steel", "leather")
	Timer      int          // Ticks until decay (-1 = no timer)
	Values     [5]int       // Type-specific values

	// Container/ownership
	InObject  *Object    // If inside a container
	InRoom    *Room      // If on the ground
	CarriedBy *Character // If in inventory/equipped
	Contents  []*Object  // If this is a container
	On        *Object    // If on furniture

	// Enchantment
	Enchanted bool       // Has been enchanted
	Affects   AffectList // Object affects (when worn)
	Owner     string     // Player owner (for quest items)
	Clan      int        // Clan restriction
	Class     int        // Class restriction

	// Extra descriptions
	ExtraDescriptions []*ExtraDescription
}

// NewObject creates a new object
func NewObject(vnum int, shortDesc string, itemType ItemType) *Object {
	return &Object{
		Vnum:      vnum,
		ShortDesc: shortDesc,
		ItemType:  itemType,
		Condition: 100,
		Timer:     -1, // No timer by default
		WearLoc:   WearLocNone,
		Contents:  make([]*Object, 0),
	}
}

// CanTake returns true if the object can be picked up
func (o *Object) CanTake() bool {
	return o.WearFlags.Has(WearTake)
}

// CanWield returns true if the object can be wielded
func (o *Object) CanWield() bool {
	return o.WearFlags.Has(WearWield)
}

// CanHold returns true if the object can be held
func (o *Object) CanHold() bool {
	return o.WearFlags.Has(WearHold)
}

// IsWorn returns true if the object is currently equipped
func (o *Object) IsWorn() bool {
	return o.WearLoc != WearLocNone
}

// IsExpired returns true if the object's timer has reached 0
func (o *Object) IsExpired() bool {
	return o.Timer == 0
}

// Weapon-specific value accessors
func (o *Object) WeaponType() WeaponClass {
	return WeaponClass(o.Values[0])
}

func (o *Object) DiceNumber() int {
	return o.Values[1]
}

func (o *Object) DiceSize() int {
	return o.Values[2]
}

func (o *Object) DamageType() DamageType {
	return DamageType(o.Values[3])
}

// Container-specific value accessors
func (o *Object) Capacity() int {
	return o.Values[0]
}

// ConditionString returns a text description of the object's condition
func (o *Object) ConditionString() string {
	switch {
	case o.Condition >= 100:
		return "perfect"
	case o.Condition >= 90:
		return "excellent"
	case o.Condition >= 75:
		return "good"
	case o.Condition >= 50:
		return "average"
	case o.Condition >= 25:
		return "poor"
	case o.Condition >= 10:
		return "worn"
	default:
		return "terrible"
	}
}

// AddContent adds an object to this container
func (o *Object) AddContent(obj *Object) {
	obj.InObject = o
	obj.InRoom = nil
	obj.CarriedBy = nil
	o.Contents = append(o.Contents, obj)
}

// RemoveContent removes an object from this container
func (o *Object) RemoveContent(obj *Object) {
	for i, item := range o.Contents {
		if item == obj {
			o.Contents = append(o.Contents[:i], o.Contents[i+1:]...)
			obj.InObject = nil
			return
		}
	}
}

// ContentsWeight returns the total weight of contents
func (o *Object) ContentsWeight() int {
	total := 0
	for _, item := range o.Contents {
		total += item.Weight
	}
	return total
}

// TotalWeight returns weight including contents
func (o *Object) TotalWeight() int {
	return o.Weight + o.ContentsWeight()
}
