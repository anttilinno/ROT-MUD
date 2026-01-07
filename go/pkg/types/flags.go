package types

// ActFlags represents mob/player action flags
type ActFlags uint64

const (
	ActNPC          ActFlags = 1 << iota // Auto set for mobs
	ActSentinel                          // Stays in one room
	ActScavenger                         // Picks up objects
	ActKey                               // Has a key
	ActRanger                            // Ranger mob
	ActAggressive                        // Attacks PCs
	ActStayArea                          // Won't leave area
	ActWimpy                             // Flees when hurt
	ActPet                               // Auto set for pets
	ActTrain                             // Can train PCs
	ActPractice                          // Can practice PCs
	ActDruid                             // Druid mob
	ActNoBody                            // Won't leave a corpse
	ActNBDrop                            // Corpseless will drop all
	ActUndead                            // Undead mob
	ActVampire                           // Vampire mob
	ActCleric                            // Cleric mob
	ActMage                              // Mage mob
	ActThief                             // Thief mob
	ActWarrior                           // Warrior mob
	ActNoAlign                           // No alignment
	ActNoPurge                           // Cannot be purged
	ActOutdoors                          // Outdoor mob
	ActIsSatan                           // Satan mob
	ActIndoors                           // Indoor mob
	ActIsPriest                          // Priest mob
	ActIsHealer                          // Healer mob
	ActGain                              // Can gain
	ActUpdateAlways                      // Always updates
	ActIsChanger                         // Class changer
	ActBanker                            // Can handle bank transactions
)

// Has returns true if the flag is set
func (f ActFlags) Has(flag ActFlags) bool {
	return f&flag != 0
}

// Set adds a flag
func (f *ActFlags) Set(flag ActFlags) {
	*f |= flag
}

// Remove clears a flag
func (f *ActFlags) Remove(flag ActFlags) {
	*f &^= flag
}

// Toggle flips a flag
func (f *ActFlags) Toggle(flag ActFlags) {
	*f ^= flag
}

// AffectFlags represents affect/buff flags on characters
type AffectFlags uint64

const (
	AffBlind        AffectFlags = 1 << iota // Blinded
	AffInvisible                            // Invisible
	AffDetectEvil                           // Can detect evil
	AffDetectInvis                          // Can detect invisible
	AffDetectMagic                          // Can detect magic
	AffDetectHidden                         // Can detect hidden
	AffDetectGood                           // Can detect good
	AffSanctuary                            // Sanctuary (half damage)
	AffFaerieFire                           // Faerie fire (glowing)
	AffInfrared                             // Infrared vision
	AffCurse                                // Cursed
	AffFarsight                             // Farsight
	AffPoison                               // Poisoned
	AffProtectEvil                          // Protected from evil
	AffProtectGood                          // Protected from good
	AffSneak                                // Sneaking
	AffHide                                 // Hidden
	AffSleep                                // Magical sleep
	AffCharm                                // Charmed
	AffFlying                               // Flying
	AffPassDoor                             // Can pass through doors
	AffHaste                                // Hasted
	AffCalm                                 // Calmed
	AffPlague                               // Plagued
	AffWeaken                               // Weakened
	AffDarkVision                           // Dark vision
	AffBerserk                              // Berserking
	AffSwim                                 // Can swim
	AffRegeneration                         // Regenerating
	AffSlow                                 // Slowed
)

// Has returns true if the flag is set
func (f AffectFlags) Has(flag AffectFlags) bool {
	return f&flag != 0
}

// Set adds a flag
func (f *AffectFlags) Set(flag AffectFlags) {
	*f |= flag
}

// Remove clears a flag
func (f *AffectFlags) Remove(flag AffectFlags) {
	*f &^= flag
}

// Toggle flips a flag
func (f *AffectFlags) Toggle(flag AffectFlags) {
	*f ^= flag
}

// ShieldFlags represents shield/protection flags
type ShieldFlags uint64

