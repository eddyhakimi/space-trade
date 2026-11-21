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

	if err := w.Travel(p.ID, s.ID, x, y, now, false); err != nil {
		t.Fatal(err)
	}
	wantArrive := now + legDuration(math.Hypot(x-startX, y-startY), 0, 0, s.Accel, s.Speed)
	if s.Status != Travelling || s.At != "" || s.To != "" || s.V0 != 0 || !near(s.ArriveAt, wantArrive) {
		t.Fatalf("after Travel: %+v, want arrival at %v", *s, wantArrive)
	}

	// Speeding up from rest, then braking: halfway in time is halfway in space.
	mid := (s.DepartAt + s.ArriveAt) / 2
	mx, my := s.PositionAt(mid)
	if !near(mx, (startX+x)/2) || !near(my, (startY+y)/2) {
		t.Errorf("midpoint (%v,%v), want (%v,%v)", mx, my, (startX+x)/2, (startY+y)/2)
	}
	if v := s.SpeedAt(now + 0.5); v <= 0 || v >= s.SpeedAt(mid) {
		t.Errorf("speed at departure %v, mid-leg %v: want accelerating", v, s.SpeedAt(mid))
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
	if err := w.Travel(p.ID, s.ID, x1, y1, 0, false); err != nil {
		t.Fatal(err)
	}
	w.Advance()
	w.Advance()

	now := 2.5 // between ticks
	px, py := s.PositionAt(now)
	v := s.SpeedAt(now)
	if s.Status != Travelling || v <= 0 {
		t.Fatalf("not flying at %v: %+v", now, *s)
	}
	dest := otherStation(w, "")
	if err := w.Travel(p.ID, s.ID, dest.X, dest.Y, now, false); err != nil {
		t.Fatal(err)
	}
	if s.X != px || s.Y != py || s.DepartAt != now {
		t.Fatalf("new leg starts at (%v,%v)@%v, want (%v,%v)@%v", s.X, s.Y, s.DepartAt, px, py, now)
	}
	d := math.Hypot(dest.X-px, dest.Y-py)
	if want := min(v, math.Sqrt(2*s.Accel*d)); !near(s.V0, want) {
		t.Fatalf("V0 %v, want %v (keeps its speed)", s.V0, want)
	}
	if want := now + legDuration(d, s.V0, s.V1, s.Accel, s.Speed); !near(s.ArriveAt, want) {
		t.Fatalf("ArriveAt %v, want %v", s.ArriveAt, want)
	}
	if s.To != dest.ID {
		t.Fatalf("To = %q, want %s", s.To, dest.ID)
	}

	// Clicking on the flying ship itself brakes it to a stop straight ahead.
	now = 3
	px, py = s.PositionAt(now)
	v = s.SpeedAt(now)
	if err := w.Travel(p.ID, s.ID, px, py, now, false); err != nil {
		t.Fatal(err)
	}
	stop := v * v / (2 * s.Accel)
	if s.Status != Travelling || s.To != "" || !near(s.V0, v) || !near(s.ArriveAt, now+v/s.Accel) {
		t.Fatalf("braking: %+v, want %v ticks to stop", *s, v/s.Accel)
	}
	if got := math.Hypot(s.ToX-px, s.ToY-py); math.Abs(got-stop) > 1e-6 {
		t.Fatalf("stops %v ahead, want %v", got, stop)
	}
	advanceUntil(t, w, s)
	if s.Status != Idle {
		t.Fatalf("after braking: %+v, want idle", *s)
	}
}

func TestQueuedRoute(t *testing.T) {
	w := NewWorld(1)
	p, s := newPlayer(t, w, "Alice")
	ax, ay := openPoint(t, w)
	b := otherStation(w, s.At)

	// Queueing while docked just departs.
	if err := w.Travel(p.ID, s.ID, ax, ay, 0, true); err != nil {
		t.Fatal(err)
	}
	if s.Status != Travelling || len(s.Route) != 0 || s.ToX != ax {
		t.Fatalf("queue while docked: %+v", *s)
	}
	for _, wp := range [][2]float64{{b.X + 5, b.Y}, {ax, ay}} {
		if err := w.Travel(p.ID, s.ID, wp[0], wp[1], 0.5, true); err != nil {
			t.Fatal(err)
		}
	}
	if len(s.Route) != 2 || s.Route[0] != (Waypoint{X: b.X, Y: b.Y, To: b.ID}) || s.Route[1] != (Waypoint{X: ax, Y: ay}) {
		t.Fatalf("route %+v", s.Route)
	}
	if err := w.Travel(p.ID, s.ID, ax, ay, 0.5, true); !errors.Is(err, ErrAlreadyThere) {
		t.Fatalf("same stop twice: %v", err)
	}

	// A is a waypoint in open space followed by more stops: the ship flies
	// through it, and the next leg starts with that speed the moment it arrives.
	firstEnd, passSpeed := s.ArriveAt, s.V1
	if passSpeed <= 0 {
		t.Fatalf("stops at a fly-through waypoint: %+v", *s)
	}
	for float64(w.Tick) < firstEnd {
		w.Advance()
	}
	if s.Status != Travelling || s.To != b.ID || s.DepartAt != firstEnd || !near(s.V0, passSpeed) || s.V1 != 0 ||
		s.X != ax || s.Y != ay || len(s.Route) != 1 {
		t.Fatalf("second leg: %+v", *s)
	}

	// B is a station: the ship stops there, then departs from rest.
	secondEnd := s.ArriveAt
	for float64(w.Tick) < secondEnd {
		w.Advance()
	}
	if s.Status != Travelling || s.X != b.X || s.Y != b.Y || s.V0 != 0 || s.V1 != 0 || s.Route != nil {
		t.Fatalf("third leg: %+v", *s)
	}
	advanceUntil(t, w, s)
	if s.Status != Idle || s.X != ax || s.Y != ay || s.Route != nil {
		t.Fatalf("end of route: %+v", *s)
	}

	// A plain travel drops the route; the route has a limit.
	if err := w.Travel(p.ID, s.ID, b.X, b.Y, float64(w.Tick), false); err != nil {
		t.Fatal(err)
	}
	for i := range MaxRouteStops {
		x, y := ax, ay
		if i%2 == 1 {
			x, y = b.X, b.Y
		}
		if err := w.Travel(p.ID, s.ID, x, y, float64(w.Tick), true); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Travel(p.ID, s.ID, b.X, b.Y, float64(w.Tick), true); !errors.Is(err, ErrRouteFull) {
		t.Fatalf("full route: %v", err)
	}
	if err := w.Travel(p.ID, s.ID, ax, ay, float64(w.Tick), false); err != nil || s.Route != nil {
		t.Fatalf("plain travel kept the route: %v %+v", err, s.Route)
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
		if err := w.Travel(c.player, c.ship, c.x, c.y, 0, false); !errors.Is(err, c.want) {
			t.Errorf("%s: got %v, want %v", c.name, err, c.want)
		}
	}
	if aShip.Status != Docked {
		t.Errorf("rejected commands changed the ship: %+v", *aShip)
	}
}
