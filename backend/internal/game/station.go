package game

import "math"

type StationID string

// SnapRadius: a travel target this close to a station targets the station itself.
const SnapRadius = 40

type Station struct {
	ID   StationID `json:"id"`
	Name string    `json:"name"`
	X    float64   `json:"x"`
	Y    float64   `json:"y"`
	// Market lists the goods traded here (see market.go).
	Market []Listing `json:"market"`
}

// defaultStations is the fixed layout of the sector, with each station's market.
func defaultStations() []Station {
	stations := []Station{
		{ID: "kepler", Name: "Kepler Yard", X: 150, Y: 200},
		{ID: "ostrava", Name: "Ostrava Ring", X: 520, Y: 120},
		{ID: "halden", Name: "Halden Deep", X: 860, Y: 300},
		{ID: "vesta", Name: "Vesta Gate", X: 300, Y: 560},
		{ID: "iolo", Name: "Iolo Station", X: 720, Y: 680},
		{ID: "marrow", Name: "Marrow Point", X: 180, Y: 880},
	}
	markets := defaultMarkets()
	for i := range stations {
		stations[i].Market = markets[stations[i].ID]
	}
	return stations
}

// Station returns the station with the given ID.
func (w *World) Station(id StationID) (Station, bool) {
	i, ok := w.stationIdx[id]
	if !ok {
		return Station{}, false
	}
	return w.Stations[i], true
}

// nearestStation returns the station within SnapRadius of (x, y), if any.
func (w *World) nearestStation(x, y float64) (Station, bool) {
	best, bestD := Station{}, math.Inf(1)
	for _, st := range w.Stations {
		if d := math.Hypot(st.X-x, st.Y-y); d < bestD {
			best, bestD = st, d
		}
	}
	return best, bestD <= SnapRadius
}
