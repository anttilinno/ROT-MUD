package combat

import (
	"testing"

	"rotmud/pkg/types"
)

func TestDice(t *testing.T) {
	t.Run("Dice returns value in range", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			result := Dice(2, 6)
			if result < 2 || result > 12 {
				t.Errorf("Dice(2,6) = %d, expected 2-12", result)
			}
		}
	})

	t.Run("Dice with invalid input returns 0", func(t *testing.T) {
		if Dice(0, 6) != 0 {
			t.Error("Dice(0,6) should return 0")
		}
		if Dice(2, 0) != 0 {
			t.Error("Dice(2,0) should return 0")
		}
	})
}

func TestNumberRange(t *testing.T) {
	t.Run("NumberRange returns value in range", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			result := NumberRange(5, 10)
			if result < 5 || result > 10 {
				t.Errorf("NumberRange(5,10) = %d, expected 5-10", result)
			}
		}
	})

	t.Run("NumberRange with equal values returns that value", func(t *testing.T) {
		result := NumberRange(5, 5)
		if result != 5 {
			t.Errorf("NumberRange(5,5) = %d, expected 5", result)
		}
	})
}

func TestNumberPercent(t *testing.T) {
	t.Run("NumberPercent returns value 1-100", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			result := NumberPercent()
			if result < 1 || result > 100 {
				t.Errorf("NumberPercent() = %d, expected 1-100", result)
			}
		}
	})
}

func TestInterpolate(t *testing.T) {
	t.Run("Interpolate at level 0", func(t *testing.T) {
		result := Interpolate(0, 20, -10)
		if result != 20 {
			t.Errorf("Interpolate(0, 20, -10) = %d, expected 20", result)
		}
	})

	t.Run("Interpolate at level 32", func(t *testing.T) {
		result := Interpolate(32, 20, -10)
		if result != -10 {
			t.Errorf("Interpolate(32, 20, -10) = %d, expected -10", result)
		}
	})

	t.Run("Interpolate at level 16", func(t *testing.T) {
		result := Interpolate(16, 20, -10)
		if result != 5 {
			t.Errorf("Interpolate(16, 20, -10) = %d, expected 5", result)
		}
	})
}

func TestSetFighting(t *testing.T) {
	t.Run("SetFighting sets fighting target", func(t *testing.T) {
		ch := types.NewCharacter("Attacker")
		victim := types.NewCharacter("Victim")

		SetFighting(ch, victim)

		if ch.Fighting != victim {
			t.Error("Expected ch.Fighting to be victim")
		}
		if ch.Position != types.PosFighting {
			t.Error("Expected ch.Position to be PosFighting")
		}
	})

	t.Run("SetFighting does nothing if already fighting", func(t *testing.T) {
		ch := types.NewCharacter("Attacker")
		victim1 := types.NewCharacter("Victim1")
		victim2 := types.NewCharacter("Victim2")

		SetFighting(ch, victim1)
		SetFighting(ch, victim2)

		if ch.Fighting != victim1 {
			t.Error("Expected ch.Fighting to still be victim1")
		}
	})
}

func TestStopFighting(t *testing.T) {
	t.Run("StopFighting clears fighting", func(t *testing.T) {
		ch := types.NewCharacter("Attacker")
		victim := types.NewCharacter("Victim")
		ch.Position = types.PosFighting

		SetFighting(ch, victim)
		StopFighting(ch, false)

		if ch.Fighting != nil {
			t.Error("Expected ch.Fighting to be nil")
		}
		if ch.Position != types.PosStanding {
			t.Error("Expected ch.Position to be PosStanding")
		}
	})

	t.Run("StopFighting with allInRoom clears others", func(t *testing.T) {
		room := types.NewRoom(1, "Test", "Test room")
		ch := types.NewCharacter("Attacker")
		victim := types.NewCharacter("Victim")

		ch.InRoom = room
		victim.InRoom = room
		room.AddPerson(ch)
		room.AddPerson(victim)

		SetFighting(ch, victim)
		SetFighting(victim, ch)

		StopFighting(ch, true)

		if victim.Fighting != nil {
			t.Error("Expected victim.Fighting to be nil")
		}
	})
}

