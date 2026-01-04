package game

import (
	"testing"
	"time"

	"rotmud/pkg/types"
)

func TestCommand(t *testing.T) {
	t.Run("Command stores character and input", func(t *testing.T) {
		ch := types.NewCharacter("Test")
		cmd := Command{
			Character: ch,
			Input:     "look",
		}

		if cmd.Character != ch {
			t.Error("expected command character to be set")
		}
		if cmd.Input != "look" {
			t.Errorf("expected input 'look', got '%s'", cmd.Input)
		}
	})
}

func TestGameLoop(t *testing.T) {
	t.Run("NewGameLoop creates loop with correct settings", func(t *testing.T) {
		loop := NewGameLoop()

		if loop.PulseRate != 250*time.Millisecond {
			t.Errorf("expected pulse rate 250ms, got %v", loop.PulseRate)
		}
		if loop.commands == nil {
			t.Error("expected commands channel to be initialized")
		}
	})

	t.Run("GameLoop tracks pulse count", func(t *testing.T) {
		loop := NewGameLoop()

		// Simulate pulses
		loop.pulse()
		loop.pulse()
		loop.pulse()

		if loop.PulseCount != 3 {
			t.Errorf("expected pulse count 3, got %d", loop.PulseCount)
		}
	})

	t.Run("Violence update triggers every 3 pulses", func(t *testing.T) {
		loop := NewGameLoop()
		violenceCount := 0
		loop.OnViolence = func() { violenceCount++ }

		// 12 pulses = 4 violence updates
		for i := 0; i < 12; i++ {
			loop.pulse()
		}

		if violenceCount != 4 {
			t.Errorf("expected 4 violence updates, got %d", violenceCount)
		}
	})

	t.Run("Mobile update triggers every 4 pulses", func(t *testing.T) {
		loop := NewGameLoop()
		mobileCount := 0
		loop.OnMobile = func() { mobileCount++ }

		// 12 pulses = 3 mobile updates
		for i := 0; i < 12; i++ {
			loop.pulse()
		}

		if mobileCount != 3 {
			t.Errorf("expected 3 mobile updates, got %d", mobileCount)
		}
	})

	t.Run("Character tick triggers every 60 pulses", func(t *testing.T) {
		loop := NewGameLoop()
		tickCount := 0
		loop.OnTick = func() { tickCount++ }

		// 120 pulses = 2 ticks
		for i := 0; i < 120; i++ {
			loop.pulse()
		}

		if tickCount != 2 {
			t.Errorf("expected 2 ticks, got %d", tickCount)
		}
	})

	t.Run("Area reset triggers every 120 pulses", func(t *testing.T) {
		loop := NewGameLoop()
		resetCount := 0
		loop.OnAreaReset = func() { resetCount++ }

		// 240 pulses = 2 area resets
		for i := 0; i < 240; i++ {
			loop.pulse()
		}

		if resetCount != 2 {
			t.Errorf("expected 2 area resets, got %d", resetCount)
		}
	})

	t.Run("QueueCommand adds command to channel", func(t *testing.T) {
		loop := NewGameLoop()
		ch := types.NewCharacter("Test")

		go func() {
			loop.QueueCommand(ch, "look")
		}()

		select {
		case cmd := <-loop.commands:
			if cmd.Input != "look" {
				t.Errorf("expected input 'look', got '%s'", cmd.Input)
			}
		case <-time.After(100 * time.Millisecond):
			t.Error("timeout waiting for command")
		}
	})
}

func TestGameLoopStartStop(t *testing.T) {
	t.Run("GameLoop can be started and stopped", func(t *testing.T) {
		loop := NewGameLoop()
		loop.PulseRate = 10 * time.Millisecond // Fast for testing

		pulseCount := 0
		loop.OnPulse = func() { pulseCount++ }

		loop.Start()
		time.Sleep(50 * time.Millisecond)
		loop.Stop()

		// Should have processed some pulses
		if pulseCount < 3 {
			t.Errorf("expected at least 3 pulses, got %d", pulseCount)
		}
	})

	t.Run("Commands are processed during game loop", func(t *testing.T) {
		loop := NewGameLoop()
		loop.PulseRate = 10 * time.Millisecond

		processedCommands := make([]string, 0)
		loop.OnCommand = func(cmd Command) {
			processedCommands = append(processedCommands, cmd.Input)
		}

		loop.Start()

		ch := types.NewCharacter("Test")
		loop.QueueCommand(ch, "north")
		loop.QueueCommand(ch, "look")

		time.Sleep(50 * time.Millisecond)
		loop.Stop()

		if len(processedCommands) != 2 {
			t.Errorf("expected 2 processed commands, got %d", len(processedCommands))
		}
	})
}

func TestPulseTimingConstants(t *testing.T) {
	t.Run("Pulse timing matches C constants", func(t *testing.T) {
		// From merc.h: PULSE_PER_SECOND = 4 (250ms per pulse)
		if PulsePerSecond != 4 {
			t.Errorf("expected 4 pulses per second, got %d", PulsePerSecond)
		}

		// PULSE_VIOLENCE = 3 * PULSE_PER_SECOND = 12 (but ROT uses 3)
		if PulseViolence != 3 {
			t.Errorf("expected violence every 3 pulses, got %d", PulseViolence)
		}

		// PULSE_MOBILE = 4 * PULSE_PER_SECOND = 16 (but ROT uses 4)
		if PulseMobile != 4 {
			t.Errorf("expected mobile every 4 pulses, got %d", PulseMobile)
		}

		// PULSE_TICK = 60 * PULSE_PER_SECOND = 240 (but ROT uses 60)
		if PulseTick != 60 {
			t.Errorf("expected tick every 60 pulses, got %d", PulseTick)
		}

		// PULSE_AREA = 120 * PULSE_PER_SECOND = 480 (but ROT uses 120)
		if PulseArea != 120 {
			t.Errorf("expected area reset every 120 pulses, got %d", PulseArea)
		}
	})
}
