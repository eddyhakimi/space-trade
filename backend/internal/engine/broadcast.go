package engine

import (
	"log"

	"github.com/eddyhakimi/space-trade/backend/internal/game"
)

// subBuffer is how many snapshots may queue up for one subscriber before
// it is considered too slow and dropped.
const subBuffer = 8

// Subscription receives a snapshot after every tick and every change.
// C is closed when the subscriber is dropped (too slow) or the engine stops.
// Snapshots are value copies shared between subscribers: read them, don't modify them.
type Subscription struct {
	C      <-chan game.Snapshot
	c      chan game.Snapshot
	player game.PlayerID
	closed bool // owned by the loop
}

func (e *Engine) broadcast() {
	if len(e.subs) == 0 {
		return
	}
	snap := e.world.Snapshot(e.now())
	for s := range e.subs {
		select {
		case s.c <- snap:
		default:
			log.Printf("dropping slow subscriber of player %s", s.player)
			e.drop(s)
		}
	}
}

// drop closes a subscription's channel. Connection counting is left to
// Unsubscribe, which the connection handler always calls.
func (e *Engine) drop(s *Subscription) {
	if s.closed {
		return
	}
	s.closed = true
	delete(e.subs, s)
	close(s.c)
}