func TestIsSafe(t *testing.T) {
	t.Run("Cannot attack yourself", func(t *testing.T) {
		ch := types.NewCharacter("Test")
		if !IsSafe(ch, ch) {
			t.Error("Expected attacking self to be safe")
		}
	})

	t.Run("Safe rooms prevent combat", func(t *testing.T) {
		room := types.NewRoom(1, "Safe Room", "A safe room")
		room.Flags.Set(types.RoomSafe)

		ch := types.NewCharacter("Attacker")
		victim := types.NewCharacter("Victim")
		ch.InRoom = room

		if !IsSafe(ch, victim) {
			t.Error("Expected combat to be prevented in safe room")
		}
	})

	t.Run("Immortals are safe", func(t *testing.T) {
		ch := types.NewCharacter("Attacker")
		victim := types.NewCharacter("Immortal")
		victim.Level = types.LevelImmortal

		if !IsSafe(ch, victim) {
			t.Error("Expected immortal to be safe")
		}
	})
}

func TestGetThac0(t *testing.T) {
	t.Run("THAC0 decreases with level", func(t *testing.T) {
		ch1 := types.NewCharacter("Level1")
		ch1.Level = 1
		ch2 := types.NewCharacter("Level20")
		ch2.Level = 20

		thac0_1 := GetThac0(ch1)
		thac0_20 := GetThac0(ch2)

		if thac0_1 <= thac0_20 {
			t.Errorf("Expected THAC0 to decrease with level: level 1 = %d, level 20 = %d", thac0_1, thac0_20)
		}
	})

	t.Run("NPC warrior has better THAC0", func(t *testing.T) {
		warrior := types.NewNPC(1, "Warrior", 20)
		warrior.Act.Set(types.ActWarrior)

		mage := types.NewNPC(2, "Mage", 20)
		mage.Act.Set(types.ActMage)

		thac0_warrior := GetThac0(warrior)
		thac0_mage := GetThac0(mage)

		if thac0_warrior >= thac0_mage {
			t.Errorf("Expected warrior to have better THAC0: warrior = %d, mage = %d", thac0_warrior, thac0_mage)
		}
	})
}

func TestUpdatePosition(t *testing.T) {
	t.Run("Positive HP sets standing", func(t *testing.T) {
		ch := types.NewCharacter("Test")
		ch.Hit = 10
		ch.Position = types.PosStunned

		UpdatePosition(ch)

		if ch.Position != types.PosStanding {
			t.Errorf("Expected PosStanding, got %v", ch.Position)
		}
	})

	t.Run("Zero HP sets stunned for player", func(t *testing.T) {
		ch := types.NewCharacter("Test")
		ch.Hit = 0

		UpdatePosition(ch)

		if ch.Position != types.PosStunned {
			t.Errorf("Expected PosStunned, got %v", ch.Position)
		}
	})

	t.Run("Negative HP sets incap", func(t *testing.T) {
		ch := types.NewCharacter("Test")
		ch.Hit = -4

		UpdatePosition(ch)

		if ch.Position != types.PosIncap {
			t.Errorf("Expected PosIncap, got %v", ch.Position)
		}
	})

	t.Run("Very negative HP sets mortal", func(t *testing.T) {
		ch := types.NewCharacter("Test")
		ch.Hit = -8

		UpdatePosition(ch)

		if ch.Position != types.PosMortal {
			t.Errorf("Expected PosMortal, got %v", ch.Position)
		}
	})

	t.Run("Very very negative HP sets dead", func(t *testing.T) {
		ch := types.NewCharacter("Test")
		ch.Hit = -15

		UpdatePosition(ch)

		if ch.Position != types.PosDead {
			t.Errorf("Expected PosDead, got %v", ch.Position)
		}
	})

	t.Run("NPC dies at 0 HP", func(t *testing.T) {
		ch := types.NewNPC(1, "mob", 5)
		ch.Hit = -11

		UpdatePosition(ch)

		if ch.Position != types.PosDead {
			t.Errorf("Expected PosDead for NPC, got %v", ch.Position)
		}
	})
}

