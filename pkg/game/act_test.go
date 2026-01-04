package game

import (
	"testing"

	"rotmud/pkg/types"
)

func TestActFormat(t *testing.T) {
	ch := types.NewCharacter("Gandalf")
	ch.Sex = types.SexMale

	victim := types.NewCharacter("Saruman")
	victim.Sex = types.SexMale

	obj := types.NewObject(3042, "a staff", types.ItemTypeWeapon)

	t.Run("$n substitutes character name", func(t *testing.T) {
		result := ActFormat("$n waves.", ch, nil, nil)
		if result != "Gandalf waves." {
			t.Errorf("expected 'Gandalf waves.', got '%s'", result)
		}
	})

	t.Run("$N substitutes victim name", func(t *testing.T) {
		result := ActFormat("$n attacks $N.", ch, victim, nil)
		if result != "Gandalf attacks Saruman." {
			t.Errorf("expected 'Gandalf attacks Saruman.', got '%s'", result)
		}
	})

	t.Run("$e substitutes he/she/it for character", func(t *testing.T) {
		result := ActFormat("$e grins.", ch, nil, nil)
		if result != "he grins." {
			t.Errorf("expected 'he grins.', got '%s'", result)
		}

		ch.Sex = types.SexFemale
		result = ActFormat("$e grins.", ch, nil, nil)
		if result != "she grins." {
			t.Errorf("expected 'she grins.', got '%s'", result)
		}
		ch.Sex = types.SexMale
	})

	t.Run("$E substitutes he/she/it for victim", func(t *testing.T) {
		result := ActFormat("$n looks at $N as $E laughs.", ch, victim, nil)
		if result != "Gandalf looks at Saruman as he laughs." {
			t.Errorf("expected 'Gandalf looks at Saruman as he laughs.', got '%s'", result)
		}
	})

	t.Run("$m substitutes him/her/it for character", func(t *testing.T) {
		result := ActFormat("You give $m a cookie.", ch, nil, nil)
		if result != "You give him a cookie." {
			t.Errorf("expected 'You give him a cookie.', got '%s'", result)
		}
	})

	t.Run("$M substitutes him/her/it for victim", func(t *testing.T) {
		result := ActFormat("$n gives $M a cookie.", ch, victim, nil)
		if result != "Gandalf gives him a cookie." {
			t.Errorf("expected 'Gandalf gives him a cookie.', got '%s'", result)
		}
	})

	t.Run("$s substitutes his/her/its for character", func(t *testing.T) {
		result := ActFormat("$n waves $s hand.", ch, nil, nil)
		if result != "Gandalf waves his hand." {
			t.Errorf("expected 'Gandalf waves his hand.', got '%s'", result)
		}
	})

	t.Run("$S substitutes his/her/its for victim", func(t *testing.T) {
		result := ActFormat("$n takes $S weapon.", ch, victim, nil)
		if result != "Gandalf takes his weapon." {
			t.Errorf("expected 'Gandalf takes his weapon.', got '%s'", result)
		}
	})

	t.Run("$p substitutes object short desc", func(t *testing.T) {
		result := ActFormat("$n picks up $p.", ch, nil, obj)
		if result != "Gandalf picks up a staff." {
			t.Errorf("expected 'Gandalf picks up a staff.', got '%s'", result)
		}
	})

	t.Run("Multiple substitutions work together", func(t *testing.T) {
		result := ActFormat("$n gives $p to $N.", ch, victim, obj)
		if result != "Gandalf gives a staff to Saruman." {
			t.Errorf("expected 'Gandalf gives a staff to Saruman.', got '%s'", result)
		}
	})

	t.Run("Unknown tokens pass through", func(t *testing.T) {
		result := ActFormat("$x unknown $y token", ch, nil, nil)
		if result != "$x unknown $y token" {
			t.Errorf("expected '$x unknown $y token', got '%s'", result)
		}
	})
}

func TestActToRoom(t *testing.T) {
	t.Run("ActToRoom sends to all in room except actor", func(t *testing.T) {
		room := types.NewRoom(3001, "Test", "A test room")

		actor := types.NewCharacter("Actor")
		actor.InRoom = room
		room.AddPerson(actor)

		bystander1 := types.NewCharacter("Bystander1")
		bystander1.InRoom = room
		room.AddPerson(bystander1)

		bystander2 := types.NewCharacter("Bystander2")
		bystander2.InRoom = room
		room.AddPerson(bystander2)

		messages := make(map[string]string)
		outputFunc := func(ch *types.Character, msg string) {
			messages[ch.Name] = msg
		}

		ActToRoom("$n waves.", actor, nil, nil, outputFunc)

		// Actor should NOT receive message
		if _, ok := messages["Actor"]; ok {
			t.Error("actor should not receive ActToRoom message")
		}

		// Bystanders should receive message
		if messages["Bystander1"] != "Actor waves.\r\n" {
			t.Errorf("expected 'Actor waves.\\r\\n', got '%s'", messages["Bystander1"])
		}
		if messages["Bystander2"] != "Actor waves.\r\n" {
			t.Errorf("expected 'Actor waves.\\r\\n', got '%s'", messages["Bystander2"])
		}
	})
}

func TestActToChar(t *testing.T) {
	t.Run("ActToChar sends only to actor", func(t *testing.T) {
		actor := types.NewCharacter("Actor")

		var received string
		outputFunc := func(ch *types.Character, msg string) {
			received = msg
		}

		ActToChar("You wave.", actor, nil, nil, outputFunc)

		if received != "You wave.\r\n" {
			t.Errorf("expected 'You wave.\\r\\n', got '%s'", received)
		}
	})
}

func TestActToVict(t *testing.T) {
	t.Run("ActToVict sends only to victim", func(t *testing.T) {
		actor := types.NewCharacter("Actor")
		victim := types.NewCharacter("Victim")

		messages := make(map[string]string)
		outputFunc := func(ch *types.Character, msg string) {
			messages[ch.Name] = msg
		}

		ActToVict("$n waves at you.", actor, victim, nil, outputFunc)

		if _, ok := messages["Actor"]; ok {
			t.Error("actor should not receive ActToVict message")
		}
		if messages["Victim"] != "Actor waves at you.\r\n" {
			t.Errorf("expected 'Actor waves at you.\\r\\n', got '%s'", messages["Victim"])
		}
	})
}
