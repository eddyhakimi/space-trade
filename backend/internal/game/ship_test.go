package game

import (
	"errors"
	"math"
	"testing"
)

func TestTravelToOpenSpace(t *testing.T) {
	w := NewWorld(1)
	p, s := newPlayer(t, w, "Alice")
	x, y := openPoint(t, w)
	startX, startY := s.X, s.Y
	now := float64(w.Tick)

	if err := w.Travel(p.ID, s.ID, x, y, now); err != nil {
		t.Fatal(err)
	}
	wantArrive := now + math.Hypot(x-startX, y-startY)/s.Speed
	if s.Status != Travelling || s.At != "" || s.To != "" || !near(s.ArriveAt, wantArrive) {
		t.Fatalf("after Travel: %+v, want arrival at %v", *s, wantArrive)
	}

	// Halfway along the leg.
	mx, my := s.PositionAt((s.DepartAt + s.ArriveAt) / 2)
	if !near(mx, (startX+x)/2) || !near(my, (startY+y)/2) {
		t.Errorf("midpoint (%v,%v), want (%v,%v)", mx, my, (startX+x)/2, (startY+y)/2)
	}

	stoppedAt := advanceUntil(t, w, s)
	if want := int64(math.Ceil(wantArrive)); stoppedAt != want {
		t.Errorf("arrived at tick %d, want first tick ≥ %v (%d)", stoppedAt, wantArrive, want)
	}
	if s.Status != Idle || s.X != x || s.Y != y || s.At != "" {
		t.Errorf("after arrival: %+v, want idle at (%v,%v)", *s, x, y)
	}
}

func TestRerouteMidFlight(t *testing.T) {
	w := NewWorld(1)
	p, s := newPlayer(t, w, "Alice")
	x1, y1 := openPoint(t, w)
	if err := w.Travel(p.ID, s.ID, x1, y1, 0); err != nil {
		t.Fatal(err)
	}
	w.Advance()
	w.Advance()

	now := 2.5 // between ticks
	px, py := s.PositionAt(now)
	dest := otherStation(w, "")
	if err := w.Travel(p.ID, s.ID, dest.X, dest.Y, now); err != nil {
		t.Fatal(err)
	}
	if s.X != px || s.Y != py || s.DepartAt != now {
		t.Fatalf("new leg starts at (%v,%v)@%v, want (%v,%v)@%v", s.X, s.Y, s.DepartAt, px, py, now)
	}
	if want := now + math.Hypot(dest.X-px, dest.Y-py)/s.Speed; !near(s.ArriveAt, want) {
		t.Fatalf("ArriveAt %v, want %v", s.ArriveAt, want)
	}
	if s.To != dest.ID {
		t.Fatalf("To = %q, want %s", s.To, dest.ID)
	}

	// Clicking on the flying ship itself stops it there.
	if err := w.Travel(p.ID, s.ID, px, py, now); err != nil {
		t.Fatal(err)
	}
	if s.Status != Idle || s.X != px || s.Y != py {
		t.Fatalf("stop in place: %+v", *s)
	}
}

func TestTravelErrors(t *testing.T) {
	w := NewWorld(1)
	alice, aShip := newPlayer(t, w, "Alice")
	bob, _ := newPlayer(t, w, "Bob")
	here, _ := w.Station(aShip.At)

	cases := []struct {
		name   string
		player PlayerID
		ship   ShipID
		x, y   float64
		want   error
	}{
		{"unknown ship", alice.ID, "nope", 500, 500, ErrUnknownShip},
		{"someone else's ship", bob.ID, aShip.ID, 500, 500, ErrNotYourShip},
		{"left of the map", alice.ID, aShip.ID, -1, 500, ErrOutOfBounds},
		{"below the map", alice.ID, aShip.ID, 500, MapSize + 1, ErrOutOfBounds},
		{"NaN", alice.ID, aShip.ID, math.NaN(), 500, ErrOutOfBounds},
		{"current station", alice.ID, aShip.ID, here.X + 5, here.Y, ErrAlreadyThere},
	}
	for _, c := range cases {
		if err := w.Travel(c.player, c.ship, c.x, c.y, 0); !errors.Is(err, c.want) {
			t.Errorf("%s: got %v, want %v", c.name, err, c.want)
		}
	}
	if aShip.Status != Docked {
		t.Errorf("rejected commands changed the ship: %+v", *aShip)
	}
}