func TestCheckImmune(t *testing.T) {
	t.Run("Normal character has no immunity", func(t *testing.T) {
		ch := types.NewCharacter("Test")

		result := CheckImmune(ch, types.DamFire)

		if result != ImmNormal {
			t.Errorf("Expected ImmNormal, got %v", result)
		}
	})

	t.Run("Immune character returns ImmImmune", func(t *testing.T) {
		ch := types.NewCharacter("Test")
		ch.Imm.Set(types.ImmFire)

		result := CheckImmune(ch, types.DamFire)

		if result != ImmImmune {
			t.Errorf("Expected ImmImmune, got %v", result)
		}
	})

	t.Run("Resistant character returns ImmResistant", func(t *testing.T) {
		ch := types.NewCharacter("Test")
		ch.Res.Set(types.ImmCold)

		result := CheckImmune(ch, types.DamCold)

		if result != ImmResistant {
			t.Errorf("Expected ImmResistant, got %v", result)
		}
	})

	t.Run("Vulnerable character returns ImmVulnerable", func(t *testing.T) {
		ch := types.NewCharacter("Test")
		ch.Vuln.Set(types.ImmFire)

		result := CheckImmune(ch, types.DamFire)

		if result != ImmVulnerable {
			t.Errorf("Expected ImmVulnerable, got %v", result)
		}
	})
}

func TestDefenses(t *testing.T) {
	t.Run("Parry requires weapon", func(t *testing.T) {
		cs := NewCombatSystem()
		cs.Output = func(ch *types.Character, msg string) {}

		ch := types.NewCharacter("Attacker")
		victim := types.NewCharacter("Defender")
		victim.Level = 50 // High level for high parry chance

		room := types.NewRoom(1, "Test", "Test")
		ch.InRoom = room
		victim.InRoom = room
		victim.Position = types.PosFighting

		// Without weapon, should not parry
		parried := false
		for i := 0; i < 100; i++ {
			result := cs.CheckDefenses(ch, victim)
			if result == DefenseParried {
				parried = true
				break
			}
		}
		// Shouldn't parry without weapon
		if parried {
			t.Error("Should not parry without a weapon")
		}
	})

	t.Run("Shield block requires shield", func(t *testing.T) {
		cs := NewCombatSystem()
		cs.Output = func(ch *types.Character, msg string) {}

		ch := types.NewCharacter("Attacker")
		victim := types.NewCharacter("Defender")
		victim.Level = 50
		victim.Position = types.PosFighting

		room := types.NewRoom(1, "Test", "Test")
		ch.InRoom = room
		victim.InRoom = room

		// Without shield, should not block (though may dodge)
		blocked := false
		for i := 0; i < 100; i++ {
			result := cs.CheckDefenses(ch, victim)
			if result == DefenseBlocked {
				blocked = true
				break
			}
		}

		if blocked {
			t.Error("Should not block without a shield")
		}
	})

	t.Run("Cannot defend when not fighting position", func(t *testing.T) {
		cs := NewCombatSystem()
		cs.Output = func(ch *types.Character, msg string) {}

		ch := types.NewCharacter("Attacker")
		victim := types.NewCharacter("Defender")
		victim.Level = 50
		victim.Position = types.PosSleeping // Can't defend while sleeping

		result := cs.CheckDefenses(ch, victim)

		if result != DefenseNone {
			t.Error("Should not be able to defend while sleeping")
		}
	})
}

