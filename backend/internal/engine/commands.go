package engine

import (
	"context"

	"github.com/eddyhakimi/space-trade/backend/internal/game"
)

// Player actions. Each runs on the loop goroutine via exec and broadcasts
// when it changed the world.

// Travel sends one of the player's ships towards (x, y), starting now, or
// with queue appends (x, y) to its route.
func (e *Engine) Travel(ctx context.Context, pid game.PlayerID, ship game.ShipID, x, y float64, queue bool) error {
	return e.exec(ctx, func() (bool, error) {
		err := e.world.Travel(pid, ship, x, y, e.now(), queue)
		return err == nil, err
	})
}
