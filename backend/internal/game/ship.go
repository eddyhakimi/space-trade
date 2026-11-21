package game

import (
	"fmt"
	"math"
)

type ShipID string

// DefaultShipSpeed is the max speed in map units per tick.
const DefaultShipSpeed = 50

// DefaultShipAccel is how fast ships speed up and brake, in map units per tick².
const DefaultShipAccel = 25

// MaxRouteStops is how many waypoints can be queued after the current leg.
const MaxRouteStops = 10

type ShipStatus string

const (
	Docked     ShipStatus = "docked"     // at a station (At)
	Idle       ShipStatus = "idle"       // holding position in open space
	Travelling ShipStatus = "travelling" // flying a leg
)

// Waypoint is a queued destination; To is set when it is a station.
type Waypoint struct {
	X  float64   `json:"x"`
	Y  float64   `json:"y"`
	To StationID `json:"to,omitempty"`
}

// Ship positions use map units; times are in ticks as float64 so legs can
// start between ticks (re-routing mid-flight). See motion.go for how a ship
// moves along a leg.
type Ship struct {
	ID      ShipID     `json:"id"`
	OwnerID PlayerID   `json:"ownerId"`
	Name    string     `json:"name"`
	Speed   float64    `json:"speed"` // max speed
	Accel   float64    `json:"accel"`
	Status  ShipStatus `json:"status"`
	// X/Y is the current position, or where the current leg started while travelling.
	X float64 `json:"x"`
	Y float64 `json:"y"`
	// At is set while docked.
	At StationID `json:"at,omitempty"`
	// The current leg, set while travelling. To is set when the target is a station.
	// V0 is the speed at departure (a re-routed ship keeps its momentum), V1
	// the speed at the target (above 0 at a waypoint it flies through).
	// (No omitempty on the floats: 0 is a valid coordinate and departure time.)
	ToX      float64   `json:"toX"`
	ToY      float64   `json:"toY"`
	To       StationID `json:"to,omitempty"`
	V0       float64   `json:"v0"`
	V1       float64   `json:"v1"`
	DepartAt float64   `json:"departAt"`
	ArriveAt float64   `json:"arriveAt"`
	// Route holds the waypoints flown after the current leg, in order. The
	// ship stops at stations and at the last one; it flies through the others.
	Route []Waypoint `json:"route,omitempty"`
}

// newShip gives owner a new ship docked at station.
func (w *World) newShip(owner *Player, station Station) *Ship {
	w.nextShip++
	s := &Ship{
		ID:      ShipID(fmt.Sprintf("s%d", w.nextShip)),
		OwnerID: owner.ID,
		Name:    owner.Name + "'s freighter",
		Speed:   DefaultShipSpeed,
		Accel:   DefaultShipAccel,
		Status:  Docked,
		X:       station.X,
		Y:       station.Y,
		At:      station.ID,
	}
	w.Ships[s.ID] = s
	return s
}

// leg returns the current leg's length and speed profile.
func (s *Ship) leg() (float64, profile) {
	d := math.Hypot(s.ToX-s.X, s.ToY-s.Y)
	return d, legProfile(d, s.V0, s.V1, s.Accel, s.Speed)
}

// PositionAt is where the ship is at time t (in ticks).
func (s *Ship) PositionAt(t float64) (x, y float64) {
	if s.Status != Travelling {
		return s.X, s.Y
	}
	d, p := s.leg()
	if d == 0 {
		return s.ToX, s.ToY
	}
	dist, _ := p.at(t - s.DepartAt)
	f := dist / d
	return s.X + (s.ToX-s.X)*f, s.Y + (s.ToY-s.Y)*f
}

// SpeedAt is the ship's speed at time t (in map units per tick).
func (s *Ship) SpeedAt(t float64) float64 {
	if s.Status != Travelling {
		return 0
	}
	_, p := s.leg()
	_, v := p.at(t - s.DepartAt)
	return v
}

