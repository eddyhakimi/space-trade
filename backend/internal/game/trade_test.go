package game

import (
	"errors"
	"maps"
	"math"
	"testing"
)

func TestBuyAndSellBack(t *testing.T) {
	w := NewWorld(1)
	p, s, l := dockedTrade(t, w, "Alice")

	if err := w.Buy(p.ID, s.ID, l.Good, 3, 0); err != nil {
		t.Fatal(err)
	}
	cost := l.BuyCost(3)
	if p.Credits != StartingCredits-cost || s.Cargo[l.Good] != 3 {
		t.Fatalf("after buy: %d credits, cargo %v", p.Credits, s.Cargo)
	}
	after, _ := w.listingRef(s.At, l.Good)
	value := after.SellValue(3)
	if err := w.Sell(p.ID, s.ID, l.Good, 3, 0); err != nil {
		t.Fatal(err)
	}
	if p.Credits != StartingCredits-cost+value || value >= cost {
		t.Fatalf("after sell: %d credits (paid %d, got %d)", p.Credits, cost, value)
	}
	if _, ok := s.Cargo[l.Good]; ok {
		t.Fatalf("empty cargo entry kept: %v", s.Cargo)
	}
}

func TestTradeRoundTripProfits(t *testing.T) {
	w := NewWorld(1)
	p, s, l := dockedTrade(t, w, "Alice")

	// The station that pays the most for the good.
	var dest Station
	for _, st := range w.Stations {
		if dl, ok := st.listing(l.Good); ok && st.ID != s.At {
			if cur, _ := dest.listing(l.Good); dl.Sell > cur.Sell {
				dest = st
			}
		}
	}
	qty := min(p.Credits/(2*l.Buy), s.Capacity) // prices rise while buying
	if err := w.Buy(p.ID, s.ID, l.Good, qty, 0); err != nil {
		t.Fatal(err)
	}
	if err := w.Travel(p.ID, s.ID, dest.X, dest.Y, 0, false); err != nil {
		t.Fatal(err)
	}
	advanceUntil(t, w, s)
	if err := w.Sell(p.ID, s.ID, l.Good, qty, float64(w.Tick)); err != nil {
		t.Fatal(err)
	}
	if p.Credits <= StartingCredits {
		t.Fatalf("round trip %s → %s ended with %d credits", s.At, dest.ID, p.Credits)
	}
}

func TestTradeResolvesArrival(t *testing.T) {
	w := NewWorld(1)
	p, s, l := dockedTrade(t, w, "Alice")
	home, _ := w.Station(s.At)
	x, y := openPoint(t, w)
	if err := w.Travel(p.ID, s.ID, x, y, 0, false); err != nil {
		t.Fatal(err)
	}
	advanceUntil(t, w, s)
	if err := w.Travel(p.ID, s.ID, home.X, home.Y, float64(w.Tick), false); err != nil {
		t.Fatal(err)
	}
	// Arrived between ticks, before Advance resolved it.
	if err := w.Buy(p.ID, s.ID, l.Good, 1, s.ArriveAt+0.1); err != nil {
		t.Fatalf("trade right after arrival: %v", err)
	}
}

func TestTradeErrors(t *testing.T) {
	w := NewWorld(1)
	p, s, l := dockedTrade(t, w, "Alice")
	bob, _ := newPlayer(t, w, "Bob")
	st, _ := w.Station(s.At)

	var notListed, needOnly GoodID
	for _, g := range w.Goods {
		gl, ok := st.listing(g.ID)
		if !ok && notListed == "" {
			notListed = g.ID
		}
		if ok && gl.Buy == 0 && needOnly == "" {
			needOnly = g.ID
		}
	}
	if notListed == "" || needOnly == "" {
		t.Fatalf("station %s has no unlisted or need-only good", st.ID)
	}

	type trade func(PlayerID, ShipID, GoodID, int, float64) error
	cases := []struct {
		name  string
		setup func()
		do    trade
		pid   PlayerID
		ship  ShipID
		good  GoodID
		qty   int
		want  error
	}{
		{"unknown ship", nil, w.Buy, p.ID, "s99", l.Good, 1, ErrUnknownShip},
		{"not your ship", nil, w.Buy, bob.ID, s.ID, l.Good, 1, ErrNotYourShip},
		{"zero quantity", nil, w.Buy, p.ID, s.ID, l.Good, 0, ErrBadQuantity},
		{"negative quantity", nil, w.Sell, p.ID, s.ID, l.Good, -1, ErrBadQuantity},
		{"unknown good", nil, w.Buy, p.ID, s.ID, "gold", 1, ErrUnknownGood},
		{"not listed", nil, w.Buy, p.ID, s.ID, notListed, 1, ErrNotTradedHere},
		{"need-only good", nil, w.Buy, p.ID, s.ID, needOnly, 1, ErrNotForSale},
		{"selling more than held", nil, w.Sell, p.ID, s.ID, l.Good, 1, ErrNotEnoughCargo},
		{"not enough credits", func() { p.Credits = l.Buy - 1 }, w.Buy, p.ID, s.ID, l.Good, 1, ErrNoCredits},
		{"cargo full", func() { p.Credits = 1 << 20; s.Cargo[l.Good] = s.Capacity }, w.Buy, p.ID, s.ID, l.Good, 1, ErrCargoFull},
		{"travelling", func() {
			if err := w.Travel(p.ID, s.ID, 500, 500, float64(w.Tick), false); err != nil {
				t.Fatal(err)
			}
		}, w.Sell, p.ID, s.ID, l.Good, 1, ErrNotDocked},
		{"idle in open space", func() { advanceUntil(t, w, s) }, w.Sell, p.ID, s.ID, l.Good, 1, ErrNotDocked},
	}
	for _, c := range cases {
		if c.setup != nil {
			c.setup()
		}
		credits, cargo := p.Credits, maps.Clone(s.Cargo)
		if err := c.do(c.pid, c.ship, c.good, c.qty, float64(w.Tick)); !errors.Is(err, c.want) {
			t.Errorf("%s: got %v, want %v", c.name, err, c.want)
		}
		if p.Credits != credits || !maps.Equal(s.Cargo, cargo) {
			t.Errorf("%s: failed trade changed credits %d→%d or cargo %v→%v", c.name, credits, p.Credits, cargo, s.Cargo)
		}
	}
}

