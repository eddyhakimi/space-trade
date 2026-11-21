package game

import "testing"

// The price table is hand-written; these invariants keep future edits sane.

func TestGoodsCatalog(t *testing.T) {
	w := NewWorld(1)
	if len(w.Goods) != 10 {
		t.Fatalf("got %d goods, want 10", len(w.Goods))
	}
	seen := map[GoodID]bool{}
	for _, g := range w.Goods {
		if g.ID == "" || g.Name == "" || seen[g.ID] {
			t.Errorf("bad or duplicate good %+v", g)
		}
		seen[g.ID] = true
	}
}

func TestMarketListings(t *testing.T) {
	w := NewWorld(1)
	for _, st := range w.Stations {
		if len(st.Market) == 0 || len(st.Market) >= len(w.Goods) {
			t.Errorf("%s trades %d of %d goods, want some but not all", st.ID, len(st.Market), len(w.Goods))
		}
		seen := map[GoodID]bool{}
		for _, l := range st.Market {
			if _, ok := w.Good(l.Good); !ok {
				t.Errorf("%s lists unknown good %q", st.ID, l.Good)
			}
			if seen[l.Good] {
				t.Errorf("%s lists %s twice", st.ID, l.Good)
			}
			seen[l.Good] = true
			if l.Sell <= 0 {
				t.Errorf("%s %s: sell price %d, want > 0", st.ID, l.Good, l.Sell)
			}
			if l.Buy > 0 && l.Sell >= l.Buy {
				t.Errorf("%s %s: buys back at %d, sells at %d (free money)", st.ID, l.Good, l.Sell, l.Buy)
			}
		}
	}
}

func TestEveryGoodHasAProfitableRoute(t *testing.T) {
	w := NewWorld(1)
	for _, g := range w.Goods {
		cheapest, best := 0, 0
		for _, st := range w.Stations {
			l, ok := st.listing(g.ID)
			if !ok {
				continue
			}
			if l.Buy > 0 && (cheapest == 0 || l.Buy < cheapest) {
				cheapest = l.Buy
			}
			best = max(best, l.Sell)
		}
		if cheapest == 0 {
			t.Errorf("%s is not for sale anywhere", g.ID)
		} else if best <= cheapest {
			t.Errorf("%s: cheapest %d, best sell %d: no profit possible", g.ID, cheapest, best)
		}
	}
}

// At normal prices, buying from one producer and selling to another loses money.
func TestNoProfitBetweenProducers(t *testing.T) {
	w := NewWorld(1)
	for _, a := range w.Stations {
		for _, b := range w.Stations {
			if a.ID == b.ID {
				continue
			}
			for _, la := range a.Market {
				lb, ok := b.listing(la.Good)
				if la.Buy > 0 && ok && lb.Buy > 0 && lb.Sell >= la.Buy {
					t.Errorf("%s: buy at %s for %d, sell at %s for %d", la.Good, a.ID, la.Buy, b.ID, lb.Sell)
				}
			}
		}
	}
}

func TestRepriceInvariants(t *testing.T) {
	w := NewWorld(1)
	for _, st := range w.Stations {
		for _, base := range st.Market {
			if base.Buy != base.BaseBuy || base.Sell != base.BaseSell || base.Pressure != 0 {
				t.Errorf("%s %s: fresh listing not at normal prices: %+v", st.ID, base.Good, base)
			}
			prev := Listing{}
			for _, p := range []float64{-MaxPressure, -40, 0, 40, MaxPressure} {
				l := base
				l.Pressure = p
				l.reprice()
				if l.Sell < 1 {
					t.Errorf("%s %s at %v: sell %d", st.ID, l.Good, p, l.Sell)
				}
				if l.BaseBuy > 0 && l.Sell >= l.Buy {
					t.Errorf("%s %s at %v: buys back at %d, sells at %d", st.ID, l.Good, p, l.Sell, l.Buy)
				}
				if prev.Sell > 0 && (l.Buy < prev.Buy || l.Sell < prev.Sell) {
					t.Errorf("%s %s: prices fell as pressure rose to %v: %+v after %+v", st.ID, l.Good, p, l, prev)
				}
				if p == 0 && (l.Buy != base.BaseBuy || l.Sell != base.BaseSell) {
					t.Errorf("%s %s at 0: %+v, want normal prices", st.ID, l.Good, l)
				}
				prev = l
			}
		}
	}
}

func TestPressureClamped(t *testing.T) {
	l := produces("ore", 20)
	l.push(500)
	if l.Pressure != MaxPressure {
		t.Fatalf("pressure %v, want %v", l.Pressure, MaxPressure)
	}
	l.push(-1000)
	if l.Pressure != -MaxPressure {
		t.Fatalf("pressure %v, want %v", l.Pressure, -MaxPressure)
	}
}
