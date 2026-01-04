package magic

import (
	"rotmud/pkg/combat"
	"rotmud/pkg/types"
)

// Spell slot numbers (for identifying spells in objects, etc.)
const (
	SlotNull           = 0
	SlotArmor          = 1
	SlotBless          = 2
	SlotBlindness      = 3
	SlotCureLight      = 4
	SlotCureSerious    = 5
	SlotCureCritical   = 6
	SlotHeal           = 7
	SlotMagicMissile   = 8
	SlotFireball       = 9
	SlotLightningBolt  = 10
	SlotSanctuary      = 11
	SlotPoison         = 12
	SlotCurse          = 13
	SlotHaste          = 14
	SlotSlow           = 15
	SlotGiantStrength  = 16
	SlotDetectInvis    = 17
	SlotInvisibility   = 18
	SlotSleep          = 19
	SlotRefresh        = 20
	SlotCureBlindness  = 21
	SlotCurePoison     = 22
	SlotRemoveCurse    = 23
	SlotDetectEvil     = 24
	SlotDetectGood     = 25
	SlotDetectHidden   = 26
	SlotDetectMagic    = 27
	SlotFly            = 28
	SlotPassDoor       = 29
	SlotStoneSkin      = 30
	SlotShield         = 31
	SlotDispelMagic    = 32
	SlotFaerieFire     = 33
	SlotInfravision    = 34
	SlotProtectEvil    = 35
	SlotProtectGood    = 36
	SlotWordOfRecall   = 37
	SlotIdentify       = 38
	SlotCalmness       = 39
	SlotFrenzy         = 40
	SlotCauseLight     = 62
	SlotCauseSerious   = 64
	SlotCauseCritical  = 63
	SlotChillTouch     = 53
	SlotBurningHands   = 54
	SlotShockingGrasp  = 55
	SlotMassHealing    = 508
	SlotTeleport       = 56
	SlotSummon         = 57
	SlotLocateObject   = 58
	SlotFarsight       = 510
	SlotCreateRose     = 511
	SlotPortal         = 519
	SlotNexus          = 520
	SlotFloatingDisc   = 522
	SlotContinualLight = 57

	// New spell slots
	SlotRestoreMana      = 416
	SlotGeneralPurpose   = 417
	SlotHighExplosive    = 418
	SlotProtectionVoodoo = 419
	SlotAnimate          = 239
	SlotResurrect        = 420
	SlotEmpower          = 421
	SlotConjure          = 422
	SlotVoodoo           = 423
)