const (
	ShdProtectVoodoo ShieldFlags = 1 << iota
	ShdInvisible
	ShdIce
	ShdFire
	ShdShock
	_
	_
	ShdSanctuary
	_
	_
	_
	_
	_
	ShdProtectEvil
	ShdProtectGood
)

// Has returns true if the flag is set
func (f ShieldFlags) Has(flag ShieldFlags) bool {
	return f&flag != 0
}

// Set adds a flag
func (f *ShieldFlags) Set(flag ShieldFlags) {
	*f |= flag
}

// Remove clears a flag
func (f *ShieldFlags) Remove(flag ShieldFlags) {
	*f &^= flag
}

// RoomFlags represents room flags
type RoomFlags uint32

const (
	RoomDark        RoomFlags = 1 << iota // Room is dark
	_                                     // Unused
	RoomNoMob                             // Mobs cannot enter
	RoomIndoors                           // Room is indoors
	_                                     // Unused
	_                                     // Unused
	_                                     // Unused
	_                                     // Unused
	_                                     // Unused
	RoomPrivate                           // Room is private (2 people max)
	RoomSafe                              // No combat allowed
	RoomSolitary                          // Only 1 person allowed
	RoomPetShop                           // Pet shop
	RoomNoRecall                          // Cannot recall from here
	RoomImpOnly                           // Implementors only
	RoomGodsOnly                          // Gods only
	RoomHeroesOnly                        // Heroes only
	RoomNewbiesOnly                       // Newbies only
	RoomLaw                               // Law room (no stealing)
	RoomNowhere                           // Room doesn't exist on map
)

// Has returns true if the flag is set
func (f RoomFlags) Has(flag RoomFlags) bool {
	return f&flag != 0
}

// Set adds a flag
func (f *RoomFlags) Set(flag RoomFlags) {
	*f |= flag
}

// Remove clears a flag
func (f *RoomFlags) Remove(flag RoomFlags) {
	*f &^= flag
}

// Toggle flips a flag
func (f *RoomFlags) Toggle(flag RoomFlags) {
	*f ^= flag
}

// ItemFlags represents object extra flags
type ItemFlags uint32

const (
	ItemGlow        ItemFlags = 1 << iota // Item glows
	ItemHum                               // Item hums
	ItemDark                              // Item is dark
	ItemLock                              // Item is locked
	ItemEvil                              // Item is evil
	ItemInvis                             // Item is invisible
	ItemMagic                             // Item is magical
	ItemNoDrop                            // Cannot drop item
	ItemBless                             // Item is blessed
	ItemAntiGood                          // Anti-good
	ItemAntiEvil                          // Anti-evil
	ItemAntiNeutral                       // Anti-neutral
	ItemNoRemove                          // Cannot remove item
	ItemInventory                         // Item is in inventory
	ItemNoPurge                           // Cannot purge item
	ItemRotDeath                          // Rots on death
	ItemVisDeath                          // Visible on death
	ItemNoSac                             // Cannot sacrifice
	ItemNonMetal                          // Non-metal item
	ItemNoLocate                          // Cannot locate
	ItemMeltDrop                          // Melts when dropped
	ItemHadTimer                          // Had a timer
	ItemSellExtract                       // Extracted on sell
	_                                     // Unused
	ItemBurnProof                         // Cannot burn
	ItemNoUncurse                         // Cannot uncurse
	ItemQuest                             // Quest item
	ItemForced                            // Item was forced
	ItemQuestPoint                        // Quest point item
)

// Has returns true if the flag is set
func (f ItemFlags) Has(flag ItemFlags) bool {
	return f&flag != 0
}

// Set adds a flag
func (f *ItemFlags) Set(flag ItemFlags) {
	*f |= flag
}

// Remove clears a flag
func (f *ItemFlags) Remove(flag ItemFlags) {
	*f &^= flag
}

// WearFlags represents where an item can be worn
type WearFlags uint32

