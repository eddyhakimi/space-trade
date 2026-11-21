package game

import "math"

// PlunderRadius is how close (in map units) a ship must be to a trader to
// plunder it. Mirrored as PLUNDER_RADIUS in the frontend.
const PlunderRadius = 40

// Plunder moves as much of a travelling AI trader's cargo into the player's
// ship as fits, in catalog order; the trader flies on with the rest. The
// player's ship must be in open space and within PlunderRadius of the trader
// at time now. Docked traders are protected.
func (w *World) Plunder(playerID PlayerID, shipID, targetID ShipID, now float64) error {
	s, ok := w.Ships[shipID]
	if !ok {
		return ErrUnknownShip
	}
	if s.OwnerID != playerID {
		return ErrNotYourShip
	}
	t, ok := w.Ships[targetID]
	if !ok {
		return ErrUnknownShip
	}
	if !t.NPC {
		return ErrNotATrader
	}
	s.advance(now)
	t.advance(now)
	if s.Status == Docked {
		return ErrPlunderDocked
	}
	if t.Status != Travelling {
		return ErrTargetProtected
	}
	sx, sy := s.PositionAt(now)
	tx, ty := t.PositionAt(now)
	if math.Hypot(tx-sx, ty-sy) > PlunderRadius {
		return ErrTooFar
	}
	if t.CargoUsed() == 0 {
		return ErrNothingToPlunder
	}
	free := s.Capacity - s.CargoUsed()
	if free <= 0 {
		return ErrCargoFull
	}
	for _, g := range w.Goods {
		n := min(free, t.Cargo[g.ID])
		if n == 0 {
			continue
		}
		s.Cargo[g.ID] += n
		t.Cargo[g.ID] -= n
		if t.Cargo[g.ID] == 0 {
			delete(t.Cargo, g.ID)
		}
		free -= n
	}
	return nil
}