// Travel sends a ship owned by playerID towards (x, y), starting at time now
// (in ticks, may be between ticks). A target near a station snaps to it and
// the ship docks there on arrival.
//
// Without queue the ship flies there directly, dropping any queued route; a
// ship re-routed mid-flight keeps its speed. Targeting the flying ship itself
// brakes it to a stop. With queue, (x, y) is appended to the route of a
// travelling ship (a ship at rest just departs). Ships stop at stations and
// at the end of the route and fly through the other waypoints.
func (w *World) Travel(playerID PlayerID, shipID ShipID, x, y, now float64, queue bool) error {
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
	s.advance(now) // legs that ended since the last tick

	if queue && s.Status == Travelling {
		last := Waypoint{X: s.ToX, Y: s.ToY}
		if n := len(s.Route); n > 0 {
			last = s.Route[n-1]
		}
		if len(s.Route) >= MaxRouteStops {
			return ErrRouteFull
		}
		if math.Hypot(x-last.X, y-last.Y) < 1 {
			return ErrAlreadyThere
		}
		s.Route = append(s.Route, Waypoint{X: x, Y: y, To: to})
		// The current target may now be a fly-through: re-plan its end speed.
		px, py := s.PositionAt(now)
		v := s.SpeedAt(now)
		s.X, s.Y = px, py
		s.startLeg(s.ToX, s.ToY, s.To, v, now)
		return nil
	}

	fromX, fromY := s.PositionAt(now)
	v := s.SpeedAt(now)
	if math.Hypot(x-fromX, y-fromY) < 1 {
		if s.Status != Travelling {
			return ErrAlreadyThere
		}
		// Clicked on the flying ship itself: brake straight ahead.
		s.Route = nil
		d, _ := s.leg()
		stop := v * v / (2 * s.Accel)
		if d == 0 || stop < 1 {
			s.arrive(fromX, fromY, "")
			return nil
		}
		dx, dy := (s.ToX-s.X)/d, (s.ToY-s.Y)/d
		s.X, s.Y = fromX, fromY
		s.startLeg(fromX+dx*stop, fromY+dy*stop, "", v, now)
		return nil
	}

	s.Route = nil
	s.X, s.Y = fromX, fromY
	s.startLeg(x, y, to, v, now)
	return nil
}

// startLeg starts a leg from the ship's position (X, Y) at time now, followed
// by s.Route.
func (s *Ship) startLeg(x, y float64, to StationID, v0, now float64) {
	d := math.Hypot(x-s.X, y-s.Y)
	s.Status = Travelling
	s.At = ""
	s.ToX, s.ToY, s.To = x, y, to
	v1 := routeEndSpeed(x, y, to, s.Route, s.Accel, s.Speed)
	s.V0, s.V1 = legSpeeds(d, v0, v1, s.Accel, s.Speed)
	s.DepartAt = now
	s.ArriveAt = now + legDuration(d, s.V0, s.V1, s.Accel, s.Speed)
}

// advance ends every leg that is over by time t; a ship with a route departs
// for its next waypoint the moment it arrives, at the speed it arrived with.
func (s *Ship) advance(t float64) {
	for s.Status == Travelling && s.ArriveAt <= t {
		end, v := s.ArriveAt, s.V1
		s.arrive(s.ToX, s.ToY, s.To)
		if len(s.Route) == 0 {
			return
		}
		next := s.Route[0]
		s.Route = s.Route[1:]
		if len(s.Route) == 0 {
			s.Route = nil
		}
		s.startLeg(next.X, next.Y, next.To, v, end)
	}
}

// arrive puts the ship at (x, y): docked if station is set, otherwise idle.
// The route is kept (advance continues it).
func (s *Ship) arrive(x, y float64, station StationID) {
	s.X, s.Y = x, y
	s.Status, s.At = Idle, ""
	if station != "" {
		s.Status, s.At = Docked, station
	}
	s.ToX, s.ToY, s.To = 0, 0, ""
	s.V0, s.V1, s.DepartAt, s.ArriveAt = 0, 0, 0, 0
}
