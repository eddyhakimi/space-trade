package game

import (
	"fmt"
	"math"
)

type ShipID string

// DefaultShipSpeed is in map units per tick.
const DefaultShipSpeed = 50

type ShipStatus string

const (
	Docked     ShipStatus = "docked"     // at a station (At)
	Idle       ShipStatus = "idle"       // holding position in open space
	Travelling ShipStatus = "travelling" // flying a leg
)

// Ship positions use map units; times are in ticks as float64 so legs can
// start between ticks (re-routing mid-flight).
type Ship struct {
	ID      ShipID     `json:"id"`
	OwnerID PlayerID   `json:"ownerId"`
	Name    string     `json:"name"`
	Speed   float64    `json:"speed"`
	Status  ShipStatus `json:"status"`
	// X/Y is the current position, or where the current leg started while travelling.
	X float64 `json:"x"`
	Y float64 `json:"y"`
	// At is set while docked.
	At StationID `json:"at,omitempty"`
	// The current leg, set while travelling. To is set when the target is a station.
	// (No omitempty on the floats: 0 is a valid coordinate and departure time.)
	ToX      float64   `json:"toX"`
	ToY      float64   `json:"toY"`
	To       StationID `json:"to,omitempty"`
	DepartAt float64   `json:"departAt"`
	ArriveAt float64   `json:"arriveAt"`
}

// newShip gives owner a new ship docked at station.
func (w *World) newShip(owner *Player, station Station) *Ship {
	w.nextShip++
	s := &Ship{
		ID:      ShipID(fmt.Sprintf("s%d", w.nextShip)),
		OwnerID: owner.ID,
		Name:    owner.Name + "'s freighter",
		Speed:   DefaultShipSpeed,
		Status:  Docked,
		X:       station.X,
		Y:       station.Y,
		At:      station.ID,
	}
	w.Ships[s.ID] = s
	return s
}

// PositionAt is where the ship is at time t (in ticks).
func (s *Ship) PositionAt(t float64) (x, y float64) {
	if s.Status != Travelling || s.ArriveAt <= s.DepartAt {
		return s.X, s.Y
	}
	p := min(max((t-s.DepartAt)/(s.ArriveAt-s.DepartAt), 0), 1)
	return s.X + (s.ToX-s.X)*p, s.Y + (s.ToY-s.Y)*p
}

// Travel sends a ship owned by playerID towards (x, y), starting at time now
// (in ticks, may be between ticks). A target near a station snaps to it and
// the ship docks there on arrival. Ships can be re-routed mid-flight.
func (w *World) Travel(playerID PlayerID, shipID ShipID, x, y, now float64) error {
	s, ok := w.Ships[shipID]
	if !ok {
		return ErrUnknownShip
	}
	if s.OwnerID != playerID {
		return ErrNotYourShip
	}
	if math.IsNaN(x) || math.IsNaN(y) || x < 0 || x > MapSize || y < 0 || y > MapSize {
		return ErrOutOfBounds
	}

	var to StationID
	if st, ok := w.nearestStation(x, y); ok {
		x, y, to = st.X, st.Y, st.ID
	}

	fromX, fromY := s.PositionAt(now)
	dist := math.Hypot(x-fromX, y-fromY)
	if dist < 1 {
		if s.Status == Travelling {
			s.arrive(x, y, to) // clicked on the flying ship itself: stop here
			return nil
		}
		return ErrAlreadyThere
	}

	s.Status = Travelling
	s.X, s.Y = fromX, fromY
	s.At = ""
	s.ToX, s.ToY, s.To = x, y, to
	s.DepartAt = now
	s.ArriveAt = now + dist/s.Speed
	return nil
}

// arrive puts the ship at (x, y): docked if station is set, otherwise idle.
func (s *Ship) arrive(x, y float64, station StationID) {
	s.X, s.Y = x, y
	s.Status, s.At = Idle, ""
	if station != "" {
		s.Status, s.At = Docked, station
	}
	s.ToX, s.ToY, s.To = 0, 0, ""
	s.DepartAt, s.ArriveAt = 0, 0
}