// DefaultSpells returns a registry populated with default spells
func DefaultSpells() *SpellRegistry {
	r := NewSpellRegistry()

	// === Damage Spells ===

	r.Register(NewSpell("magic missile", SlotMagicMissile, TargetCharOffense, 15, spellMagicMissile).
		SetLevels(map[string]int{"mage": 1, "cleric": 53, "thief": 53, "warrior": 53}).
		SetDamageNoun("magic missile"))

	r.Register(NewSpell("fireball", SlotFireball, TargetCharOffense, 25, spellFireball).
		SetLevels(map[string]int{"mage": 22, "cleric": 53, "thief": 53, "warrior": 53}).
		SetDamageNoun("fireball"))

	r.Register(NewSpell("lightning bolt", SlotLightningBolt, TargetCharOffense, 20, spellLightningBolt).
		SetLevels(map[string]int{"mage": 13, "cleric": 23, "thief": 53, "warrior": 53}).
		SetDamageNoun("lightning bolt"))

	// === Healing Spells ===

	r.Register(NewSpell("cure light", SlotCureLight, TargetCharDefense, 10, spellCureLight).
		SetLevels(map[string]int{"mage": 53, "cleric": 1, "thief": 53, "warrior": 53}))

	r.Register(NewSpell("cure serious", SlotCureSerious, TargetCharDefense, 15, spellCureSerious).
		SetLevels(map[string]int{"mage": 53, "cleric": 7, "thief": 53, "warrior": 53}))

	r.Register(NewSpell("cure critical", SlotCureCritical, TargetCharDefense, 20, spellCureCritical).
		SetLevels(map[string]int{"mage": 53, "cleric": 13, "thief": 53, "warrior": 53}))

	r.Register(NewSpell("heal", SlotHeal, TargetCharDefense, 50, spellHeal).
		SetLevels(map[string]int{"mage": 53, "cleric": 21, "thief": 53, "warrior": 53}))

	r.Register(NewSpell("refresh", SlotRefresh, TargetCharDefense, 12, spellRefresh).
		SetLevels(map[string]int{"mage": 8, "cleric": 5, "thief": 53, "warrior": 53}))

	// === Buff Spells ===

	r.Register(NewSpell("armor", SlotArmor, TargetCharDefense, 5, spellArmor).
		SetLevels(map[string]int{"mage": 7, "cleric": 1, "thief": 53, "warrior": 53}).
		SetWearOff("You feel less armored."))

	r.Register(NewSpell("bless", SlotBless, TargetCharDefense, 5, spellBless).
		SetLevels(map[string]int{"mage": 53, "cleric": 5, "thief": 53, "warrior": 53}).
		SetWearOff("You feel less righteous."))

	r.Register(NewSpell("sanctuary", SlotSanctuary, TargetCharDefense, 75, spellSanctuary).
		SetLevels(map[string]int{"mage": 53, "cleric": 15, "thief": 53, "warrior": 53}).
		SetWearOff("The white aura around your body fades."))

	r.Register(NewSpell("haste", SlotHaste, TargetCharDefense, 30, spellHaste).
		SetLevels(map[string]int{"mage": 21, "cleric": 53, "thief": 53, "warrior": 53}).
		SetWearOff("You feel yourself slow down."))

	r.Register(NewSpell("giant strength", SlotGiantStrength, TargetCharDefense, 20, spellGiantStrength).
		SetLevels(map[string]int{"mage": 11, "cleric": 53, "thief": 53, "warrior": 53}).
		SetWearOff("You feel weaker."))

	r.Register(NewSpell("detect invis", SlotDetectInvis, TargetCharSelf, 5, spellDetectInvis).
		SetLevels(map[string]int{"mage": 3, "cleric": 8, "thief": 53, "warrior": 53}).
		SetWearOff("You no longer see invisible objects."))

	r.Register(NewSpell("invisibility", SlotInvisibility, TargetCharDefense, 5, spellInvisibility).
		SetLevels(map[string]int{"mage": 5, "cleric": 53, "thief": 53, "warrior": 53}).
		SetWearOff("You are no longer invisible."))

	// === Debuff Spells ===

	r.Register(NewSpell("blindness", SlotBlindness, TargetCharOffense, 5, spellBlindness).
		SetLevels(map[string]int{"mage": 12, "cleric": 8, "thief": 53, "warrior": 53}).
		SetWearOff("You can see again."))

	r.Register(NewSpell("curse", SlotCurse, TargetCharOffense, 20, spellCurse).
		SetLevels(map[string]int{"mage": 18, "cleric": 18, "thief": 53, "warrior": 53}).
		SetWearOff("The curse wears off."))

	r.Register(NewSpell("poison", SlotPoison, TargetCharOffense, 10, spellPoison).
		SetLevels(map[string]int{"mage": 17, "cleric": 12, "thief": 53, "warrior": 53}).
		SetWearOff("You feel less sick."))

	r.Register(NewSpell("slow", SlotSlow, TargetCharOffense, 30, spellSlow).
		SetLevels(map[string]int{"mage": 23, "cleric": 30, "thief": 53, "warrior": 53}).
		SetWearOff("You feel yourself speed up."))

	r.Register(NewSpell("sleep", SlotSleep, TargetCharOffense, 15, spellSleep).
		SetLevels(map[string]int{"mage": 10, "cleric": 53, "thief": 53, "warrior": 53}).
		SetWearOff("You feel well rested."))

	// === Cure Spells ===

	r.Register(NewSpell("cure blindness", SlotCureBlindness, TargetCharDefense, 5, spellCureBlindness).
		SetLevels(map[string]int{"mage": 53, "cleric": 6, "thief": 53, "warrior": 53}))

	r.Register(NewSpell("cure poison", SlotCurePoison, TargetCharDefense, 5, spellCurePoison).
		SetLevels(map[string]int{"mage": 53, "cleric": 14, "thief": 53, "warrior": 53}))

	r.Register(NewSpell("remove curse", SlotRemoveCurse, TargetCharDefense, 5, spellRemoveCurse).
		SetLevels(map[string]int{"mage": 53, "cleric": 18, "thief": 53, "warrior": 53}))

	// === Detection Spells ===

	r.Register(NewSpell("detect evil", SlotDetectEvil, TargetCharSelf, 5, spellDetectEvil).
		SetLevels(map[string]int{"mage": 11, "cleric": 4, "thief": 53, "warrior": 53}).
		SetWearOff("The red in your vision disappears."))

	r.Register(NewSpell("detect good", SlotDetectGood, TargetCharSelf, 5, spellDetectGood).
		SetLevels(map[string]int{"mage": 11, "cleric": 4, "thief": 53, "warrior": 53}).
		SetWearOff("The gold in your vision disappears."))

	r.Register(NewSpell("detect hidden", SlotDetectHidden, TargetCharSelf, 5, spellDetectHidden).
		SetLevels(map[string]int{"mage": 15, "cleric": 11, "thief": 53, "warrior": 53}).
		SetWearOff("You feel less aware of your surroundings."))

	r.Register(NewSpell("detect magic", SlotDetectMagic, TargetCharSelf, 5, spellDetectMagic).
		SetLevels(map[string]int{"mage": 2, "cleric": 6, "thief": 53, "warrior": 53}).
		SetWearOff("The detect magic wears off."))

	r.Register(NewSpell("infravision", SlotInfravision, TargetCharDefense, 5, spellInfravision).
		SetLevels(map[string]int{"mage": 9, "cleric": 13, "thief": 53, "warrior": 53}).
		SetWearOff("You no longer see in the dark."))

	// === Movement/Utility Buff Spells ===

	r.Register(NewSpell("fly", SlotFly, TargetCharDefense, 10, spellFly).
		SetLevels(map[string]int{"mage": 10, "cleric": 18, "thief": 53, "warrior": 53}).
		SetWearOff("You slowly float to the ground."))

	r.Register(NewSpell("pass door", SlotPassDoor, TargetCharSelf, 20, spellPassDoor).
		SetLevels(map[string]int{"mage": 24, "cleric": 32, "thief": 53, "warrior": 53}).
		SetWearOff("You feel solid again."))

	// === Defensive Buff Spells ===

	r.Register(NewSpell("stone skin", SlotStoneSkin, TargetCharSelf, 12, spellStoneSkin).
		SetLevels(map[string]int{"mage": 25, "cleric": 53, "thief": 53, "warrior": 53}).
		SetWearOff("Your skin feels soft again."))

	r.Register(NewSpell("shield", SlotShield, TargetCharSelf, 12, spellShield).
		SetLevels(map[string]int{"mage": 20, "cleric": 35, "thief": 53, "warrior": 53}).
		SetWearOff("Your force shield shimmers then fades away."))

	r.Register(NewSpell("protection evil", SlotProtectEvil, TargetCharSelf, 12, spellProtectEvil).
		SetLevels(map[string]int{"mage": 53, "cleric": 9, "thief": 53, "warrior": 53}).
		SetWearOff("You feel less protected."))

	r.Register(NewSpell("protection good", SlotProtectGood, TargetCharSelf, 12, spellProtectGood).
		SetLevels(map[string]int{"mage": 53, "cleric": 9, "thief": 53, "warrior": 53}).
		SetWearOff("You feel less protected."))

	// === Other Debuff Spells ===

	r.Register(NewSpell("faerie fire", SlotFaerieFire, TargetCharOffense, 5, spellFaerieFire).
		SetLevels(map[string]int{"mage": 4, "cleric": 3, "thief": 53, "warrior": 53}).
		SetWearOff("The pink aura around you fades away."))

	// === Utility Spells ===

	r.Register(NewSpell("dispel magic", SlotDispelMagic, TargetCharOffense, 15, spellDispelMagic).
		SetLevels(map[string]int{"mage": 16, "cleric": 24, "thief": 53, "warrior": 53}))

	r.Register(NewSpell("word of recall", SlotWordOfRecall, TargetCharSelf, 5, spellWordOfRecall).
		SetLevels(map[string]int{"mage": 32, "cleric": 28, "thief": 53, "warrior": 53}))

	r.Register(NewSpell("calm", SlotCalmness, TargetCharOffense, 15, spellCalm).
		SetLevels(map[string]int{"mage": 53, "cleric": 16, "thief": 53, "warrior": 53}))

	r.Register(NewSpell("frenzy", SlotFrenzy, TargetCharDefense, 30, spellFrenzy).
		SetLevels(map[string]int{"mage": 53, "cleric": 24, "thief": 53, "warrior": 53}).
		SetWearOff("Your rage ebbs."))

	// === Additional Damage Spells ===

	r.Register(NewSpell("cause light", SlotCauseLight, TargetCharOffense, 15, spellCauseLight).
		SetLevels(map[string]int{"mage": 53, "cleric": 1, "thief": 53, "warrior": 53}).
		SetDamageNoun("spell"))

	r.Register(NewSpell("cause serious", SlotCauseSerious, TargetCharOffense, 17, spellCauseSerious).
		SetLevels(map[string]int{"mage": 53, "cleric": 7, "thief": 53, "warrior": 53}).
		SetDamageNoun("spell"))

	r.Register(NewSpell("cause critical", SlotCauseCritical, TargetCharOffense, 20, spellCauseCritical).
		SetLevels(map[string]int{"mage": 53, "cleric": 13, "thief": 53, "warrior": 53}).
		SetDamageNoun("spell"))

	r.Register(NewSpell("chill touch", SlotChillTouch, TargetCharOffense, 15, spellChillTouch).
		SetLevels(map[string]int{"mage": 4, "cleric": 53, "thief": 53, "warrior": 53}).
		SetDamageNoun("chilling touch").
		SetWearOff("You feel less cold."))

	r.Register(NewSpell("burning hands", SlotBurningHands, TargetCharOffense, 15, spellBurningHands).
		SetLevels(map[string]int{"mage": 7, "cleric": 53, "thief": 53, "warrior": 53}).
		SetDamageNoun("burning hands"))

	r.Register(NewSpell("shocking grasp", SlotShockingGrasp, TargetCharOffense, 15, spellShockingGrasp).
		SetLevels(map[string]int{"mage": 10, "cleric": 53, "thief": 53, "warrior": 53}).
		SetDamageNoun("shocking grasp"))

	// === Healing Spells ===

	r.Register(NewSpell("mass healing", 508, TargetIgnore, 100, spellMassHealing).
		SetLevels(map[string]int{"mage": 53, "cleric": 50, "thief": 53, "warrior": 53}))

	// === Utility Spells ===

	r.Register(NewSpell("teleport", SlotTeleport, TargetCharSelf, 35, spellTeleport).
		SetLevels(map[string]int{"mage": 13, "cleric": 22, "thief": 53, "warrior": 53}))

	r.Register(NewSpell("summon", SlotSummon, TargetCharDefense, 50, spellSummon).
		SetLevels(map[string]int{"mage": 24, "cleric": 32, "thief": 53, "warrior": 53}))

	r.Register(NewSpell("locate object", SlotLocateObject, TargetIgnore, 20, spellLocateObject).
		SetLevels(map[string]int{"mage": 9, "cleric": 15, "thief": 53, "warrior": 53}))

	r.Register(NewSpell("farsight", SlotFarsight, TargetIgnore, 36, spellFarsight).
		SetLevels(map[string]int{"mage": 14, "cleric": 20, "thief": 53, "warrior": 53}))

	// === Utility Spells ===

	r.Register(NewSpell("create food", 12, TargetIgnore, 5, spellCreateFood).
		SetLevels(map[string]int{"mage": 53, "cleric": 3, "thief": 53, "warrior": 53}))

	r.Register(NewSpell("create water", 13, TargetObjInv, 5, spellCreateWater).
		SetLevels(map[string]int{"mage": 53, "cleric": 4, "thief": 53, "warrior": 53}))

	r.Register(NewSpell("create spring", 80, TargetIgnore, 20, spellCreateSpring).
		SetLevels(map[string]int{"mage": 53, "cleric": 17, "thief": 53, "warrior": 53}))

	// === Healing Spells ===

	r.Register(NewSpell("cure disease", 501, TargetCharDefense, 20, spellCureDisease).
		SetLevels(map[string]int{"mage": 53, "cleric": 13, "thief": 53, "warrior": 53}))

	// === Area Effect Spells ===

	r.Register(NewSpell("earthquake", 23, TargetIgnore, 15, spellEarthquake).
		SetLevels(map[string]int{"mage": 53, "cleric": 10, "thief": 53, "warrior": 53}))

	r.Register(NewSpell("call lightning", 6, TargetIgnore, 15, spellCallLightning).
		SetLevels(map[string]int{"mage": 53, "cleric": 18, "thief": 53, "warrior": 53}))

	// === Additional Damage Spells ===

	r.Register(NewSpell("acid blast", 70, TargetCharOffense, 20, spellAcidBlast).
		SetLevels(map[string]int{"mage": 30, "cleric": 53, "thief": 53, "warrior": 53}).
		SetDamageNoun("acid blast"))

	r.Register(NewSpell("colour spray", 10, TargetCharOffense, 15, spellColourSpray).
		SetLevels(map[string]int{"mage": 16, "cleric": 53, "thief": 53, "warrior": 53}).
		SetDamageNoun("colour spray"))

	r.Register(NewSpell("demonfire", 505, TargetCharOffense, 20, spellDemonfire).
		SetLevels(map[string]int{"mage": 53, "cleric": 34, "thief": 53, "warrior": 53}).
		SetDamageNoun("torments"))

	r.Register(NewSpell("energy drain", 25, TargetCharOffense, 35, spellEnergyDrain).
		SetLevels(map[string]int{"mage": 35, "cleric": 53, "thief": 53, "warrior": 53}).
		SetDamageNoun("energy drain"))

	r.Register(NewSpell("flamestrike", 28, TargetCharOffense, 20, spellFlamestrike).
		SetLevels(map[string]int{"mage": 53, "cleric": 20, "thief": 53, "warrior": 53}).
		SetDamageNoun("flamestrike"))

	r.Register(NewSpell("chain lightning", 500, TargetCharOffense, 25, spellChainLightning).
		SetLevels(map[string]int{"mage": 33, "cleric": 53, "thief": 53, "warrior": 53}).
		SetDamageNoun("lightning"))

	r.Register(NewSpell("meteor swarm", 520, TargetCharOffense, 35, spellMeteorSwarm).
		SetLevels(map[string]int{"mage": 45, "cleric": 53, "thief": 53, "warrior": 53}).
		SetDamageNoun("meteors"))

	r.Register(NewSpell("implode", 521, TargetCharOffense, 30, spellImplode).
		SetLevels(map[string]int{"mage": 40, "cleric": 53, "thief": 53, "warrior": 53}).
		SetDamageNoun("implosion"))

	r.Register(NewSpell("disintegrate", 522, TargetCharOffense, 40, spellDisintegrate).
		SetLevels(map[string]int{"mage": 45, "cleric": 53, "thief": 53, "warrior": 53}).
		SetDamageNoun("disintegration"))

	// === Ultimate Spells ===

	r.Register(NewSpell("holy word", 506, TargetIgnore, 200, spellHolyWord).
		SetLevels(map[string]int{"mage": 53, "cleric": 50, "thief": 53, "warrior": 53}))

	r.Register(NewSpell("unholy word", 507, TargetIgnore, 200, spellUnholyWord).
		SetLevels(map[string]int{"mage": 53, "cleric": 50, "thief": 53, "warrior": 53}))

	// === Alignment Spells ===

	r.Register(NewSpell("dispel evil", 220, TargetCharOffense, 15, spellDispelEvil).
		SetDamageNoun("holy fire").
		SetLevels(map[string]int{"mage": 53, "cleric": 15, "thief": 53, "warrior": 53}))

	r.Register(NewSpell("dispel good", 221, TargetCharOffense, 15, spellDispelGood).
		SetDamageNoun("unholy energy").
		SetLevels(map[string]int{"mage": 53, "cleric": 15, "thief": 53, "warrior": 53}))

	r.Register(NewSpell("harm", 222, TargetCharOffense, 35, spellHarm).
		SetDamageNoun("harmful touch").
		SetLevels(map[string]int{"mage": 53, "cleric": 23, "thief": 53, "warrior": 53}))

	// === Transportation Spells ===

	r.Register(NewSpell("gate", 223, TargetIgnore, 80, spellGate).
		SetLevels(map[string]int{"mage": 28, "cleric": 32, "thief": 53, "warrior": 53}))

	// === Shield Spells ===
	// These create elemental shields that damage attackers

	r.Register(NewSpell("fireshield", 409, TargetCharDefense, 20, spellFireShield).
		SetLevels(map[string]int{"mage": 24, "cleric": 53, "thief": 53, "warrior": 53}).
		SetWearOff("Your fiery shield gutters out."))

	r.Register(NewSpell("iceshield", 410, TargetCharDefense, 20, spellIceShield).
		SetLevels(map[string]int{"mage": 24, "cleric": 53, "thief": 53, "warrior": 53}).
		SetWearOff("Your icy shield melts away."))

	r.Register(NewSpell("shockshield", 411, TargetCharDefense, 20, spellShockShield).
		SetLevels(map[string]int{"mage": 24, "cleric": 53, "thief": 53, "warrior": 53}).
		SetWearOff("Your crackling shield sizzles and fades."))

	// === Cleric Holy Spells ===

	r.Register(NewSpell("ray of truth", 412, TargetCharOffense, 20, spellRayOfTruth).
		SetDamageNoun("ray of truth").
		SetLevels(map[string]int{"mage": 53, "cleric": 35, "thief": 53, "warrior": 53}))

	// === Weather/Nature Spells ===

	r.Register(NewSpell("control weather", 413, TargetIgnore, 25, spellControlWeather).
		SetLevels(map[string]int{"mage": 53, "cleric": 17, "thief": 53, "warrior": 53}))

	// === Object Enchantment Spells ===

	r.Register(NewSpell("fireproof", 414, TargetObjInv, 10, spellFireproof).
		SetLevels(map[string]int{"mage": 13, "cleric": 16, "thief": 53, "warrior": 53}))

	// === Metal Manipulation Spells ===

	r.Register(NewSpell("heat metal", 415, TargetCharOffense, 25, spellHeatMetal).
		SetDamageNoun("heat").
		SetLevels(map[string]int{"mage": 53, "cleric": 21, "thief": 53, "warrior": 53}))

	// === Mana Restoration Spells ===

	r.Register(NewSpell("restore mana", SlotRestoreMana, TargetCharDefense, 0, spellRestoreMana).
		SetLevels(map[string]int{"mage": 102, "cleric": 102, "thief": 102, "warrior": 102})) // Healer-only spell

	// === Robot NPC Spells ===

	r.Register(NewSpell("general purpose", SlotGeneralPurpose, TargetCharOffense, 0, spellGeneralPurpose).
		SetDamageNoun("general purpose ammo").
		SetLevels(map[string]int{"mage": 102, "cleric": 102, "thief": 102, "warrior": 102})) // NPC only

	r.Register(NewSpell("high explosive", SlotHighExplosive, TargetCharOffense, 0, spellHighExplosive).
		SetDamageNoun("high explosive ammo").
		SetLevels(map[string]int{"mage": 102, "cleric": 102, "thief": 102, "warrior": 102})) // NPC only

	// === Protection Spells ===

	r.Register(NewSpell("protection voodoo", SlotProtectionVoodoo, TargetCharSelf, 50, spellProtectionVoodoo).
		SetLevels(map[string]int{"mage": 102, "cleric": 102, "thief": 102, "warrior": 102}). // Healer-only
		SetWearOff("Your voodoo protection fades."))

	// === Necromancy Spells ===

	r.Register(NewSpell("animate", SlotAnimate, TargetObjInv, 20, spellAnimate).
		SetLevels(map[string]int{"mage": 102, "cleric": 102, "thief": 102, "warrior": 102, "necromancer": 45, "vampire": 35}))

	r.Register(NewSpell("resurrect", SlotResurrect, TargetIgnore, 50, spellResurrect).
		SetLevels(map[string]int{"mage": 102, "cleric": 102, "thief": 102, "warrior": 102, "necromancer": 50, "vampire": 40}))

	r.Register(NewSpell("conjure", SlotConjure, TargetIgnore, 100, spellConjure).
		SetLevels(map[string]int{"mage": 102, "cleric": 102, "thief": 102, "warrior": 102, "necromancer": 35, "warlock": 30}))

	// === Item Creation Spells ===

	r.Register(NewSpell("empower", SlotEmpower, TargetIgnore, 100, spellEmpower).
		SetLevels(map[string]int{"mage": 35, "cleric": 40, "thief": 102, "warrior": 102, "druid": 45}))

	r.Register(NewSpell("voodoo", SlotVoodoo, TargetIgnore, 50, spellVoodoo).
		SetLevels(map[string]int{"mage": 102, "cleric": 102, "thief": 102, "warrior": 102, "necromancer": 25, "warlock": 20}))

	return r
}

