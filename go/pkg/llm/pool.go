package llm

import (
	"context"
	"sync"
	"sync/atomic"
)

// chatter is the subset of Client the Pool needs, so tests can inject a stub.
type chatter interface {
	Chat(ctx context.Context, req Request) (Action, error)
}

// Request is one dialog turn to think about. Key is opaque to the Pool and is
// echoed back on the Result so the caller can map it to a mob (we pass the
// *types.Character). The Pool keeps at most one in-flight request per Key.
type Request struct {
	Key          any
	Persona      string
	PlayerName   string
	PlayerSay    string
	PlayerState  string // optional: what the NPC can observe about the speaker
	Greeting     bool   // mob-initiated greeting on player entry (no PlayerSay)
	WorldContext string // optional: area/exits/nearby-mob hints for grounding
}

// Result carries the outcome of a Request. On Err != nil the caller must use
// its scripted fallback. On success Action has already passed Validate.
type Result struct {
	Key    any
	Action Action
	Err    error
}

// Stats is a snapshot for the llmstat immortal command.
type Stats struct {
	Enabled     bool
	Workers     int
	QueueDepth  int   // pending requests in the inbox
	Inflight    int   // requests currently being processed
	BreakerOpen bool  // circuit breaker tripped
	Calls       int64 // LLM calls attempted
	Failures    int64 // calls that failed (transport/parse/validate)
	Drops       int64 // requests dropped (overflow / breaker open / dupe)
}

// Pool runs LLM requests off the game-loop goroutine and never blocks the
// caller. Submit enqueues; results arrive on the channel returned by Results.
type Pool struct {
	cfg     Config
	chat    chatter
	breaker *Breaker

	inbox   chan Request
	results chan Result

	mu       sync.Mutex
	inflight map[any]bool

	calls, failures, drops int64

	wg     sync.WaitGroup
	stop   chan struct{}
	closed bool
}

// NewPool builds a Pool with a real llama.cpp Client from cfg.
func NewPool(cfg Config) *Pool {
	return newPool(cfg, NewClient(cfg.Endpoint, cfg.Model))
}

func newPool(cfg Config, c chatter) *Pool {
	if cfg.Workers < 1 {
		cfg.Workers = 1
	}
	if cfg.Queue < 1 {
		cfg.Queue = 1
	}
	return &Pool{
		cfg:      cfg,
		chat:     c,
		breaker:  NewBreaker(),
		inbox:    make(chan Request, cfg.Queue),
		results:  make(chan Result, cfg.Queue),
		inflight: make(map[any]bool),
		stop:     make(chan struct{}),
	}
}

// Enabled reports whether the feature is on.
func (p *Pool) Enabled() bool { return p.cfg.Enabled }

// Start launches the worker goroutines. Safe to call once.
func (p *Pool) Start() {
	for i := 0; i < p.cfg.Workers; i++ {
		p.wg.Add(1)
		go p.worker()
	}
}

// Stop signals workers to finish and waits for them.
func (p *Pool) Stop() {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.closed = true
	p.mu.Unlock()
	close(p.stop)
	p.wg.Wait()
}

// Results is the channel the game loop drains each tick (non-blocking).
func (p *Pool) Results() <-chan Result { return p.results }

// Submit enqueues a request, returning false if it was dropped: feature off,
// breaker open, a request for this Key is already in flight, or the inbox is
// full. Never blocks. Dropped requests simply leave the mob on its scripted
// path this turn.
func (p *Pool) Submit(req Request) bool {
	if !p.cfg.Enabled {
		return false
	}
	if !p.breaker.Allow() {
		atomic.AddInt64(&p.drops, 1)
		return false
	}

	p.mu.Lock()
	if p.closed || p.inflight[req.Key] {
		p.mu.Unlock()
		atomic.AddInt64(&p.drops, 1)
		return false
	}
	p.inflight[req.Key] = true
	p.mu.Unlock()

	select {
	case p.inbox <- req:
		return true
	default:
		// Inbox full: undo the inflight reservation and drop.
		p.clearInflight(req.Key)
		atomic.AddInt64(&p.drops, 1)
		return false
	}
}

func (p *Pool) worker() {
	defer p.wg.Done()
	for {
		select {
		case <-p.stop:
			return
		case req := <-p.inbox:
			p.handle(req)
		}
	}
}

func (p *Pool) handle(req Request) {
	defer p.clearInflight(req.Key)

	ctx, cancel := context.WithTimeout(context.Background(), p.cfg.Timeout)
	defer cancel()

	atomic.AddInt64(&p.calls, 1)
	act, err := p.chat.Chat(ctx, req)
	if err == nil {
		err = act.Validate()
	}
	p.breaker.Record(err != nil)
	if err != nil {
		atomic.AddInt64(&p.failures, 1)
	}

	res := Result{Key: req.Key, Action: act, Err: err}
	select {
	case p.results <- res:
	case <-p.stop:
	}
}

func (p *Pool) clearInflight(key any) {
	p.mu.Lock()
	delete(p.inflight, key)
	p.mu.Unlock()
}

// Stats returns a snapshot for monitoring.
func (p *Pool) Stats() Stats {
	p.mu.Lock()
	inflight := len(p.inflight)
	p.mu.Unlock()
	return Stats{
		Enabled:     p.cfg.Enabled,
		Workers:     p.cfg.Workers,
		QueueDepth:  len(p.inbox),
		Inflight:    inflight,
		BreakerOpen: p.breaker.IsOpen(),
		Calls:       atomic.LoadInt64(&p.calls),
		Failures:    atomic.LoadInt64(&p.failures),
		Drops:       atomic.LoadInt64(&p.drops),
	}
}
