package game

import "math"

// Ships fly each leg in a straight line with a trapezoidal speed profile:
// accelerate from the start speed v0 to a peak (at most the max speed),
// cruise, then brake to the end speed v1 at the target. v1 is 0 at a full stop
// (a station or the end of the route) and may be higher at a waypoint the
// ship flies through. Accelerating and braking use the same rate. A leg too
// short to reach the max speed has no cruise phase.
//
// Distances are in map units, times in ticks. frontend/src/geometry.js
// mirrors this file (legSpeeds, legProfile, profileAt, routeEndSpeed): keep
// them identical.

// legSpeeds limits a leg's start and end speed to what is possible: at most
// vmax, the ship can brake from v0 to v1 and accelerate from v0 to v1 within d.
func legSpeeds(d, v0, v1, a, vmax float64) (float64, float64) {
	v1 = min(max(v1, 0), vmax)
	v0 = min(max(v0, 0), vmax, math.Sqrt(v1*v1+2*a*d))
	v1 = min(v1, math.Sqrt(v0*v0+2*a*d))
	return v0, v1
}

// profile is one leg's speed curve.
type profile struct {
	d, v0, v1, a, peak float64
	t1, t2, total      float64 // end of accelerating, end of cruising, end of leg
	d1                 float64 // distance covered while accelerating
}

func legProfile(d, v0, v1, a, vmax float64) profile {
	if d <= 0 || a <= 0 || vmax <= 0 {
		return profile{d: max(d, 0)}
	}
	v0, v1 = legSpeeds(d, v0, v1, a, vmax)
	peak := vmax
	if (2*vmax*vmax-v0*v0-v1*v1)/(2*a) > d {
		peak = math.Sqrt((2*a*d + v0*v0 + v1*v1) / 2)
	}
	t1 := (peak - v0) / a
	d1 := (peak*peak - v0*v0) / (2 * a)
	d3 := (peak*peak - v1*v1) / (2 * a)
	t2 := t1
	if peak > 0 {
		t2 += max(d-d1-d3, 0) / peak
	}
	return profile{d: d, v0: v0, v1: v1, a: a, peak: peak, t1: t1, t2: t2, total: t2 + (peak-v1)/a, d1: d1}
}

// at returns the distance covered and the speed dt ticks into the leg.
func (p profile) at(dt float64) (dist, speed float64) {
	switch {
	case p.total <= 0:
		return p.d, p.v1
	case dt <= 0:
		return 0, p.v0
	case dt < p.t1:
		return p.v0*dt + p.a*dt*dt/2, p.v0 + p.a*dt
	case dt < p.t2:
		return p.d1 + p.peak*(dt-p.t1), p.peak
	case dt < p.total:
		r := p.total - dt
		return p.d - p.v1*r - p.a*r*r/2, p.v1 + p.a*r
	default:
		return p.d, p.v1
	}
}

// legDuration is how many ticks a leg of distance d takes.
func legDuration(d, v0, v1, a, vmax float64) float64 {
	return legProfile(d, v0, v1, a, vmax).total
}

// legState returns the distance covered and the speed dt ticks into a leg.
func legState(d, v0, v1, a, vmax, dt float64) (dist, speed float64) {
	return legProfile(d, v0, v1, a, vmax).at(dt)
}

// routeEndSpeed is the fastest a ship may pass the point (x, y) (reached
// with station to, followed by route) and still brake in time for its next
// full stop: a station or the end of the route.
func routeEndSpeed(x, y float64, to StationID, route []Waypoint, a, vmax float64) float64 {
	if to != "" || len(route) == 0 {
		return 0
	}
	// Distances of the legs up to the next full stop.
	var legs []float64
	for _, wp := range route {
		legs = append(legs, math.Hypot(wp.X-x, wp.Y-y))
		x, y = wp.X, wp.Y
		if wp.To != "" {
			break
		}
	}
	v := 0.0
	for i := len(legs) - 1; i >= 0; i-- {
		v = min(vmax, math.Sqrt(v*v+2*a*legs[i]))
	}
	return v
}
