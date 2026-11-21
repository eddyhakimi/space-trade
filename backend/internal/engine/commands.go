package engine

import (
	"context"

	"github.com/eddyhakimi/space-trade/backend/internal/game"
)

// Player actions. Each runs on the loop goroutine via exec and broadcasts
// when it changed the world.

// Travel sends one of the player's ships towards (x, y), starting now.
func (e *Engine) Travel(ctx context.Context, pid game.PlayerID, ship game.ShipID, x, y float64) error {
	return e.exec(ctx, func() (bool, error) {
		err := e.world.Travel(pid, ship, x, y, e.now())
		return err == nil, err
	})
}
