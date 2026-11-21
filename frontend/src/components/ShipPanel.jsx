import { routeEtaSeconds, shipState, tickNow } from "../geometry.js";
import { useClock } from "../hooks.js";

// Fill bar plus one row per good carried (catalog order).
function CargoHold({ state, ship }) {
  const items = state.goods.filter((g) => ship.cargo?.[g.id] > 0);
  const used = items.reduce((n, g) => n + ship.cargo[g.id], 0);
  const pct = ship.capacity > 0 ? Math.min(100, (used / ship.capacity) * 100) : 0;
  const dockedAt = ship.status === "docked" ? ship.at : null;

  return (
    <div className="cargo">
      <div className="cargo-head">
        <span>Cargo hold</span>
        <span className={used >= ship.capacity ? "cargo-count full" : "cargo-count"}>
          {used} / {ship.capacity}
        </span>
      </div>
      <div
        className="cargo-bar"
        role="meter"
        aria-label="Cargo used"
        aria-valuemin={0}
        aria-valuemax={ship.capacity}
        aria-valuenow={used}
      >
        <div className={used >= ship.capacity ? "fill full" : "fill"} style={{ width: `${pct}%` }} />
      </div>
      {items.length === 0 ? (
        <p className="cargo-empty">
          {dockedAt ? "Empty — buy goods in the market below." : "Empty — dock at a station to buy goods."}
        </p>
      ) : (
        <ul className="cargo-list">
          {items.map((g) => (
            <li key={g.id}>
              <span className="cargo-name">{g.name}</span>
              <span className="cargo-qty">× {ship.cargo[g.id]}</span>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

const coords = (x, y) => `(${Math.round(x)}, ${Math.round(y)})`;

export default function ShipPanel({ game, me }) {
  const now = useClock(250);
  const { state } = game;
  const ship = state.ships.find((s) => s.ownerId === me);
  const stationName = (id) => state.stations.find((s) => s.id === id)?.name ?? id;

  if (!ship) return null;
  const t = tickNow(game, now);

  let where;
  if (ship.status === "docked") {
    where = (
      <p>
        Docked at <strong>{stationName(ship.at)}</strong>
      </p>
    );
  } else if (ship.status === "idle") {
    where = <p>Holding position at {coords(ship.x, ship.y)}</p>;
  } else {
    const pos = shipState(ship, t);
    const more = ship.route?.length ?? 0;
    where = (
      <p>
        En route to <strong>{ship.to ? stationName(ship.to) : coords(ship.toX, ship.toY)}</strong>
        {more > 0 && ` · then ${more} more stop${more > 1 ? "s" : ""}`}
        <br />
        Now at {coords(pos.x, pos.y)} · {Math.round((pos.speed * 1000) / state.tickMs)} u/s
        <br />
        ETA {routeEtaSeconds(ship, t, state.tickMs)} s
      </p>
    );
  }

  return (
    <section className="panel">
      <h2>Your ship</h2>
      <p className="ship-name">{ship.name}</p>
      {where}
      <CargoHold state={state} ship={ship} />
      <p className="hint">
        Click the map to fly, a station to dock and trade. Shift+click adds a stop.
      </p>
    </section>
  );
}