// === Damage Spell Implementations ===

func spellMagicMissile(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		return false
	}

	// 1d4+level damage
	dam := combat.Dice(1, 4) + level
	if dam < 1 {
		dam = 1
	}

	// Magic missile always hits (no save)
	victim.Hit -= dam

	return true
}

func spellFireball(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		return false
	}

	// 3d6 + level*2 damage
	dam := combat.Dice(3, 6) + level*2
	if dam < 1 {
		dam = 1
	}

	// Fire damage - check resistance
	dam = checkDamageResist(victim, dam, types.DamFire)

	victim.Hit -= dam
	return true
}

func spellLightningBolt(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		return false
	}

	// 2d6 + level damage
	dam := combat.Dice(2, 6) + level
	if dam < 1 {
		dam = 1
	}

	// Lightning damage - check resistance
	dam = checkDamageResist(victim, dam, types.DamLightning)

	victim.Hit -= dam
	return true
}

// === Healing Spell Implementations ===

func spellCureLight(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		return false
	}

	heal := combat.Dice(1, 8) + level/3
	victim.Hit = min(victim.Hit+heal, victim.MaxHit)
	return true
}

func spellCureSerious(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		return false
	}

	heal := combat.Dice(2, 8) + level/2
	victim.Hit = min(victim.Hit+heal, victim.MaxHit)
	return true
}

func spellCureCritical(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		return false
	}

	heal := combat.Dice(3, 8) + level
	victim.Hit = min(victim.Hit+heal, victim.MaxHit)
	return true
}

func spellHeal(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		return false
	}

	heal := 100 + level
	victim.Hit = min(victim.Hit+heal, victim.MaxHit)
	return true
}

func spellRefresh(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		return false
	}

	restore := level
	victim.Move = min(victim.Move+restore, victim.MaxMove)
	return true
}

// === Buff Spell Implementations ===

func spellArmor(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		return false
	}

	if IsAffectedBy(victim, "armor") {
		return false // Already affected
	}

	af := NewAffect("armor", level, level+12, types.ApplyAC, -20)
	AddAffect(victim, af)
	return true
}

func spellBless(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		return false
	}

	if IsAffectedBy(victim, "bless") {
		return false
	}

	af := NewAffect("bless", level, level/2+6, types.ApplyHitroll, level/8+1)
	AddAffect(victim, af)

	af2 := NewAffect("bless", level, level/2+6, types.ApplySaves, -(level/8 + 1))
	AddAffect(victim, af2)

	return true
}

func spellSanctuary(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		return false
	}

	if victim.IsAffected(types.AffSanctuary) {
		return false
	}

	af := NewAffectWithBit("sanctuary", level, level/6+10, types.AffSanctuary)
	AddAffect(victim, af)
	return true
}

func spellHaste(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		return false
	}

	if victim.IsAffected(types.AffHaste) || victim.IsAffected(types.AffSlow) {
		return false
	}

	af := NewAffectWithBit("haste", level, level/2+8, types.AffHaste)
	AddAffect(victim, af)

	// Also add dex bonus
	af2 := NewAffect("haste", level, level/2+8, types.ApplyDex, 1+level/20)
	AddAffect(victim, af2)

	return true
}

func spellGiantStrength(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		return false
	}

	if IsAffectedBy(victim, "giant strength") {
		return false
	}

	af := NewAffect("giant strength", level, level+12, types.ApplyStr, 1+level/20)
	AddAffect(victim, af)
	return true
}

func spellDetectInvis(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		return false
	}

	if victim.IsAffected(types.AffDetectInvis) {
		return false
	}

	af := NewAffectWithBit("detect invis", level, level+12, types.AffDetectInvis)
	AddAffect(victim, af)
	return true
}

func spellInvisibility(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		return false
	}

	if victim.IsAffected(types.AffInvisible) {
		return false
	}

	// Can't go invisible while fighting
	if victim.InCombat() {
		return false
	}

	af := NewAffectWithBit("invisibility", level, level+12, types.AffInvisible)
	AddAffect(victim, af)
	return true
}

// === Debuff Spell Implementations ===

func spellBlindness(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		return false
	}

	if victim.IsAffected(types.AffBlind) {
		return false
	}

	// Save negates
	if combat.NumberPercent() < 50+level-victim.Level {
		af := NewAffectWithBit("blindness", level, 1+level/8, types.AffBlind)
		af.Location = types.ApplyHitroll
		af.Modifier = -4
		AddAffect(victim, af)
		return true
	}
	return false
}

func spellCurse(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		return false
	}

	if victim.IsAffected(types.AffCurse) {
		return false
	}

	// Save negates
	if combat.NumberPercent() < 50+level-victim.Level {
		af := NewAffectWithBit("curse", level, 2+level/8, types.AffCurse)
		af.Location = types.ApplyHitroll
		af.Modifier = -level / 8
		AddAffect(victim, af)

		af2 := NewAffect("curse", level, 2+level/8, types.ApplySaves, level/8)
		AddAffect(victim, af2)
		return true
	}
	return false
}

func spellPoison(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		return false
	}

	if victim.IsAffected(types.AffPoison) {
		return false
	}

	// Save negates
	if combat.NumberPercent() < 50+level-victim.Level {
		af := NewAffectWithBit("poison", level, level/2, types.AffPoison)
		af.Location = types.ApplyStr
		af.Modifier = -2
		AddAffect(victim, af)
		return true
	}
	return false
}

func spellSlow(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		return false
	}

	if victim.IsAffected(types.AffSlow) || victim.IsAffected(types.AffHaste) {
		return false
	}

	// Save negates
	if combat.NumberPercent() < 50+level-victim.Level {
		af := NewAffectWithBit("slow", level, level/2+8, types.AffSlow)
		AddAffect(victim, af)

		af2 := NewAffect("slow", level, level/2+8, types.ApplyDex, -(1 + level/20))
		AddAffect(victim, af2)
		return true
	}
	return false
}