func TestCombatSystemDamage(t *testing.T) {
	t.Run("Damage reduces victim HP", func(t *testing.T) {
		cs := NewCombatSystem()
		outputs := make(map[string]string)
		cs.Output = func(ch *types.Character, msg string) {
			outputs[ch.Name] += msg
		}

		ch := types.NewCharacter("Attacker")
		victim := types.NewCharacter("Victim")
		victim.Hit = 100
		victim.MaxHit = 100
		victim.Position = types.PosStunned // Can't defend when stunned

		room := types.NewRoom(1, "Test", "Test room")
		ch.InRoom = room
		victim.InRoom = room
		room.AddPerson(ch)
		room.AddPerson(victim)

		result := cs.Damage(ch, victim, 20, types.DamSlash, true)

		if victim.Hit != 80 {
			t.Errorf("Expected victim HP to be 80, got %d", victim.Hit)
		}
		if result.Damage != 20 {
			t.Errorf("Expected result.Damage to be 20, got %d", result.Damage)
		}
		if result.Killed {
			t.Error("Victim should not be killed")
		}
	})

	t.Run("Lethal damage kills victim", func(t *testing.T) {
		cs := NewCombatSystem()
		outputs := make(map[string]string)
		cs.Output = func(ch *types.Character, msg string) {
			outputs[ch.Name] += msg
		}

		ch := types.NewCharacter("Attacker")
		victim := types.NewNPC(1, "mob", 5)
		victim.Hit = 10
		victim.MaxHit = 10
		victim.Level = 1                   // Low level so defenses rarely trigger
		victim.Position = types.PosStunned // Can't defend when stunned

		room := types.NewRoom(1, "Test", "Test room")
		ch.InRoom = room
		victim.InRoom = room
		room.AddPerson(ch)
		room.AddPerson(victim)

		result := cs.Damage(ch, victim, 100, types.DamSlash, true)

		if !result.Killed {
			t.Error("Victim should be killed")
		}
		if victim.Position != types.PosDead {
			t.Errorf("Expected victim position to be PosDead, got %v", victim.Position)
		}
	})

	t.Run("Sanctuary halves damage", func(t *testing.T) {
		cs := NewCombatSystem()
		cs.Output = func(ch *types.Character, msg string) {}
		// Set skill getter to return 0 to prevent random defense checks from succeeding
		cs.SkillGetter = func(ch *types.Character, skillName string) int { return 0 }

		ch := types.NewCharacter("Attacker")
		victim := types.NewCharacter("Victim")
		victim.Hit = 100
		victim.MaxHit = 100
		victim.AffectedBy.Set(types.AffSanctuary)

		room := types.NewRoom(1, "Test", "Test room")
		ch.InRoom = room
		victim.InRoom = room
		room.AddPerson(ch)
		room.AddPerson(victim)

		result := cs.Damage(ch, victim, 20, types.DamSlash, true)

		// 20 / 2 = 10 damage with sanctuary
		if result.Damage != 10 {
			t.Errorf("Expected 10 damage with sanctuary, got %d", result.Damage)
		}
	})

	t.Run("Immune target takes no damage", func(t *testing.T) {
		cs := NewCombatSystem()
		cs.Output = func(ch *types.Character, msg string) {}

		ch := types.NewCharacter("Attacker")
		victim := types.NewCharacter("Victim")
		victim.Hit = 100
		victim.MaxHit = 100
		victim.Imm.Set(types.ImmFire)

		room := types.NewRoom(1, "Test", "Test room")
		ch.InRoom = room
		victim.InRoom = room
		room.AddPerson(ch)
		room.AddPerson(victim)

		result := cs.Damage(ch, victim, 20, types.DamFire, true)

		if result.Damage != 0 {
			t.Errorf("Expected 0 damage from immunity, got %d", result.Damage)
		}
		if !result.Immune {
			t.Error("Expected result.Immune to be true")
		}
		if victim.Hit != 100 {
			t.Errorf("Expected victim HP unchanged at 100, got %d", victim.Hit)
		}
	})
}