func TestBuyFillsCargoExactly(t *testing.T) {
	w := NewWorld(1)
	p, s, l := dockedTrade(t, w, "Alice")
	p.Credits = 1 << 20
	if err := w.Buy(p.ID, s.ID, l.Good, s.Capacity, 0); err != nil {
		t.Fatalf("filling the hold exactly: %v", err)
	}
	if err := w.Buy(p.ID, s.ID, l.Good, 1, 0); !errors.Is(err, ErrCargoFull) {
		t.Fatalf("one more unit: %v", err)
	}
}

func TestBuyRaisesPriceSellLowers(t *testing.T) {
	w := NewWorld(1)
	p, s, l := dockedTrade(t, w, "Alice")
	if err := w.Buy(p.ID, s.ID, l.Good, 5, 0); err != nil {
		t.Fatal(err)
	}
	got, _ := w.listingRef(s.At, l.Good)
	if got.Pressure != 5 || got.Buy <= l.Buy || got.Sell < l.Sell {
		t.Fatalf("after buying 5: %+v, before %+v", *got, l)
	}

	// Sell them to a station that only needs the good.
	for _, st := range w.Stations {
		if nl, ok := st.listing(l.Good); ok && nl.Buy == 0 {
			if err := w.Travel(p.ID, s.ID, st.X, st.Y, float64(w.Tick), false); err != nil {
				t.Fatal(err)
			}
			advanceUntil(t, w, s)
			before, _ := w.listingRef(st.ID, l.Good)
			sell := before.Sell
			if err := w.Sell(p.ID, s.ID, l.Good, 5, float64(w.Tick)); err != nil {
				t.Fatal(err)
			}
			if before.Sell >= sell || before.Pressure >= 0 {
				t.Fatalf("after selling 5 at %s: %+v, sell was %d", st.ID, *before, sell)
			}
			return
		}
	}
	t.Fatalf("no station only needs %s", l.Good)
}

func TestBulkEqualsSingleUnits(t *testing.T) {
	bulk, single := NewWorld(1), NewWorld(1)
	pb, sb, l := dockedTrade(t, bulk, "Alice")
	ps, ss, _ := dockedTrade(t, single, "Alice")
	if err := bulk.Buy(pb.ID, sb.ID, l.Good, 10, 0); err != nil {
		t.Fatal(err)
	}
	for range 10 {
		if err := single.Buy(ps.ID, ss.ID, l.Good, 1, 0); err != nil {
			t.Fatal(err)
		}
	}
	lb, _ := bulk.listingRef(sb.At, l.Good)
	ls, _ := single.listingRef(ss.At, l.Good)
	if pb.Credits != ps.Credits || *lb != *ls {
		t.Fatalf("bulk: %d credits %+v; single: %d credits %+v", pb.Credits, *lb, ps.Credits, *ls)
	}
}

func TestPressureDecays(t *testing.T) {
	w := NewWorld(1)
	p, s, l := dockedTrade(t, w, "Alice")
	if err := w.Buy(p.ID, s.ID, l.Good, 20, 0); err != nil {
		t.Fatal(err)
	}
	got, _ := w.listingRef(s.At, l.Good)
	for range PressureHalfLife {
		w.Advance()
	}
	if math.Abs(got.Pressure-10) > 0.1 {
		t.Fatalf("after one half-life: pressure %v, want ~10", got.Pressure)
	}
	for range 20 * PressureHalfLife {
		w.Advance()
		if got.Pressure == 0 {
			break
		}
	}
	if got.Pressure != 0 || got.Buy != got.BaseBuy || got.Sell != got.BaseSell {
		t.Fatalf("pressure never faded back to normal: %+v", *got)
	}
}