func spellSleep(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		return false
	}

	if victim.IsAffected(types.AffSleep) {
		return false
	}

	// Higher level mobs are harder to put to sleep
	if victim.Level > level+4 {
		return false
	}

	// Save negates
	if combat.NumberPercent() < 50+level-victim.Level {
		af := NewAffectWithBit("sleep", level, 4+level/10, types.AffSleep)
		AddAffect(victim, af)

		// Actually put them to sleep
		if victim.Position > types.PosSleeping {
			victim.Position = types.PosSleeping
		}
		return true
	}
	return false
}

// === Helper Functions ===

// checkDamageResist applies damage resistance/vulnerability
func checkDamageResist(ch *types.Character, dam int, damType types.DamageType) int {
	// Check immunity
	immune := combat.CheckImmune(ch, damType)
	switch immune {
	case combat.ImmImmune:
		return 0
	case combat.ImmResistant:
		return dam / 2
	case combat.ImmVulnerable:
		return dam * 3 / 2
	}
	return dam
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// === Cure Spell Implementations ===

func spellCureBlindness(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		return false
	}

	if !victim.IsAffected(types.AffBlind) {
		return false // Not blind
	}

	// Remove blindness affect
	RemoveAffectByType(victim, "blindness")
	return true
}

func spellCurePoison(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		return false
	}

	if !victim.IsAffected(types.AffPoison) {
		return false // Not poisoned
	}

	// Remove poison affect
	RemoveAffectByType(victim, "poison")
	return true
}

func spellRemoveCurse(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		return false
	}

	if !victim.IsAffected(types.AffCurse) {
		return false // Not cursed
	}

	// Remove curse affect
	RemoveAffectByType(victim, "curse")
	return true
}

// === Detection Spell Implementations ===

func spellDetectEvil(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		victim = caster
	}

	if victim.IsAffected(types.AffDetectEvil) {
		return false
	}

	af := NewAffectWithBit("detect evil", level, level+12, types.AffDetectEvil)
	AddAffect(victim, af)
	return true
}

func spellDetectGood(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		victim = caster
	}

	if victim.IsAffected(types.AffDetectGood) {
		return false
	}

	af := NewAffectWithBit("detect good", level, level+12, types.AffDetectGood)
	AddAffect(victim, af)
	return true
}

func spellDetectHidden(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		victim = caster
	}

	if victim.IsAffected(types.AffDetectHidden) {
		return false
	}

	af := NewAffectWithBit("detect hidden", level, level+12, types.AffDetectHidden)
	AddAffect(victim, af)
	return true
}

func spellDetectMagic(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		victim = caster
	}

	if victim.IsAffected(types.AffDetectMagic) {
		return false
	}

	af := NewAffectWithBit("detect magic", level, level+12, types.AffDetectMagic)
	AddAffect(victim, af)
	return true
}

func spellInfravision(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		return false
	}

	if victim.IsAffected(types.AffInfrared) {
		return false
	}

	af := NewAffectWithBit("infravision", level, 2*level+12, types.AffInfrared)
	AddAffect(victim, af)
	return true
}

// === Movement/Utility Buff Implementations ===

func spellFly(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		return false
	}

	if victim.IsAffected(types.AffFlying) {
		return false
	}

	af := NewAffectWithBit("fly", level, level+12, types.AffFlying)
	AddAffect(victim, af)
	return true
}

func spellPassDoor(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		victim = caster
	}

	if victim.IsAffected(types.AffPassDoor) {
		return false
	}

	af := NewAffectWithBit("pass door", level, level/4+8, types.AffPassDoor)
	AddAffect(victim, af)
	return true
}

// === Defensive Buff Implementations ===

func spellStoneSkin(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		victim = caster
	}

	if IsAffectedBy(victim, "stone skin") {
		return false
	}

	af := NewAffect("stone skin", level, level+12, types.ApplyAC, -40)
	AddAffect(victim, af)
	return true
}

func spellShield(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		victim = caster
	}

	if IsAffectedBy(victim, "shield") {
		return false
	}

	af := NewAffect("shield", level, level/2+8, types.ApplyAC, -20)
	AddAffect(victim, af)
	return true
}

func spellProtectEvil(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		victim = caster
	}

	if victim.IsAffected(types.AffProtectEvil) || victim.IsAffected(types.AffProtectGood) {
		return false
	}

	af := NewAffectWithBit("protection evil", level, level+12, types.AffProtectEvil)
	af.Location = types.ApplySaves
	af.Modifier = -1
	AddAffect(victim, af)
	return true
}

func spellProtectGood(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		victim = caster
	}

	if victim.IsAffected(types.AffProtectGood) || victim.IsAffected(types.AffProtectEvil) {
		return false
	}

	af := NewAffectWithBit("protection good", level, level+12, types.AffProtectGood)
	af.Location = types.ApplySaves
	af.Modifier = -1
	AddAffect(victim, af)
	return true
}

// === Other Debuff Implementations ===

func spellFaerieFire(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		return false
	}

	if victim.IsAffected(types.AffFaerieFire) {
		return false
	}

	af := NewAffectWithBit("faerie fire", level, level+12, types.AffFaerieFire)
	af.Location = types.ApplyAC
	af.Modifier = 2 * level // Makes AC worse
	AddAffect(victim, af)
	return true
}

// === Utility Spell Implementations ===

func spellDispelMagic(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		return false
	}

	// Try to dispel each affect
	dispelled := false
	for _, aff := range victim.Affected.All() {
		// Only dispel magic affects, not racial abilities or equipment
		if aff.Duration < 0 {
			continue // Permanent affects can't be dispelled
		}

		// Check if dispel succeeds (based on levels)
		chance := 50 + (caster.Level-aff.Level)*2
		if chance < 5 {
			chance = 5
		}
		if chance > 95 {
			chance = 95
		}

		if combat.NumberPercent() < chance {
			RemoveAffectByType(victim, aff.Type)
			dispelled = true
		}
	}

	return dispelled
}

func spellWordOfRecall(caster *types.Character, level int, target interface{}) bool {
	// Word of recall teleports to temple (vnum 3001)
	// The actual movement is handled by the spell system output
	// This spell just succeeds - movement is handled by MagicSystem.Cast
	return true
}

func spellCalm(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		return false
	}

	if victim.IsAffected(types.AffCalm) {
		return false
	}

	// Stop fighting
	if victim.Fighting != nil {
		combat.StopFighting(victim, true)
	}

	af := NewAffectWithBit("calm", level, level/4+4, types.AffCalm)
	af.Location = types.ApplyHitroll
	af.Modifier = -5
	AddAffect(victim, af)

	af2 := NewAffect("calm", level, level/4+4, types.ApplyDamroll, -5)
	AddAffect(victim, af2)

	return true
}

func spellFrenzy(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		return false
	}

	if victim.IsAffected(types.AffBerserk) || IsAffectedBy(victim, "frenzy") {
		return false
	}

	af := NewAffect("frenzy", level, level/3+12, types.ApplyHitroll, level/6)
	AddAffect(victim, af)

	af2 := NewAffect("frenzy", level, level/3+12, types.ApplyDamroll, level/6)
	AddAffect(victim, af2)

	af3 := NewAffect("frenzy", level, level/3+12, types.ApplyAC, level)
	AddAffect(victim, af3)

	return true
}

// === Additional Spell Implementations ===

// Cause Light - Cleric damage spell
func spellCauseLight(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		return false
	}

	// 1d8 + level/3 damage
	dam := combat.Dice(1, 8) + level/3
	if dam < 1 {
		dam = 1
	}

	victim.Hit -= dam
	return true
}

// Cause Serious - Cleric damage spell
func spellCauseSerious(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		return false
	}

	// 2d8 + level/2 damage
	dam := combat.Dice(2, 8) + level/2
	if dam < 1 {
		dam = 1
	}

	victim.Hit -= dam
	return true
}

// Cause Critical - Cleric damage spell
func spellCauseCritical(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		return false
	}

	// 3d8 + level damage
	dam := combat.Dice(3, 8) + level
	if dam < 1 {
		dam = 1
	}

	victim.Hit -= dam
	return true
}

// Chill Touch - Mage damage spell with strength debuff
func spellChillTouch(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		return false
	}

	// 1d6 + level/2 damage
	dam := combat.Dice(1, 6) + level/2
	if dam < 1 {
		dam = 1
	}

	victim.Hit -= dam

	// 10% chance to reduce strength
	if combat.NumberPercent() < 10 {
		af := NewAffect("chill touch", level, 6, types.ApplyStr, -1)
		AddAffect(victim, af)
	}

	return true
}

// Burning Hands - Mage area damage spell
func spellBurningHands(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		return false
	}

	// 4d3 + level/2 damage
	dam := combat.Dice(4, 3) + level/2
	if dam < 1 {
		dam = 1
	}

	victim.Hit -= dam
	return true
}

// Shocking Grasp - Mage damage spell that works better when wet
func spellShockingGrasp(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		return false
	}

	// 26d4 damage (can be very high)
	dam := combat.Dice(26, 4)
	if dam < 1 {
		dam = 1
	}

	victim.Hit -= dam
	return true
}

// Mass Healing - Cleric area healing spell
func spellMassHealing(caster *types.Character, level int, target interface{}) bool {
	// This would heal all characters in the room
	// For now, just heal the caster
	if caster.Hit < caster.MaxHit {
		heal := combat.Dice(3, 8) + level
		caster.Hit += heal
		if caster.Hit > caster.MaxHit {
			caster.Hit = caster.MaxHit
		}
	}
	return true
}

// Teleport - Mage utility spell
// Note: Actual room selection happens in the magic system after this returns true
// The spell sets up the teleport, but movement is handled by the game loop
func spellTeleport(caster *types.Character, level int, target interface{}) bool {
	// Can't teleport while fighting
	if caster.InCombat() {
		return false
	}

	// Can't teleport from no-recall rooms
	if caster.InRoom != nil && caster.InRoom.Flags.Has(types.RoomNoRecall) {
		return false
	}

	// Success - the actual teleport destination is picked randomly by the game
	// This spell just validates that teleport is allowed
	return true
}