const (
	WearTake   WearFlags = 1 << iota // Can be picked up
	WearFinger                       // Worn on finger
	WearNeck                         // Worn on neck
	WearBody                         // Worn on body
	WearHead                         // Worn on head
	WearLegs                         // Worn on legs
	WearFeet                         // Worn on feet
	WearHands                        // Worn on hands
	WearArms                         // Worn on arms
	WearShield                       // Worn as shield
	WearAbout                        // Worn about body
	WearWaist                        // Worn on waist
	WearWrist                        // Worn on wrist
	WearWield                        // Can be wielded
	WearHold                         // Can be held
	WearNoSac                        // Cannot sacrifice
	WearFloat                        // Floats nearby
	WearFace                         // Worn on face
)

// Has returns true if the flag is set
func (f WearFlags) Has(flag WearFlags) bool {
	return f&flag != 0
}

// Set adds a flag
func (f *WearFlags) Set(flag WearFlags) {
	*f |= flag
}

// Remove clears a flag
func (f *WearFlags) Remove(flag WearFlags) {
	*f &^= flag
}

// ExitFlags represents exit/door flags
type ExitFlags uint32

const (
	ExitIsDoor      ExitFlags = 1 << iota // Is a door
	ExitClosed                            // Door is closed
	ExitLocked                            // Door is locked
	_                                     // Unused
	_                                     // Unused
	ExitPickproof                         // Cannot be picked
	ExitNoPass                            // Cannot pass through
	ExitEasy                              // Easy to pick
	ExitHard                              // Hard to pick
	ExitInfuriating                       // Very hard to pick
	ExitNoClose                           // Cannot be closed
	ExitNoLock                            // Cannot be locked
)

// Has returns true if the flag is set
func (f ExitFlags) Has(flag ExitFlags) bool {
	return f&flag != 0
}

// Set adds a flag
func (f *ExitFlags) Set(flag ExitFlags) {
	*f |= flag
}

// Remove clears a flag
func (f *ExitFlags) Remove(flag ExitFlags) {
	*f &^= flag
}

// OffFlags represents offensive capability flags for mobs
type OffFlags uint32

const (
	OffAreaAttack OffFlags = 1 << iota // Area attack
	OffBackstab                        // Can backstab
	OffBash                            // Can bash
	OffBerserk                         // Can berserk
	OffDisarm                          // Can disarm
	OffDodge                           // Can dodge
	OffFade                            // Can fade
	OffFast                            // Fast attacks
	OffKick                            // Can kick
	OffKickDirt                        // Can kick dirt
	OffParry                           // Can parry
	OffRescue                          // Can rescue
	OffTail                            // Tail attack
	OffTrip                            // Can trip
	OffCrush                           // Crush attack
)

// Has returns true if the flag is set
func (f OffFlags) Has(flag OffFlags) bool {
	return f&flag != 0
}

// Set adds a flag
func (f *OffFlags) Set(flag OffFlags) {
	*f |= flag
}

// Remove clears a flag
func (f *OffFlags) Remove(flag OffFlags) {
	*f &^= flag
}

// ImmFlags represents immunity flags
type ImmFlags uint32

const (
	ImmSummon    ImmFlags = 1 << iota // Immune to summon
	ImmCharm                          // Immune to charm
	ImmMagic                          // Immune to magic
	ImmWeapon                         // Immune to weapons
	ImmBash                           // Immune to bash
	ImmPierce                         // Immune to pierce
	ImmSlash                          // Immune to slash
	ImmFire                           // Immune to fire
	ImmCold                           // Immune to cold
	ImmLightning                      // Immune to lightning
	ImmAcid                           // Immune to acid
	ImmPoison                         // Immune to poison
	ImmNegative                       // Immune to negative
	ImmHoly                           // Immune to holy
	ImmEnergy                         // Immune to energy
	ImmMental                         // Immune to mental
	ImmDisease                        // Immune to disease
	ImmDrowning                       // Immune to drowning
	ImmLight                          // Immune to light
	ImmSound                          // Immune to sound
)

// Has returns true if the flag is set
func (f ImmFlags) Has(flag ImmFlags) bool {
	return f&flag != 0
}

