package game

import (
	"math"
	"testing"
)

func TestLegFullSpeed(t *testing.T) {
	const d, a, vmax = 1000.0, 25.0, 50.0
	// From rest: 2 ticks accelerating (50 units), 18 cruising, 2 braking (50 units).
	total := legDuration(d, 0, 0, a, vmax)
	if want := d/vmax + vmax/a; !near(total, want) {
		t.Fatalf("duration %v, want %v", total, want)
	}
	prev := 0.0
	for dt := 0.0; dt <= total; dt += 0.05 {
		dist, v := legState(d, 0, 0, a, vmax, dt)
		if v > vmax+1e-9 || v < 0 {
			t.Fatalf("speed %v at %v, want 0..%v", v, dt, vmax)
		}
		if dist < prev-1e-9 {
			t.Fatalf("went backwards at %v: %v < %v", dt, dist, prev)
		}
		prev = dist
	}
	if _, v := legState(d, 0, 0, a, vmax, 1); !near(v, a) {
		t.Errorf("speed after 1 tick %v, want %v (accelerating)", v, a)
	}
	if _, v := legState(d, 0, 0, a, vmax, total/2); !near(v, vmax) {
		t.Errorf("speed mid-leg %v, want max %v", v, vmax)
	}
	if _, v := legState(d, 0, 0, a, vmax, total-1); !near(v, a) {
		t.Errorf("speed 1 tick before arrival %v, want %v (braking)", v, a)
	}
	if dist, v := legState(d, 0, 0, a, vmax, total); !near(dist, d) || v != 0 {
		t.Errorf("at arrival (%v, %v), want (%v, 0)", dist, v, d)
	}
	if dist, v := legState(d, 0, 0, a, vmax, total+5); dist != d || v != 0 {
		t.Errorf("after arrival (%v, %v), want (%v, 0)", dist, v, d)
	}

	// Starting at full speed skips the acceleration phase.
	if got, want := legDuration(d, vmax, 0, a, vmax), d/vmax+vmax/(2*a); !near(got, want) {
		t.Errorf("duration at full speed %v, want %v", got, want)
	}
}

func TestLegTooShortForFullSpeed(t *testing.T) {
	const d, a, vmax = 40.0, 25.0, 50.0
	peak := math.Sqrt(a * d)
	total := legDuration(d, 0, 0, a, vmax)
	if !near(total, 2*peak/a) {
		t.Fatalf("duration %v, want %v", total, 2*peak/a)
	}
	dist, v := legState(d, 0, 0, a, vmax, total/2)
	if !near(v, peak) || !near(dist, d/2) {
		t.Errorf("halfway (%v, %v), want (%v, %v)", dist, v, d/2, peak)
	}
}

func TestLegStartSpeedClamped(t *testing.T) {
	// Too fast to stop within 10 units: the start speed is capped.
	const d, a, vmax = 10.0, 25.0, 50.0
	if _, v := legState(d, vmax, 0, a, vmax, 0); !near(v, math.Sqrt(2*a*d)) {
		t.Errorf("start speed %v, want %v", v, math.Sqrt(2*a*d))
	}
	if dist, v := legState(d, vmax, 0, a, vmax, legDuration(d, vmax, 0, a, vmax)); !near(dist, d) || v != 0 {
		t.Errorf("end (%v, %v), want (%v, 0)", dist, v, d)
	}
}

func TestLegEndSpeed(t *testing.T) {
	const d, a, vmax = 1000.0, 25.0, 50.0
	// Flying through at full speed: no braking at all.
	if got := legDuration(d, vmax, vmax, a, vmax); !near(got, d/vmax) {
		t.Errorf("cruise-through duration %v, want %v", got, d/vmax)
	}
	total := legDuration(d, 0, 20, a, vmax)
	if dist, v := legState(d, 0, 20, a, vmax, total); !near(dist, d) || !near(v, 20) {
		t.Errorf("end (%v, %v), want (%v, 20)", dist, v, d)
	}
	// From rest, a short leg can't reach the requested end speed.
	if _, v := legState(8, 0, vmax, a, vmax, 100); !near(v, math.Sqrt(2*a*8)) {
		t.Errorf("end speed %v, want %v", v, math.Sqrt(2*a*8))
	}
}

func TestRouteEndSpeed(t *testing.T) {
	const a, vmax = 25.0, 50.0
	far := []Waypoint{{X: 500, Y: 0}}
	if v := routeEndSpeed(0, 0, "", nil, a, vmax); v != 0 {
		t.Errorf("last stop: %v, want 0", v)
	}
	if v := routeEndSpeed(0, 0, "st", far, a, vmax); v != 0 {
		t.Errorf("station: %v, want 0", v)
	}
	if v := routeEndSpeed(0, 0, "", far, a, vmax); v != vmax {
		t.Errorf("long next leg: %v, want %v", v, vmax)
	}
	// Two short legs to the final stop: enough room to brake from sqrt(2·a·20).
	short := []Waypoint{{X: 10, Y: 0}, {X: 20, Y: 0}}
	if v := routeEndSpeed(0, 0, "", short, a, vmax); !near(v, math.Sqrt(2*a*20)) {
		t.Errorf("short legs: %v, want %v", v, math.Sqrt(2*a*20))
	}
	// A station ends the look-ahead even if more stops follow.
	station := []Waypoint{{X: 10, Y: 0, To: "st"}, {X: 900, Y: 0}}
	if v := routeEndSpeed(0, 0, "", station, a, vmax); !near(v, math.Sqrt(2*a*10)) {
		t.Errorf("station next: %v, want %v", v, math.Sqrt(2*a*10))
	}
}