// Summon - Mage utility spell
// Note: Target finding and movement happens in the magic system
func spellSummon(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		return false
	}

	// Can't summon yourself
	if victim == caster {
		return false
	}

	// Can't summon while fighting
	if victim.InCombat() {
		return false
	}

	// Check level difference (can't summon much higher level)
	if victim.Level > level+3 {
		return false
	}

	// Can't summon from no-recall rooms
	if victim.InRoom != nil && victim.InRoom.Flags.Has(types.RoomNoRecall) {
		return false
	}

	// Can't summon to safe rooms
	if caster.InRoom != nil && caster.InRoom.IsSafe() {
		return false
	}

	// Players have a chance to resist summon
	if !victim.IsNPC() {
		// Check if player has nosummon flag
		if victim.PlayerAct.Has(types.PlrNoSummon) {
			return false
		}

		if combat.NumberPercent() < 50+victim.Level-level {
			return false
		}
	}

	// Success - actual movement is handled by game system after spell succeeds
	return true
}

// Gate - Teleport to target's location
// Unlike summon (brings target to you), gate takes you to target
func spellGate(caster *types.Character, level int, target interface{}) bool {
	// Gate requires a character target (passed via targetArg in Cast)
	// The actual target finding is done by the magic system
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		return false
	}

	// Can't gate to yourself
	if victim == caster {
		return false
	}

	// Can't gate while fighting
	if caster.InCombat() {
		return false
	}

	// Check level difference
	if victim.Level >= level+3 {
		return false
	}

	// Can't gate from no-recall rooms
	if caster.InRoom != nil && caster.InRoom.Flags.Has(types.RoomNoRecall) {
		return false
	}

	// Can't gate to restricted rooms
	if victim.InRoom != nil {
		flags := victim.InRoom.Flags
		if flags.Has(types.RoomSafe) || flags.Has(types.RoomPrivate) ||
			flags.Has(types.RoomSolitary) || flags.Has(types.RoomNoRecall) {
			return false
		}
	}

	// Can't gate to immortals
	if !victim.IsNPC() && victim.Level >= 51 {
		return false
	}

	// NPCs get a save based on level difference
	if victim.IsNPC() {
		saveChance := 50 + victim.Level - level
		if combat.NumberPercent() < saveChance {
			return false
		}
	}

	// Success - move caster to victim's room
	if caster.InRoom != nil && victim.InRoom != nil {
		// Remove from current room
		caster.InRoom.RemovePerson(caster)
		// Add to victim's room
		victim.InRoom.AddPerson(caster)
		caster.InRoom = victim.InRoom
	}

	return true
}

// Locate Object - Mage utility spell
func spellLocateObject(caster *types.Character, level int, target interface{}) bool {
	// This spell finds objects in the world matching the target name
	// Actual searching happens in the game system - we just validate here
	return true
}

// Identify - Mage utility spell
// Reveals all properties of an object
func spellIdentify(caster *types.Character, level int, target interface{}) bool {
	// Target can be an object or character
	obj, isObj := target.(*types.Object)
	if isObj && obj != nil {
		// The spell succeeds - actual info display happens in the magic system output
		// Set a flag or use output callback to display info
		return true
	}

	victim, isChar := target.(*types.Character)
	if isChar && victim != nil {
		// Identify on a character shows their stats
		return true
	}

	return false
}

// Farsight - Mage utility spell
// Allows seeing further in scan, revealing hidden exits and distant rooms
func spellFarsight(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		victim = caster
	}

	if victim.IsAffected(types.AffFarsight) {
		return false // Already has farsight
	}

	af := NewAffectWithBit("farsight", level, level, types.AffFarsight)
	AddAffect(victim, af)
	return true
}

// Create Food - Cleric utility spell
func spellCreateFood(caster *types.Character, level int, target interface{}) bool {
	// Create a food item (mushroom or loaf of bread)
	food := types.NewObject(21, "a magic mushroom", types.ItemTypeFood)
	food.Values[0] = level/2 + 5 // Food value
	food.Values[3] = 6           // Hours to decay

	// Put food on ground (would need proper game object management)
	// For now, just succeed
	return true
}

// Create Water - Cleric utility spell
func spellCreateWater(caster *types.Character, level int, target interface{}) bool {
	obj, ok := target.(*types.Object)
	if !ok || obj == nil {
		return false
	}

	if obj.ItemType != types.ItemTypeDrinkCon {
		return false
	}

	// Fill container with water
	amount := level * 2
	if obj.Values[1]+amount > obj.Values[0] {
		amount = obj.Values[0] - obj.Values[1]
	}

	obj.Values[1] += amount
	obj.Values[2] = 0 // Water

	return true
}

// Create Spring - Cleric utility spell
func spellCreateSpring(caster *types.Character, level int, target interface{}) bool {
	// Create a fountain object (would need proper game object management)
	return true
}

// Cure Disease - Cleric healing spell
func spellCureDisease(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		return false
	}

	if victim.IsAffected(types.AffPlague) {
		RemoveAffectByType(victim, "plague")
		return true
	}

	return false
}

// Earthquake - Cleric area damage spell
func spellEarthquake(caster *types.Character, level int, target interface{}) bool {
	dam := combat.Dice(1, 8) + level/2

	// Damage all characters in room except caster
	for _, victim := range caster.InRoom.People {
		if victim == caster {
			continue
		}

		if combat.IsSafe(caster, victim) {
			continue
		}

		victim.Hit -= dam
		if victim.Hit <= 0 {
			victim.Hit = -10 // Mark as dead
		}
	}

	return true
}

// Call Lightning - Cleric area damage spell
func spellCallLightning(caster *types.Character, level int, target interface{}) bool {
	// Only works outdoors (simplified - would need proper sector type check)
	if caster.InRoom != nil && caster.InRoom.Sector == 1 { // Assuming 1 = indoors
		return false
	}

	dam := combat.Dice(level/2, 6)

	// Damage all characters in room except caster
	for _, victim := range caster.InRoom.People {
		if victim == caster {
			continue
		}

		if combat.IsSafe(caster, victim) {
			continue
		}

		victim.Hit -= dam
		if victim.Hit <= 0 {
			victim.Hit = -10 // Mark as dead
		}
	}

	return true
}

// === Additional Spell Implementations ===

// Acid Blast - Mage damage spell
func spellAcidBlast(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		return false
	}

	// 1d8 + level damage
	dam := combat.Dice(1, 8) + level
	if dam < 1 {
		dam = 1
	}

	victim.Hit -= dam
	return true
}

// Colour Spray - Mage damage spell
func spellColourSpray(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		return false
	}

	// 1d6 damage per 2 levels
	dam := combat.Dice(level/2, 6)
	if dam < 1 {
		dam = 1
	}

	victim.Hit -= dam
	return true
}

// Demonfire - Cleric damage spell
func spellDemonfire(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		return false
	}

	// 2d8 + level damage
	dam := combat.Dice(2, 8) + level
	if dam < 1 {
		dam = 1
	}

	victim.Hit -= dam
	return true
}

// Energy Drain - Mage damage spell
func spellEnergyDrain(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		return false
	}

	// 5d6 + level/2 damage
	dam := combat.Dice(5, 6) + level/2
	if dam < 1 {
		dam = 1
	}

	victim.Hit -= dam

	// Chance to reduce victim's level (simplified)
	if combat.NumberPercent() < 20 {
		// Would reduce victim's level in a full implementation
	}

	return true
}

// Flamestrike - Cleric damage spell
func spellFlamestrike(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		return false
	}

	// 6d8 + level damage
	dam := combat.Dice(6, 8) + level
	if dam < 1 {
		dam = 1
	}

	victim.Hit -= dam
	return true
}

// Chain Lightning - Mage area damage spell
func spellChainLightning(caster *types.Character, level int, target interface{}) bool {
	dam := combat.Dice(level/2, 6)

	// Damage all characters in room except caster
	for _, victim := range caster.InRoom.People {
		if victim == caster {
			continue
		}

		if combat.IsSafe(caster, victim) {
			continue
		}

		victim.Hit -= dam
		if victim.Hit <= 0 {
			victim.Hit = -10 // Mark as dead
		}
	}

	return true
}

// Mass Invis - Mage area buff spell
// Makes all group members in the room invisible
func spellMassInvis(caster *types.Character, level int, target interface{}) bool {
	if caster.InRoom == nil {
		return false
	}

	affected := false
	for _, member := range caster.InRoom.People {
		// Only affect group members
		if member == caster ||
			member.Leader == caster ||
			(caster.Leader != nil && member.Leader == caster.Leader) ||
			member.Leader == caster.Leader {

			// Skip if already invisible or fighting
			if member.IsAffected(types.AffInvisible) {
				continue
			}
			if member.InCombat() {
				continue
			}

			af := NewAffectWithBit("mass invis", level, level/4+12, types.AffInvisible)
			AddAffect(member, af)
			affected = true
		}
	}

	// Also affect caster
	if !caster.IsAffected(types.AffInvisible) && !caster.InCombat() {
		af := NewAffectWithBit("mass invis", level, level/4+12, types.AffInvisible)
		AddAffect(caster, af)
		affected = true
	}

	return affected
}

// Know Alignment - Cleric detection spell
// Reveals the target's alignment (good/neutral/evil)
// The actual message output is handled by the game system's spell output
func spellKnowAlignment(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		return false
	}

	// Spell succeeds - alignment info is stored in the victim
	// The game system reads victim.Alignment and generates appropriate message:
	// < -750: satanic, -750 to -350: evil, -350 to 350: neutral,
	// 350 to 750: good, > 750: angelic
	return true
}

// Fire Shield - Mage defensive spell
// Surrounds the target with a fiery shield that damages attackers
func spellFireShield(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		victim = caster
	}

	// Already has fireshield
	if victim.IsShielded(types.ShdFire) {
		return false
	}

	// Shield spells only work on players
	if victim.IsNPC() {
		return false
	}

	af := NewAffectWithShield("fireshield", level, level/6, types.ShdFire)
	AddAffect(victim, af)
	return true
}

// Ice Shield - Mage defensive spell
// Surrounds the target with an icy shield that damages attackers
func spellIceShield(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		victim = caster
	}

	// Already has iceshield
	if victim.IsShielded(types.ShdIce) {
		return false
	}

	// Shield spells only work on players
	if victim.IsNPC() {
		return false
	}

	af := NewAffectWithShield("iceshield", level, level/6, types.ShdIce)
	AddAffect(victim, af)
	return true
}

