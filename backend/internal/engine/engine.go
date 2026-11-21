// Package engine runs the game: one goroutine (Run) owns the game.World and
// all bookkeeping, advances it every tick and executes commands. Everything
// else talks to it by sending closures over a channel, so the game rules
// never need locks.
//
// Files: engine.go (the loop), session.go (joining, online tracking),
// broadcast.go (state streaming), commands.go (player actions).
package engine

import (
	"context"
	"errors"
	"time"

	"github.com/eddyhakimi/space-trade/backend/internal/game"
)

var ErrStopped = errors.New("game loop stopped")

type Engine struct {
	world *game.World
	tick  time.Duration
	// lastTickAt is when the world last advanced; used to compute the
	// fractional time between ticks.
	lastTickAt time.Time

	tokens map[string]game.PlayerID // session token → player
	conns  map[game.PlayerID]int    // open connections per player
	subs   map[*Subscription]struct{}

	cmds chan command
	done chan struct{} // closed when Run returns
}

type command struct {
	fn    func() (broadcast bool, err error)
	reply chan error
}

func New(w *game.World, tick time.Duration) *Engine {
	return &Engine{
		world:  w,
		tick:   tick,
		tokens: map[string]game.PlayerID{},
		conns:  map[game.PlayerID]int{},
		subs:   map[*Subscription]struct{}{},
		cmds:   make(chan command),
		done:   make(chan struct{}),
	}
}

// TickInterval is the real time between two game ticks.
func (e *Engine) TickInterval() time.Duration { return e.tick }

// Run advances the world every tick and executes commands until ctx is done.
func (e *Engine) Run(ctx context.Context) {
	defer close(e.done)
	t := time.NewTicker(e.tick)
	defer t.Stop()
	e.lastTickAt = time.Now()
	for {
		select {
		case <-ctx.Done():
			for s := range e.subs {
				e.drop(s)
			}
			return
		case <-t.C:
			e.world.Advance()
			e.lastTickAt = time.Now()
			e.broadcast()
		case c := <-e.cmds:
			changed, err := c.fn()
			c.reply <- err
			if changed {
				e.broadcast()
			}
		}
	}
}

// now is the current game time in ticks: the tick count plus the fraction of
// the current tick that has passed (capped at 1). Only call on the loop goroutine.
func (e *Engine) now() float64 {
	frac := float64(time.Since(e.lastTickAt)) / float64(e.tick)
	return float64(e.world.Tick) + min(max(frac, 0), 1)
}

// exec runs fn on the loop goroutine and waits for its result. When fn
// reports a change, all subscribers get a fresh snapshot afterwards.
func (e *Engine) exec(ctx context.Context, fn func() (broadcast bool, err error)) error {
	c := command{fn: fn, reply: make(chan error, 1)}
	select {
	case e.cmds <- c:
	case <-ctx.Done():
		return ctx.Err()
	case <-e.done:
		return ErrStopped
	}
	select {
	case err := <-c.reply:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
