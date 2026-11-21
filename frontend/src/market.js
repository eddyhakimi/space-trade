// Market pricing, mirrored from backend/internal/game/market.go: change both
// together. The server is authoritative; the client uses this only to show
// totals and to keep Max from asking for more than you can afford.

const PRICE_IMPACT = 0.01; // each unit of pressure moves a price by e^0.01 (≈1 %)
const MAX_PRESSURE = 100;
const BUYBACK_PERCENT = 80;
const TREND_THRESHOLD = 0.05; // show ▲/▼ when a price is ≥5 % off normal

const priceAt = (base, p) => Math.max(1, Math.round(base * Math.exp(PRICE_IMPACT * p)));

// Current prices of a listing at pressure p (same rules as Listing.reprice).
function pricesAt(l, p) {
  if (l.baseBuy) {
    const buy = priceAt(l.baseBuy, p);
    return { buy, sell: Math.max(1, Math.floor((buy * BUYBACK_PERCENT) / 100)) };
  }
  return { buy: 0, sell: priceAt(l.baseSell, p) };
}

const clamp = (p) => Math.min(Math.max(p, -MAX_PRESSURE), MAX_PRESSURE);

// What buying qty units costs, unit by unit as the price rises (Listing.BuyCost).
export function buyCost(l, qty) {
  let p = l.pressure;
  let total = 0;
  for (let i = 0; i < qty; i++) {
    total += pricesAt(l, p).buy;
    p = clamp(p + 1);
  }
  return total;
}

// What selling qty units pays, unit by unit as the price falls (Listing.SellValue).
export function sellValue(l, qty) {
  let p = l.pressure;
  let total = 0;
  for (let i = 0; i < qty; i++) {
    total += pricesAt(l, p).sell;
    p = clamp(p - 1);
  }
  return total;
}

// The most units you can buy with credits and free cargo space: { qty, cost }.
export function maxBuy(l, credits, free) {
  let p = l.pressure;
  let qty = 0;
  let cost = 0;
  while (qty < free) {
    const price = pricesAt(l, p).buy;
    if (!price || cost + price > credits) break;
    cost += price;
    qty += 1;
    p = clamp(p + 1);
  }
  return { qty, cost };
}

// "up" / "down" when price is clearly above / below normal, else null.
export function trend(price, base) {
  if (!price || !base) return null;
  if (price >= base * (1 + TREND_THRESHOLD)) return "up";
  if (price <= base * (1 - TREND_THRESHOLD)) return "down";
  return null;
}