// Shock Shield - Mage defensive spell
// Surrounds the target with a crackling shield that damages attackers
func spellShockShield(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		victim = caster
	}

	// Already has shockshield
	if victim.IsShielded(types.ShdShock) {
		return false
	}

	// Shield spells only work on players
	if victim.IsNPC() {
		return false
	}

	af := NewAffectWithShield("shockshield", level, level/6, types.ShdShock)
	AddAffect(victim, af)
	return true
}

// Weaken - Mage debuff spell
func spellWeaken(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		return false
	}

	af := NewAffect("weaken", level, level/4+12, types.ApplyStr, -(level/5 + 1))
	AddAffect(victim, af)
	return true
}

// Plague - Cleric debuff spell
func spellPlague(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		return false
	}

	af := NewAffectWithBit("plague", level, level/4+12, types.AffPlague)
	AddAffect(victim, af)
	return true
}

// Charm Person - Mage control spell
func spellCharmPerson(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		return false
	}

	// Can't charm yourself
	if victim == caster {
		return false
	}

	// Check if victim is already charmed
	if victim.IsAffected(types.AffCharm) {
		return false
	}

	// Check if victim is immune to charm
	if victim.Imm.Has(types.ImmCharm) {
		return false
	}

	// Check if victim can be charmed (level check for NPCs)
	if victim.IsNPC() && victim.Level >= level+2 {
		return false
	}

	// Can't charm players in normal gameplay
	if !victim.IsNPC() {
		return false
	}

	// Stop victim from following anyone else
	if victim.Master != nil && victim.Master != caster {
		victim.Master = nil
		victim.Leader = nil
	}

	// Set caster as victim's master
	victim.Master = caster

	af := NewAffectWithBit("charm person", level, level/4+12, types.AffCharm)
	AddAffect(victim, af)
	return true
}

// Ventriloquate - Mage utility spell
// Makes caster's voice appear to come from another character or object
// The actual speech output is handled by the game system
func spellVentriloquate(caster *types.Character, level int, target interface{}) bool {
	// Target can be a character or object
	// Speech will appear to come from the target
	// Viewers get a saving throw to notice the deception
	victim, isChar := target.(*types.Character)
	if isChar && victim != nil {
		return true
	}

	obj, isObj := target.(*types.Object)
	if isObj && obj != nil {
		return true
	}

	return false
}

// Continual Light - Mage/Cleric utility spell
// Makes an object glow, or creates a light ball if no target
func spellContinualLight(caster *types.Character, level int, target interface{}) bool {
	obj, ok := target.(*types.Object)
	if ok && obj != nil {
		// Make the object glow
		if obj.ExtraFlags.Has(types.ItemGlow) {
			return false // Already glowing
		}
		obj.ExtraFlags.Set(types.ItemGlow)
		return true
	}

	// No object target - create a light ball
	// Light ball creation handled via MagicSystem.ObjectCreator callback
	return true
}

// Change Sex - Mage utility spell
// Randomly changes the target's sex
func spellChangeSex(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		return false
	}

	// Already affected
	if IsAffectedBy(victim, "change sex") {
		return false
	}

	// Save negates for hostile targets
	if victim != caster && combat.NumberPercent() < 50+victim.Level-level {
		return false
	}

	// Determine new sex (randomly different from current)
	var newSex types.Sex
	switch victim.Sex {
	case types.SexMale:
		newSex = types.SexFemale
	case types.SexFemale:
		newSex = types.SexMale
	default:
		// Neutral becomes random
		if combat.NumberPercent() < 50 {
			newSex = types.SexMale
		} else {
			newSex = types.SexFemale
		}
	}

	// Store original sex in modifier for restoration
	originalSex := int(victim.Sex)
	victim.Sex = newSex

	af := NewAffect("change sex", level, 10*level, types.ApplySex, originalSex)
	AddAffect(victim, af)
	return true
}

// Cancellation - Mage utility spell
// Removes all magical effects from the target (defensive version of dispel magic)
// Only works on willing targets (self or group members)
func spellCancellation(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		return false
	}

	// Can only cancel self or group members (not enemies)
	if victim != caster {
		// Check if they're in the same group
		if caster.Leader != victim && victim.Leader != caster &&
			(caster.Leader == nil || caster.Leader != victim.Leader) {
			return false
		}
	}

	cancelled := false

	// Remove beneficial and harmful magical effects
	spellsToRemove := []string{
		"armor", "bless", "blindness", "calm", "change sex", "charm person",
		"chill touch", "curse", "detect evil", "detect good", "detect hidden",
		"detect invis", "detect magic", "faerie fire", "fly", "frenzy",
		"giant strength", "haste", "infravision", "invisibility", "mass invis",
		"pass door", "plague", "poison", "protection evil", "protection good",
		"sanctuary", "shield", "sleep", "slow", "stone skin", "weaken",
		"farsight",
	}

	for _, spellName := range spellsToRemove {
		if IsAffectedBy(victim, spellName) {
			RemoveAffectByType(victim, spellName)
			cancelled = true
		}
	}

	return cancelled
}

// Faerie Fog - Mage utility spell
// Reveals hidden and invisible characters in the room
func spellFaerieFog(caster *types.Character, level int, target interface{}) bool {
	if caster.InRoom == nil {
		return false
	}

	revealed := false
	for _, victim := range caster.InRoom.People {
		if victim == caster {
			continue
		}

		// Remove invisibility
		if victim.IsAffected(types.AffInvisible) {
			RemoveAffectByType(victim, "invisibility")
			RemoveAffectByType(victim, "mass invis")
			revealed = true
		}

		// Remove hide
		if victim.IsAffected(types.AffHide) {
			RemoveAffectByType(victim, "hide")
			victim.AffectedBy.Remove(types.AffHide)
			revealed = true
		}

		// Remove sneak
		if victim.IsAffected(types.AffSneak) {
			RemoveAffectByType(victim, "sneak")
			victim.AffectedBy.Remove(types.AffSneak)
			revealed = true
		}
	}

	return revealed || len(caster.InRoom.People) > 1
}

// Detect Poison - Cleric detection spell
func spellDetectPoison(caster *types.Character, level int, target interface{}) bool {
	obj, ok := target.(*types.Object)
	if ok && obj != nil {
		if obj.ItemType == types.ItemTypeFood || obj.ItemType == types.ItemTypeDrinkCon {
			if obj.Values[3] != 0 { // Poisoned
				// Would show poison status
			}
		}
	}
	return true
}

// === Final Spell Implementations ===

// Dispel Evil - Cleric damage spell
func spellDispelEvil(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		return false
	}

	// Only damages evil creatures
	if victim.Alignment > -350 {
		return false
	}

	dam := combat.Dice(6, 8) + level
	if dam < 1 {
		dam = 1
	}

	victim.Hit -= dam
	return true
}

// Dispel Good - Cleric damage spell
func spellDispelGood(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		return false
	}

	// Only damages good creatures
	if victim.Alignment < 350 {
		return false
	}

	dam := combat.Dice(6, 8) + level
	if dam < 1 {
		dam = 1
	}

	victim.Hit -= dam
	return true
}

// Harm - Cleric damage spell
func spellHarm(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		return false
	}

	dam := combat.Dice(10, 10) + level*2
	if dam < 1 {
		dam = 1
	}

	victim.Hit -= dam
	return true
}

// Ray of Truth - Cleric damage spell
// Holy light that damages evil creatures based on how evil they are
// If caster is evil, the spell backfires and hits the caster instead
// Good creatures are unaffected; also blinds the target
func spellRayOfTruth(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		return false
	}

	// If caster is evil, spell backfires
	if caster.IsEvil() {
		victim = caster
	}

	// Good creatures are immune
	if victim.IsGood() {
		return true // Spell succeeds but does nothing
	}

	// Base damage
	dam := combat.Dice(level, 10)

	// Save for half damage
	if combat.NumberPercent() < 50 {
		dam /= 2
	}

	// Damage scales with how evil the target is
	// More evil = more damage
	align := victim.Alignment - 350
	if align < -1000 {
		align = -1000 + (align+1000)/3
	}
	dam = (dam * align * align) / 1000000

	if dam > 0 {
		victim.Hit -= dam
	}

	// Also blinds the target (handled by setting AffBlind)
	if !victim.IsAffected(types.AffBlind) {
		blindDuration := level / 4
		if blindDuration < 1 {
			blindDuration = 1
		}
		af := NewAffectWithBit("blindness", level*3/4, blindDuration, types.AffBlind)
		af.Location = types.ApplyHitroll
		af.Modifier = -4
		AddAffect(victim, af)
	}

	return true
}

// Conjure - Summons a demon to follow the caster
// Requires a demon stone held in the caster's hand
// Based on spell_conjure from magic2.c
func spellConjure(caster *types.Character, level int, target interface{}) bool {
	// Can't be NPC
	if caster.IsNPC() {
		return false
	}

	// Check if caster already has a pet
	if caster.Pet != nil {
		return false
	}

	// Check for demon stone in held position (unless immortal)
	if !caster.IsImmortal() {
		heldItem := caster.GetEquipment(types.WearLocHold)
		if heldItem == nil {
			return false
		}
		// Note: In a full implementation, check for ITEM_DEMON_STONE type
		// and stone->value[0] for charges
	}

	// Spell validates conditions - actual creation handled by MagicSystem.handleConjure
	return true
}

// Nexus - Mage utility spell
// Creates a two-way portal between caster's room and target's room
// Requires a warp stone component (consumed)
func spellNexus(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		return false
	}

	// Can't nexus to yourself
	if victim == caster {
		return false
	}

	// Target must be in a valid room
	if victim.InRoom == nil || caster.InRoom == nil {
		return false
	}

	// Can't nexus to/from safe rooms
	if victim.InRoom.IsSafe() || caster.InRoom.IsSafe() {
		return false
	}

	// Can't nexus to/from no-recall rooms
	if victim.InRoom.Flags.Has(types.RoomNoRecall) || caster.InRoom.Flags.Has(types.RoomNoRecall) {
		return false
	}

	// Can't nexus to private/solitary rooms
	if victim.InRoom.Flags.Has(types.RoomPrivate) || victim.InRoom.Flags.Has(types.RoomSolitary) {
		return false
	}

	// Level restriction
	if victim.Level >= level+3 {
		return false
	}

	// NPCs can resist with saves
	if victim.IsNPC() && victim.Imm.Has(types.ImmSummon) {
		return false
	}

	// Save for NPCs
	if victim.IsNPC() && combat.NumberPercent() < 50+victim.Level-level {
		return false
	}

	// Success - portals are created by game system via ObjectCreator callback
	// Portal timer: 1 + level/10 ticks
	return true
}