// Set adds a flag
func (f *ImmFlags) Set(flag ImmFlags) {
	*f |= flag
}

// Remove clears a flag
func (f *ImmFlags) Remove(flag ImmFlags) {
	*f &^= flag
}

// CommFlags represents communication flags
type CommFlags uint64

const (
	CommQuiet       CommFlags = 1 << iota // Quiet mode
	CommDeaf                              // Deaf to channels
	CommNoWiz                             // No wiznet
	CommNoOOC                             // No OOC channel
	CommNoGossip                          // No gossip
	CommNoAsk                             // No ask channel
	CommNoMusic                           // No music channel
	CommNoClan                            // No clan channel
	CommNoQuote                           // No quote channel
	CommShoutsOff                         // Shouts off
	CommTrueTrust                         // Show true trust
	CommCompact                           // Compact mode
	CommBrief                             // Brief mode
	CommPrompt                            // Show prompt
	CommCombine                           // Combine inventory
	CommTelnetGA                          // Telnet GA
	CommShowAffects                       // Show affects
	CommNoGrats                           // No grats channel
	CommWiped                             // Wiped
	CommNoEmote                           // Cannot emote
	CommNoShout                           // Cannot shout
	CommNoTell                            // Cannot tell
	CommNoChannels                        // No channels
	CommNoCGossip                         // No cgossip
	CommSnoopProof                        // Snoop proof
	CommAFK                               // Away from keyboard
	CommLong                              // Long descriptions
	CommStore                             // Store mode
	CommColor                             // Color enabled
	CommWiznet                            // Wiznet enabled (for immortals)
)

// PlayerFlags represents player-specific toggle flags (auto actions, etc.)
// These are separate from ActFlags which are for mobs
type PlayerFlags uint64

const (
	PlrAutoExit   PlayerFlags = 1 << iota // Show exits automatically
	PlrAutoLoot                           // Auto-loot corpses
	PlrAutoSac                            // Auto-sacrifice corpses
	PlrAutoGold                           // Auto-get gold from corpses
	PlrAutoSplit                          // Auto-split gold with group
	PlrAutoAssist                         // Auto-assist group members
	PlrAutoStore                          // Auto-store mode
	PlrAutoPeek                           // Auto-peek at characters
	PlrHolyLight                          // Immortal holy light (see all)
	PlrNoFollow                           // Cannot be followed
	PlrNoSummon                           // Cannot be summoned
	PlrNoTrans                            // Cannot be transferred
	PlrNoLoot                             // Corpse cannot be looted
	PlrWizInvis                           // Wizard invisible
	PlrIncog                              // Incognito
	PlrNoClan                             // Banned from joining PK clans

	// Penalty flags
	PlrKiller     // Player is a killer
	PlrThief      // Player is a thief
	PlrFrozen     // Player is frozen (cannot act)
	PlrDeny       // Player is denied login
	PlrLog        // Player's commands are logged
	PlrNoShout    // Cannot use shout/yell
	PlrNoTell     // Cannot use tell
	PlrNoEmote    // Cannot use emote
	PlrNoChannels // Cannot use any channels
	PlrNoTitle    // Cannot change title
	PlrNoRestore  // Cannot be restored
)

// Has returns true if the flag is set
func (f CommFlags) Has(flag CommFlags) bool {
	return f&flag != 0
}

// Set adds a flag
func (f *CommFlags) Set(flag CommFlags) {
	*f |= flag
}

// Remove clears a flag
func (f *CommFlags) Remove(flag CommFlags) {
	*f &^= flag
}

// Has returns true if a player flag is set
func (f PlayerFlags) Has(flag PlayerFlags) bool {
	return f&flag != 0
}

// Set adds a player flag
func (f *PlayerFlags) Set(flag PlayerFlags) {
	*f |= flag
}

// Remove clears a player flag
func (f *PlayerFlags) Remove(flag PlayerFlags) {
	*f &^= flag
}

// Toggle flips a player flag
func (f *PlayerFlags) Toggle(flag PlayerFlags) {
	*f ^= flag
}
