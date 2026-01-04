package types

import "testing"

func TestActFlags(t *testing.T) {
	t.Run("Has returns true for set flags", func(t *testing.T) {
		flags := ActNPC | ActSentinel
		if !flags.Has(ActNPC) {
			t.Error("expected ActNPC to be set")
		}
		if !flags.Has(ActSentinel) {
			t.Error("expected ActSentinel to be set")
		}
	})

	t.Run("Has returns false for unset flags", func(t *testing.T) {
		flags := ActNPC
		if flags.Has(ActSentinel) {
			t.Error("expected ActSentinel to not be set")
		}
	})

	t.Run("Set adds a flag", func(t *testing.T) {
		var flags ActFlags
		flags.Set(ActNPC)
		if !flags.Has(ActNPC) {
			t.Error("expected ActNPC to be set after Set()")
		}
	})

	t.Run("Remove clears a flag", func(t *testing.T) {
		flags := ActNPC | ActSentinel
		flags.Remove(ActNPC)
		if flags.Has(ActNPC) {
			t.Error("expected ActNPC to be cleared after Remove()")
		}
		if !flags.Has(ActSentinel) {
			t.Error("expected ActSentinel to still be set")
		}
	})

	t.Run("Toggle flips a flag", func(t *testing.T) {
		flags := ActNPC
		flags.Toggle(ActNPC)
		if flags.Has(ActNPC) {
			t.Error("expected ActNPC to be cleared after Toggle()")
		}
		flags.Toggle(ActNPC)
		if !flags.Has(ActNPC) {
			t.Error("expected ActNPC to be set after second Toggle()")
		}
	})
}

func TestAffectFlags(t *testing.T) {
	t.Run("Has returns true for set flags", func(t *testing.T) {
		flags := AffBlind | AffInvisible
		if !flags.Has(AffBlind) {
			t.Error("expected AffBlind to be set")
		}
		if !flags.Has(AffInvisible) {
			t.Error("expected AffInvisible to be set")
		}
	})

	t.Run("Set adds a flag", func(t *testing.T) {
		var flags AffectFlags
		flags.Set(AffHaste)
		if !flags.Has(AffHaste) {
			t.Error("expected AffHaste to be set after Set()")
		}
	})

	t.Run("Multiple flags can be combined", func(t *testing.T) {
		flags := AffBlind | AffPoison | AffSanctuary
		if !flags.Has(AffBlind) || !flags.Has(AffPoison) || !flags.Has(AffSanctuary) {
			t.Error("expected all three flags to be set")
		}
	})
}

func TestRoomFlags(t *testing.T) {
	t.Run("Room flags work correctly", func(t *testing.T) {
		flags := RoomDark | RoomSafe
		if !flags.Has(RoomDark) {
			t.Error("expected RoomDark to be set")
		}
		if !flags.Has(RoomSafe) {
			t.Error("expected RoomSafe to be set")
		}
		if flags.Has(RoomPrivate) {
			t.Error("expected RoomPrivate to not be set")
		}
	})
}

func TestItemFlags(t *testing.T) {
	t.Run("Item extra flags work correctly", func(t *testing.T) {
		flags := ItemGlow | ItemMagic | ItemNoDrop
		if !flags.Has(ItemGlow) {
			t.Error("expected ItemGlow to be set")
		}
		if !flags.Has(ItemMagic) {
			t.Error("expected ItemMagic to be set")
		}
		if !flags.Has(ItemNoDrop) {
			t.Error("expected ItemNoDrop to be set")
		}
	})
}

func TestWearFlags(t *testing.T) {
	t.Run("Wear flags work correctly", func(t *testing.T) {
		flags := WearTake | WearWield | WearHold
		if !flags.Has(WearTake) {
			t.Error("expected WearTake to be set")
		}
		if !flags.Has(WearWield) {
			t.Error("expected WearWield to be set")
		}
	})
}
