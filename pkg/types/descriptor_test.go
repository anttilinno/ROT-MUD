package types

import "testing"

func TestDescriptor(t *testing.T) {
	t.Run("NewDescriptor creates descriptor with correct state", func(t *testing.T) {
		desc := NewDescriptor("127.0.0.1")
		if desc.Host != "127.0.0.1" {
			t.Errorf("expected host '127.0.0.1', got '%s'", desc.Host)
		}
		if desc.State != ConGetName {
			t.Errorf("expected state ConGetName, got %v", desc.State)
		}
	})

	t.Run("Descriptor state transitions", func(t *testing.T) {
		desc := NewDescriptor("127.0.0.1")

		desc.State = ConGetOldPassword
		if desc.State != ConGetOldPassword {
			t.Error("expected state to be ConGetOldPassword")
		}

		desc.State = ConPlaying
		if !desc.IsPlaying() {
			t.Error("expected IsPlaying to be true when state is ConPlaying")
		}
	})

	t.Run("Descriptor can be associated with character", func(t *testing.T) {
		desc := NewDescriptor("127.0.0.1")
		ch := NewCharacter("TestPlayer")

		desc.Character = ch
		ch.Descriptor = desc

		if desc.Character != ch {
			t.Error("expected descriptor's character to be the player")
		}
		if ch.Descriptor != desc {
			t.Error("expected character's descriptor to be the descriptor")
		}
	})

	t.Run("Descriptor snoop functionality", func(t *testing.T) {
		desc := NewDescriptor("127.0.0.1")
		snooper := NewDescriptor("192.168.1.1")

		desc.SnoopedBy = snooper
		if !desc.IsSnooped() {
			t.Error("expected IsSnooped to be true when SnoopedBy is set")
		}
	})

	t.Run("Descriptor command repeat tracking", func(t *testing.T) {
		desc := NewDescriptor("127.0.0.1")

		desc.LastCommand = "north"
		desc.RepeatCount = 1

		if desc.LastCommand != "north" {
			t.Errorf("expected last command 'north', got '%s'", desc.LastCommand)
		}
	})
}

func TestDescriptorEditorState(t *testing.T) {
	t.Run("Descriptor OLC editor state", func(t *testing.T) {
		desc := NewDescriptor("127.0.0.1")

		desc.Editor = EditorNone
		if desc.InEditor() {
			t.Error("expected InEditor to be false when editor is EditorNone")
		}

		desc.Editor = EditorRoom
		if !desc.InEditor() {
			t.Error("expected InEditor to be true when in room editor")
		}
	})
}
