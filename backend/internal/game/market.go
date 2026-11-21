package game

import "math"

// Listing is one good a station trades, priced from the ship's point of view:
// Buy is what a ship pays per unit now (0 = not for sale here), Sell is what
// a ship gets per unit now. Both follow the normal (base) prices, moved by
// the market pressure.
type Listing struct {
	Good GoodID `json:"good"`
	Buy  int    `json:"buy,omitempty"`
	Sell int    `json:"sell"`
	// Normal prices, at pressure 0.
	BaseBuy  int `json:"baseBuy,omitempty"`
	BaseSell int `json:"baseSell"`
	// Pressure is the net number of units taken from the station (bought minus
	// sold); it raises prices when positive, lowers them when negative, and
	// decays every tick.
	Pressure float64 `json:"pressure"`
}

const (
	// BuybackPercent is the share of its own price a station pays to buy back
	// a good it produces.
	BuybackPercent = 80
	// PriceImpact: each unit of pressure moves a price by a factor e^PriceImpact (≈1 %).
	PriceImpact = 0.01
	// PressureHalfLife is how many ticks it takes for half of a pressure to fade.
	PressureHalfLife = 60
	// MaxPressure bounds the pressure, so prices stay within e^±1 of normal.
	MaxPressure = 100
)

// Pressure below this snaps to 0, so prices return exactly to normal.
const pressureEpsilon = 0.05

// pressureDecay is the factor applied to every pressure each tick.
var pressureDecay = math.Pow(0.5, 1.0/PressureHalfLife)

// priceAt is the price of a good with normal price base at pressure p.
func priceAt(base int, p float64) int {
	return max(1, int(math.Round(float64(base)*math.Exp(PriceImpact*p))))
}

// reprice recomputes the current prices from the base prices and pressure.
// A station always buys back its own produce below its selling price.
func (l *Listing) reprice() {
	if l.BaseBuy > 0 {
		l.Buy = priceAt(l.BaseBuy, l.Pressure)
		l.Sell = max(1, l.Buy*BuybackPercent/100)
		return
	}
	l.Sell = priceAt(l.BaseSell, l.Pressure)
}

// push moves the pressure by units (+ for units taken from the station),
// within ±MaxPressure, and reprices.
func (l *Listing) push(units float64) {
	l.Pressure = min(max(l.Pressure+units, -MaxPressure), MaxPressure)
	l.reprice()
}

// decay lets the pressure fade by one tick.
func (l *Listing) decay() {
	if l.Pressure == 0 {
		return
	}
	l.Pressure *= pressureDecay
	if math.Abs(l.Pressure) < pressureEpsilon {
		l.Pressure = 0
	}
	l.reprice()
}

// BuyCost is what buying qty units costs, unit by unit as the price rises.
// The listing itself is not changed.
func (l Listing) BuyCost(qty int) int {
	total := 0
	for range qty {
		total += l.Buy
		l.push(1)
	}
	return total
}

// SellValue is what selling qty units pays, unit by unit as the price falls.
// The listing itself is not changed.
func (l Listing) SellValue(qty int) int {
	total := 0
	for range qty {
		total += l.Sell
		l.push(-1)
	}
	return total
}

// produces lists a good the station sells at price and buys back cheaper.
func produces(g GoodID, price int) Listing {
	l := Listing{Good: g, BaseBuy: price, BaseSell: price * BuybackPercent / 100}
	l.reprice()
	return l
}

// needs lists a good the station only buys, at price.
func needs(g GoodID, price int) Listing {
	l := Listing{Good: g, BaseSell: price}
	l.reprice()
	return l
}

// defaultMarkets is the table of normal prices (see PLAN.md). Listings are in
// catalog order; goods a station doesn't list can't be traded there.
func defaultMarkets() map[StationID][]Listing {
	return map[StationID][]Listing{
		"kepler": {
			needs("food", 26), needs("ore", 38), produces("metals", 45), needs("fuel", 50),
			needs("electronics", 185), produces("machinery", 78), needs("luxuries", 300),
		},
		"ostrava": {
			needs("food", 30), needs("water", 16), produces("ore", 18), produces("metals", 40),
			needs("textiles", 44), needs("medicine", 150), needs("machinery", 135),
		},
		"halden": {
			needs("food", 28), produces("water", 8), produces("ore", 22), produces("fuel", 26),
			needs("textiles", 46), needs("medicine", 160), needs("machinery", 140), needs("luxuries", 320),
		},
		"vesta": {
			produces("food", 14), needs("water", 14), needs("metals", 72), needs("fuel", 48),
			produces("textiles", 24), needs("electronics", 175),
		},
		"iolo": {
			needs("water", 18), needs("ore", 42), needs("metals", 80), produces("medicine", 85),
			produces("electronics", 105), needs("machinery", 130), needs("luxuries", 290),
		},
		"marrow": {
			produces("food", 16), needs("fuel", 55), needs("textiles", 42), needs("medicine", 140),
			needs("electronics", 190), produces("luxuries", 170),
		},
	}
}

// listing returns the station's listing for g, if it trades it.
func (st Station) listing(g GoodID) (Listing, bool) {
	for _, l := range st.Market {
		if l.Good == g {
			return l, true
		}
	}
	return Listing{}, false
}

// listingRef returns the live listing for g at station at, for trades.
func (w *World) listingRef(at StationID, g GoodID) (*Listing, bool) {
	i, ok := w.stationIdx[at]
	if !ok {
		return nil, false
	}
	m := w.Stations[i].Market
	for j := range m {
		if m[j].Good == g {
			return &m[j], true
		}
	}
	return nil, false
}

// decayMarkets lets every station's price pressure fade by one tick.
func (w *World) decayMarkets() {
	for i := range w.Stations {
		for j := range w.Stations[i].Market {
			w.Stations[i].Market[j].decay()
		}
	}
}
