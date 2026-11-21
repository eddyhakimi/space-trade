import { useEffect, useState } from "react";
import { routeEtaSeconds, shipState, tickNow } from "../geometry.js";

// Re-render a few times per second so the ETA counts down between snapshots.
function useClock(ms) {
  const [now, setNow] = useState(() => performance.now());
  useEffect(() => {
    const id = setInterval(() => setNow(performance.now()), ms);
    return () => clearInterval(id);
  }, [ms]);
  return now;
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
      <p className="hint">
        Click anywhere on the map to fly there — click a station to dock. Shift+click to add a stop to your route.
      </p>
    </section>
  );
}
