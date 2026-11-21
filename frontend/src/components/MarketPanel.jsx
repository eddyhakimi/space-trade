import { maxBuy, sellValue, trend } from "../market.js";

// A price with ▲/▼ when it is clearly above/below the station's normal price.
function Price({ value, base }) {
  const t = trend(value, base);
  return (
    <span className="price" title={t ? `normal ${base}` : undefined}>
      {value}
      {t && <span className={`trend ${t}`}>{t === "up" ? "▲" : "▼"}</span>}
    </span>
  );
}

// The market of the station your ship is docked at. Prices move with every
// unit traded (see market.js); the server is authoritative, the limits here
// only disable buttons that would be rejected anyway.
export default function MarketPanel({ game, me, onBuy, onSell }) {
  const { state } = game;
  const ship = state.ships.find((s) => s.ownerId === me);
  const player = state.players.find((p) => p.id === me);
  if (!ship || !player || ship.status !== "docked") return null;
  const station = state.stations.find((s) => s.id === ship.at);
  if (!station) return null;

  const goodName = (id) => state.goods.find((g) => g.id === id)?.name ?? id;
  const used = Object.values(ship.cargo ?? {}).reduce((a, b) => a + b, 0);
  const free = ship.capacity - used;

  return (
    <section className="panel market">
      <h2>Market — {station.name}</h2>
      <table>
        <thead>
          <tr>
            <th>Good</th>
            <th>Hold</th>
            <th className="buy">Buy</th>
            <th className="sell">Sell</th>
          </tr>
        </thead>
        <tbody>
          {station.market.map((l) => {
            const held = ship.cargo?.[l.good] ?? 0;
            const max = l.buy ? maxBuy(l, player.credits, free) : { qty: 0, cost: 0 };
            return (
              <tr key={l.good}>
                <td>{goodName(l.good)}</td>
                <td>{held || ""}</td>
                <td className="buy">
                  {l.buy ? (
                    <>
                      <Price value={l.buy} base={l.baseBuy} />
                      <button disabled={max.qty < 1} onClick={() => onBuy(l.good, 1)} title={`Buy 1 for ${l.buy} cr`}>
                        1
                      </button>
                      <button
                        disabled={max.qty < 1}
                        onClick={() => onBuy(l.good, max.qty)}
                        title={`Buy ${max.qty} for ${max.cost} cr`}
                      >
                        Max
                      </button>
                    </>
                  ) : (
                    <span className="na" title="not for sale here">
                      —
                    </span>
                  )}
                </td>
                <td className="sell">
                  <Price value={l.sell} base={l.baseSell} />
                  <button disabled={held < 1} onClick={() => onSell(l.good, 1)} title={`Sell 1 for ${l.sell} cr`}>
                    1
                  </button>
                  <button
                    disabled={held < 1}
                    onClick={() => onSell(l.good, held)}
                    title={`Sell ${held} for ${sellValue(l, held)} cr`}
                  >
                    All
                  </button>
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
      <p className="hint">Prices move as you trade, then recover. Unlisted goods can't be traded here.</p>
    </section>
  );
}