// Portal - Mage utility spell
// Creates a one-way portal to target's location
// Requires a warp stone component (consumed)
func spellPortal(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		return false
	}

	// Can't portal to yourself
	if victim == caster {
		return false
	}

	// Target must be in a valid room
	if victim.InRoom == nil || caster.InRoom == nil {
		return false
	}

	// Can't portal to/from safe rooms
	if victim.InRoom.IsSafe() {
		return false
	}

	// Can't portal to/from no-recall rooms
	if victim.InRoom.Flags.Has(types.RoomNoRecall) || caster.InRoom.Flags.Has(types.RoomNoRecall) {
		return false
	}

	// Can't portal to private/solitary rooms
	if victim.InRoom.Flags.Has(types.RoomPrivate) || victim.InRoom.Flags.Has(types.RoomSolitary) {
		return false
	}

	// Level restriction
	if victim.Level >= level+3 {
		return false
	}

	// Hero/immortal players can't be portaled to
	if !victim.IsNPC() && victim.Level >= 51 {
		return false
	}

	// NPCs can resist
	if victim.IsNPC() && victim.Imm.Has(types.ImmSummon) {
		return false
	}

	// Save for NPCs
	if victim.IsNPC() && combat.NumberPercent() < 50+victim.Level-level {
		return false
	}

	// Success - portal is created by game system via ObjectCreator callback
	// Portal timer: 2 + level/25 ticks
	return true
}

// Transport - Mage utility spell
// Transports an object to another player
func spellTransport(caster *types.Character, level int, target interface{}) bool {
	obj, ok := target.(*types.Object)
	if !ok || obj == nil {
		return false
	}

	// Can't transport worn items
	if obj.IsWorn() {
		return false
	}

	// Can't transport cursed items
	if obj.ExtraFlags.Has(types.ItemNoDrop) {
		return false
	}

	// Object must be in caster's inventory
	if obj.CarriedBy != caster {
		return false
	}

	// Success - actual transport to target player handled by game system
	// which finds the target by name from the spell arguments
	return true
}

// Recharge - Mage utility spell
// Recharges wands and staves with a chance of failure or explosion
func spellRecharge(caster *types.Character, level int, target interface{}) bool {
	obj, ok := target.(*types.Object)
	if !ok || obj == nil {
		return false
	}

	if obj.ItemType != types.ItemTypeWand && obj.ItemType != types.ItemTypeStaff {
		return false // Not a chargeable item
	}

	// Values: [0]=spell level, [1]=max charges (0=already recharged once), [2]=current charges, [3]=spell
	spellLevel := obj.Values[0]
	maxCharges := obj.Values[1]
	currentCharges := obj.Values[2]

	// Can only recharge items at or below 3/4 of caster level
	if spellLevel >= 3*level/2 {
		return false // Spell level too high
	}

	// Can only recharge once (max charges becomes 0 after first recharge)
	if maxCharges == 0 {
		return false // Already recharged once
	}

	// Calculate success chance
	chance := 40 + 2*level
	chance -= spellLevel // Harder to do high-level spells
	chargesMissing := maxCharges - currentCharges
	chance -= chargesMissing * chargesMissing // Much harder if many charges used

	if chance < level/2 {
		chance = level / 2
	}

	percent := combat.NumberPercent()

	// Excellent success - full recharge
	if percent < chance/2 {
		if maxCharges > currentCharges {
			obj.Values[2] = maxCharges // Restore to max
		}
		obj.Values[1] = 0 // Mark as already recharged
		return true
	}

	// Good success - partial recharge
	if percent <= chance {
		if chargesMissing > 0 {
			chargeback := chargesMissing * percent / 100
			if chargeback < 1 {
				chargeback = 1
			}
			obj.Values[2] += chargeback
		}
		obj.Values[1] = 0 // Mark as already recharged
		return true
	}

	// Mild failure - nothing happens, loses potential charge
	if percent <= 95 && percent <= 3*chance/2 {
		if maxCharges > 1 {
			obj.Values[1]-- // Reduce max by 1
		}
		return false
	}

	// Catastrophic failure - item explodes!
	obj.Timer = -999 // Mark for destruction
	return false
}

// Enchant Armor - Mage utility spell
// Adds AC bonus to armor with chance of failure/destruction
func spellEnchantArmor(caster *types.Character, level int, target interface{}) bool {
	obj, ok := target.(*types.Object)
	if !ok || obj == nil {
		return false
	}

	if obj.ItemType != types.ItemTypeArmor {
		return false
	}

	// Can't enchant worn items
	if obj.IsWorn() {
		return false
	}

	// Calculate failure chance based on existing enchantments
	fail := 25 // Base 25% failure chance

	// Check existing affects on armor
	for _, aff := range obj.Affects.All() {
		if aff.Location == types.ApplyAC {
			// Harder to enchant already enchanted items
			fail += 5 * (aff.Modifier * aff.Modifier)
		} else {
			fail += 20
		}
	}

	// Level helps
	fail -= level

	// Blessed/glowing items easier to enchant
	if obj.ExtraFlags.Has(types.ItemBless) {
		fail -= 15
	}
	if obj.ExtraFlags.Has(types.ItemGlow) {
		fail -= 5
	}

	// Clamp failure chance
	if fail < 5 {
		fail = 5
	}
	if fail > 85 {
		fail = 85
	}

	result := combat.NumberRange(1, 100)

	// Catastrophic failure - item destroyed
	if result < fail/5 {
		// Object is destroyed (set a flag for caller to handle)
		obj.Timer = -999 // Mark for destruction
		return false
	}

	// Moderate failure - item disenchanted
	if result < fail/3 {
		obj.Affects.Clear()
		obj.ExtraFlags = 0
		return false
	}

	// Regular failure
	if result <= fail {
		return false
	}

	// Success! Add or improve AC bonus
	added := -1 // AC bonus (negative is better)
	if result > 90-level/5 {
		// Exceptional success
		added = -2
		obj.ExtraFlags.Set(types.ItemGlow)
	}

	obj.ExtraFlags.Set(types.ItemMagic)

	// Find existing AC affect or add new one
	acFound := false
	for _, aff := range obj.Affects.All() {
		if aff.Location == types.ApplyAC {
			aff.Modifier += added
			aff.Level = level
			acFound = true
			break
		}
	}

	if !acFound {
		// Add new AC affect
		aff := types.NewAffect("enchant armor", level, -1, types.ApplyAC, added, 0)
		obj.Affects.Add(aff)
	}

	// Increase level slightly
	if obj.Level < 91 {
		obj.Level++
	}

	return true
}

// Enchant Weapon - Mage utility spell
// Adds hitroll/damroll bonus to weapons with chance of failure/destruction
func spellEnchantWeapon(caster *types.Character, level int, target interface{}) bool {
	obj, ok := target.(*types.Object)
	if !ok || obj == nil {
		return false
	}

	if obj.ItemType != types.ItemTypeWeapon {
		return false
	}

	// Can't enchant wielded items
	if obj.IsWorn() {
		return false
	}

	// Calculate failure chance
	fail := 25

	// Check existing affects
	for _, aff := range obj.Affects.All() {
		if aff.Location == types.ApplyHitroll || aff.Location == types.ApplyDamroll {
			fail += 5 * (aff.Modifier * aff.Modifier)
		} else {
			fail += 20
		}
	}

	fail -= level

	if obj.ExtraFlags.Has(types.ItemBless) {
		fail -= 15
	}
	if obj.ExtraFlags.Has(types.ItemGlow) {
		fail -= 5
	}

	if fail < 5 {
		fail = 5
	}
	if fail > 95 {
		fail = 95
	}

	result := combat.NumberRange(1, 100)

	// Catastrophic failure
	if result < fail/5 {
		obj.Timer = -999 // Mark for destruction
		return false
	}

	// Moderate failure
	if result < fail/3 {
		obj.Affects.Clear()
		obj.ExtraFlags = 0
		return false
	}

	// Regular failure
	if result <= fail {
		return false
	}

	// Success!
	added := 1 // Hitroll/damroll bonus
	if result > 90-level/5 {
		added = 2
		obj.ExtraFlags.Set(types.ItemGlow)
	}

	obj.ExtraFlags.Set(types.ItemMagic)

	// Find and improve existing hitroll/damroll or add new
	hitFound := false
	damFound := false

	for _, aff := range obj.Affects.All() {
		if aff.Location == types.ApplyHitroll {
			aff.Modifier += added
			aff.Level = level
			hitFound = true
		}
		if aff.Location == types.ApplyDamroll {
			aff.Modifier += added
			aff.Level = level
			damFound = true
		}
	}

	if !hitFound {
		aff := types.NewAffect("enchant weapon", level, -1, types.ApplyHitroll, added, 0)
		obj.Affects.Add(aff)
	}
	if !damFound {
		aff := types.NewAffect("enchant weapon", level, -1, types.ApplyDamroll, added, 0)
		obj.Affects.Add(aff)
	}

	// Increase level slightly
	if obj.Level < 91 {
		obj.Level++
	}

	return true
}

// Floating Disc - Mage utility spell
// Creates a floating container that follows the caster
// The disc has capacity based on caster level
func spellFloatingDisc(caster *types.Character, level int, target interface{}) bool {
	// Create a floating disc (container object)
	// Capacity: 10 pounds per level
	// Max item weight: 5 pounds per level
	// Duration: 2*level ticks (minus random amount)

	// The disc is created and equipped by the game system
	// This spell just validates and returns success
	// Actual disc creation handled via MagicSystem.ObjectCreator callback
	return true
}

// Create Rose - Cleric creation spell
// Creates a beautiful red rose (gift item)
func spellCreateRose(caster *types.Character, level int, target interface{}) bool {
	// Rose is created and given to caster by game system
	// This spell just validates and returns success
	// Actual rose creation handled via MagicSystem.ObjectCreator callback
	return true
}

// Breath Weapons (for special creatures)
func spellAcidBreath(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		return false
	}

	dam := combat.Dice(10, 16) + level
	victim.Hit -= dam
	return true
}

func spellFireBreath(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		return false
	}

	dam := combat.Dice(11, 20) + level
	victim.Hit -= dam
	return true
}

func spellFrostBreath(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		return false
	}

	dam := combat.Dice(8, 12) + level
	victim.Hit -= dam
	return true
}

