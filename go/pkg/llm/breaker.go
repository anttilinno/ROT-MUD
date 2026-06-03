package llm

import (
	"sync"
	"time"
)

// Breaker is a circuit breaker over the LLM endpoint (LLM-NPC.md "Circuit
// breaker"). It tracks the last `window` outcomes; if at least `threshold`
// failed it opens for `cooldown`, during which Allow returns false and callers
// skip the LLM entirely. After cooldown it goes half-open: one trial call is
// allowed; success closes the breaker, failure re-opens it.
//
// The clock is injectable (now) so tests need not sleep.
type Breaker struct {
	window    int
	threshold int
	cooldown  time.Duration
	now       func() time.Time

	mu       sync.Mutex
	outcomes []bool // true = failure; ring of up to `window`
	openedAt time.Time
	open     bool
	halfOpen bool // a trial call is currently permitted/in-flight
}

// NewBreaker returns a breaker matching the plan defaults: window 10,
// threshold 5, cooldown 60s.
func NewBreaker() *Breaker {
	return &Breaker{
		window:    10,
		threshold: 5,
		cooldown:  60 * time.Second,
		now:       time.Now,
	}
}

// Allow reports whether a call may proceed. When the breaker is open it stays
// closed-to-traffic until cooldown elapses, then admits a single trial.
func (b *Breaker) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.open {
		return true
	}
	if b.now().Sub(b.openedAt) >= b.cooldown {
		// Cooldown elapsed: admit exactly one trial call.
		if !b.halfOpen {
			b.halfOpen = true
			return true
		}
		return false // a trial is already out; wait for its Record
	}
	return false
}

// Record reports the outcome of a call (failed=true on any transport, parse, or
// validation failure). It updates the rolling window and breaker state.
func (b *Breaker) Record(failed bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.halfOpen {
		// Trial result decides the breaker.
		b.halfOpen = false
		if failed {
			b.open = true
			b.openedAt = b.now()
		} else {
			b.reset()
		}
		return
	}

	b.outcomes = append(b.outcomes, failed)
	if len(b.outcomes) > b.window {
		b.outcomes = b.outcomes[len(b.outcomes)-b.window:]
	}

	if !b.open && len(b.outcomes) >= b.window && b.failures() >= b.threshold {
		b.open = true
		b.openedAt = b.now()
	}
}

// IsOpen reports the current breaker state (for llmstat).
func (b *Breaker) IsOpen() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.open
}

func (b *Breaker) failures() int {
	n := 0
	for _, f := range b.outcomes {
		if f {
			n++
		}
	}
	return n
}

func (b *Breaker) reset() {
	b.open = false
	b.halfOpen = false
	b.outcomes = b.outcomes[:0]
}
