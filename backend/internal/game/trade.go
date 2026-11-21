package game

// Buy loads qty units of good onto a docked ship, paid from its owner's
// credits at the station's price; each unit raises the price for the next
// (see Listing.BuyCost). Sell is the reverse. Both resolve arrivals up to now
// (in ticks) first, and happen in full or not at all.
func (w *World) Buy(playerID PlayerID, shipID ShipID, good GoodID, qty int, now float64) error {
	s, p, l, err := w.tradeAt(playerID, shipID, good, qty, now)
	if err != nil {
		return err
	}
	if l.Buy == 0 {
		return ErrNotForSale
	}
	// Checked before pricing, so qty is small and the cost can't overflow.
	if s.CargoUsed()+qty > s.Capacity {
		return ErrCargoFull
	}
	cost := l.BuyCost(qty)
	if p.Credits < cost {
		return ErrNoCredits
	}
	p.Credits -= cost
	s.Cargo[good] += qty
	l.push(float64(qty))
	return nil
}

// Sell unloads qty units of good from a docked ship for the station's price.
func (w *World) Sell(playerID PlayerID, shipID ShipID, good GoodID, qty int, now float64) error {
	s, p, l, err := w.tradeAt(playerID, shipID, good, qty, now)
	if err != nil {
		return err
	}
	if s.Cargo[good] < qty {
		return ErrNotEnoughCargo
	}
	p.Credits += l.SellValue(qty)
	l.push(-float64(qty))
	s.Cargo[good] -= qty
	if s.Cargo[good] == 0 {
		delete(s.Cargo, good)
	}
	return nil
}

// tradeAt checks what Buy and Sell share and returns the ship, its owner and
// the station's live listing for good.
func (w *World) tradeAt(playerID PlayerID, shipID ShipID, good GoodID, qty int, now float64) (*Ship, *Player, *Listing, error) {
	s, ok := w.Ships[shipID]
	if !ok {
		return nil, nil, nil, ErrUnknownShip
	}
	if s.OwnerID != playerID {
		return nil, nil, nil, ErrNotYourShip
	}
	if qty < 1 {
		return nil, nil, nil, ErrBadQuantity
	}
	if _, ok := w.Good(good); !ok {
		return nil, nil, nil, ErrUnknownGood
	}
	s.advance(now) // a ship that arrived since the last tick can trade right away
	if s.Status != Docked {
		return nil, nil, nil, ErrNotDocked
	}
	l, ok := w.listingRef(s.At, good)
	if !ok {
		return nil, nil, nil, ErrNotTradedHere
	}
	p, ok := w.Players[s.OwnerID]
	if !ok {
		return nil, nil, nil, ErrUnknownPlayer
	}
	return s, p, l, nil
}