func spellGasBreath(caster *types.Character, level int, target interface{}) bool {
	// Area effect gas attack
	for _, victim := range caster.InRoom.People {
		if victim == caster {
			continue
		}
		dam := combat.Dice(6, 10) + level/2
		victim.Hit -= dam
	}
	return true
}

func spellLightningBreath(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		return false
	}

	dam := combat.Dice(10, 14) + level
	victim.Hit -= dam
	return true
}

// === Ultimate Spell Implementations ===

// Meteor Swarm - Mage ultimate damage spell
func spellMeteorSwarm(caster *types.Character, level int, target interface{}) bool {
	dam := combat.Dice(40, 10) + level*2

	// Massive damage to all in room
	for _, victim := range caster.InRoom.People {
		if victim == caster {
			continue
		}

		victim.Hit -= dam
		if victim.Hit <= 0 {
			victim.Hit = -10
		}
	}

	return true
}

// Implode - Mage high-level damage spell
func spellImplode(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		return false
	}

	dam := combat.Dice(20, 10) + level*2
	victim.Hit -= dam
	return true
}

// Disintegrate - Mage ultimate damage spell
func spellDisintegrate(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		return false
	}

	dam := combat.Dice(30, 10) + level*3
	victim.Hit -= dam
	return true
}

// Holy Word - Cleric ultimate area spell
func spellHolyWord(caster *types.Character, level int, target interface{}) bool {
	// Damage evil creatures, heal good ones
	for _, victim := range caster.InRoom.People {
		if victim == caster {
			continue
		}

		// Simplified alignment check
		if victim.Alignment < 0 { // Evil
			dam := combat.Dice(20, 10) + level*2
			victim.Hit -= dam
		} else if victim.Alignment > 0 { // Good
			heal := combat.Dice(10, 10) + level
			victim.Hit += heal
			if victim.Hit > victim.MaxHit {
				victim.Hit = victim.MaxHit
			}
		}
	}

	return true
}

// Unholy Word - Cleric ultimate area spell
func spellUnholyWord(caster *types.Character, level int, target interface{}) bool {
	// Damage good creatures, heal evil ones
	for _, victim := range caster.InRoom.People {
		if victim == caster {
			continue
		}

		if victim.Alignment > 0 { // Good
			dam := combat.Dice(20, 10) + level*2
			victim.Hit -= dam
		} else if victim.Alignment < 0 { // Evil
			heal := combat.Dice(10, 10) + level
			victim.Hit += heal
			if victim.Hit > victim.MaxHit {
				victim.Hit = victim.MaxHit
			}
		}
	}

	return true
}

// Control Weather - Cleric utility spell
// Changes the weather to be better or worse
// Note: Full implementation requires weather system
func spellControlWeather(caster *types.Character, level int, target interface{}) bool {
	// This spell requires argument "better" or "worse" via targetArg
	// For now, just succeed - actual weather changes would need weather system
	return true
}

// Fireproof - Mage utility spell
// Protects an object from fire damage
func spellFireproof(caster *types.Character, level int, target interface{}) bool {
	obj, ok := target.(*types.Object)
	if !ok || obj == nil {
		return false
	}

	// Already fireproof
	if obj.ExtraFlags.Has(types.ItemBurnProof) {
		return false
	}

	// Add burn proof flag via object affect
	obj.ExtraFlags.Set(types.ItemBurnProof)

	return true
}

// Heat Metal - Cleric damage spell
// Heats metal armor/weapons, forcing victims to drop them or take damage
func spellHeatMetal(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		return false
	}

	// Check fire immunity
	if victim.Imm.Has(types.ImmFire) {
		return false
	}

	// Save negates
	if combat.NumberPercent() < 50+victim.Level-level {
		return false
	}

	dam := 0

	// Check each equipped item
	for i := types.WearLocation(0); i < types.WearLocMax; i++ {
		obj := victim.GetEquipment(i)
		if obj == nil {
			continue
		}

		// Skip non-metal items and fireproof items
		if obj.ExtraFlags.Has(types.ItemNonMetal) || obj.ExtraFlags.Has(types.ItemBurnProof) {
			continue
		}

		// Only affects armor and weapons
		if obj.ItemType != types.ItemTypeArmor && obj.ItemType != types.ItemTypeWeapon {
			continue
		}

		// Random chance based on level
		if combat.NumberRange(1, 2*level) <= obj.Level {
			continue
		}

		// Item heats up - damage based on item level
		dam += combat.NumberRange(1, obj.Level)
	}

	// Also check inventory for metal items
	for _, obj := range victim.Inventory {
		if obj.ExtraFlags.Has(types.ItemNonMetal) || obj.ExtraFlags.Has(types.ItemBurnProof) {
			continue
		}

		if obj.ItemType != types.ItemTypeArmor && obj.ItemType != types.ItemTypeWeapon {
			continue
		}

		if combat.NumberRange(1, 2*level) <= obj.Level {
			continue
		}

		dam += combat.NumberRange(1, obj.Level) / 6
	}

	if dam > 0 {
		victim.Hit -= dam
	}

	return dam > 0
}

// === Mana Restoration Spell ===

// spellRestoreMana - Healer spell that restores 51 mana to target
// Typically used by healer NPCs to restore mana
func spellRestoreMana(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		return false
	}

	// Restore exactly 51 mana (as per original ROT)
	victim.Mana = min(victim.Mana+51, victim.MaxMana)
	return true
}

// === Robot NPC Spells ===

// spellGeneralPurpose - Robot NPC attack spell (from mega1.are)
// Does 25-100 pierce damage
func spellGeneralPurpose(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		return false
	}

	dam := combat.NumberRange(25, 100)

	// Save for half damage
	if combat.NumberPercent() < 50+victim.Level-level {
		dam /= 2
	}

	// Pierce damage
	dam = checkDamageResist(victim, dam, types.DamPierce)

	victim.Hit -= dam
	return true
}

// spellHighExplosive - Robot NPC attack spell (from mega1.are)
// Does 30-120 pierce damage
func spellHighExplosive(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		return false
	}

	dam := combat.NumberRange(30, 120)

	// Save for half damage
	if combat.NumberPercent() < 50+victim.Level-level {
		dam /= 2
	}

	// Pierce damage
	dam = checkDamageResist(victim, dam, types.DamPierce)

	victim.Hit -= dam
	return true
}

// === Protection Spells ===

// spellProtectionVoodoo - Protects against voodoo dolls
// Duration: level ticks
func spellProtectionVoodoo(caster *types.Character, level int, target interface{}) bool {
	victim, ok := target.(*types.Character)
	if !ok || victim == nil {
		return false
	}

	// Already protected
	if victim.IsShielded(types.ShdProtectVoodoo) {
		return false
	}

	// Add voodoo protection affect
	affect := &types.Affect{
		Type:         "protection voodoo",
		Level:        level,
		Duration:     level,
		Location:     types.ApplyNone,
		Modifier:     0,
		ShieldVector: types.ShdProtectVoodoo,
	}
	victim.Affected.Add(affect)
	victim.ShieldedBy.Set(types.ShdProtectVoodoo)

	return true
}

// === Necromancy Spells ===

// spellAnimate - Animates a body part into a floating follower
// Body parts are objects with VNUMs 12-17 (brains, guts, hearts, etc.)
// The spell validates the target; actual pet creation is handled by MagicSystem
func spellAnimate(caster *types.Character, level int, target interface{}) bool {
	obj, ok := target.(*types.Object)
	if !ok || obj == nil {
		return false
	}

	// Check if it's a valid body part (VNUM 12-17)
	if obj.Vnum < 12 || obj.Vnum > 17 {
		return false
	}

	// Check if caster already has a pet
	if caster.Pet != nil {
		return false
	}

	// Spell validates target - actual creation is handled by MagicSystem.handleAnimate
	return true
}

// spellResurrect - Raises a corpse as a zombie follower
// This spell targets a corpse in the room (TargetIgnore)
// The spell validates conditions; actual pet creation is handled by MagicSystem
func spellResurrect(caster *types.Character, level int, target interface{}) bool {
	// Check if caster already has a pet
	if caster.Pet != nil {
		return false
	}

	// Spell validates conditions - corpse finding and creation handled by MagicSystem.handleResurrect
	return true
}

// === Item Creation Spells ===

// spellEmpower - Creates a potion or scroll containing a spell
// Defensive spells become potions, offensive spells become scrolls
// Note: Full implementation requires spell argument parsing
func spellEmpower(caster *types.Character, level int, target interface{}) bool {
	// This spell requires a spell name argument (TargetIgnore)
	// Full implementation would:
	// 1. Parse spell name from target_name argument
	// 2. Verify caster knows the spell and can cast it
	// 3. Check if spell is "empower" itself (fail if so)
	// 4. Calculate additional mana cost for the empowered spell
	// 5. Create potion (defensive spells) or scroll (offensive spells)
	// 6. Set object values to contain the spell slot number
	// 7. Give the item to the caster

	// This requires access to spell registry and object creation
	// For now, return false - requires additional systems
	return false
}

// Body part VNUMs from the C source (OBJ_VNUM_SEVERED_HEAD through OBJ_VNUM_BRAINS)
const (
	VnumSeveredHead = 12
	VnumTornHeart   = 13
	VnumSlicedArm   = 14
	VnumSlicedLeg   = 15
	VnumGuts        = 16
	VnumBrains      = 17
	VnumVoodooDoll  = 51
)

// spellVoodoo - Creates a voodoo doll from a body part
// The caster must be holding a body part from a player (not NPC)
// The resulting voodoo doll can be used to remotely affect the player whose body part it was
// Ported from spell_voodoo in magic2.c
func spellVoodoo(caster *types.Character, level int, target interface{}) bool {
	// Check what caster is holding
	heldItem := caster.GetEquipment(types.WearLocHold)
	if heldItem == nil {
		return false
	}

	// Check if it's a valid body part (VNUM 12-17)
	if heldItem.Vnum < VnumSeveredHead || heldItem.Vnum > VnumBrains {
		return false
	}

	// Check if the body part is from a player (not an NPC)
	// In the C code, value[4] is set to 1 for player body parts
	if heldItem.Values[4] == 0 {
		return false
	}

	// The spell validates the target body part is valid
	// Actual voodoo doll creation is handled by MagicSystem.handleVoodoo
	// which will:
	// 1. Create a voodoo doll object (VNUM 51)
	// 2. Set the doll's name/keywords to include the victim's name
	// 3. Remove the body part from the caster
	// 4. Give the doll to the caster and equip it
	return true
}
