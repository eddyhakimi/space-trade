import { tickNow, tradersInRange } from "../geometry.js";
import { useClock } from "../hooks.js";

// AI traders close enough to plunder. Shown only while your ship is in open
// space and a travelling trader is within range; the server re-checks. Their
// cargo is unknown until you plunder (the server doesn't send it).
export default function PlunderPanel({ game, me, onPlunder }) {
  const now = useClock(250);
  const { state } = game;
  const ship = state.ships.find((s) => s.ownerId === me);
  const targets = tradersInRange(state.ships, ship, tickNow(game, now));
  if (targets.length === 0) return null;
  const free = ship.capacity - Object.values(ship.cargo ?? {}).reduce((a, b) => a + b, 0);

  return (
    <section className="panel plunder">
      <h2>Traders in range</h2>
      <ul className="plunder-list">
        {targets.map((tr) => (
          <li key={tr.id}>
            <span className="plunder-name">{tr.name}</span>
            <span className="plunder-cargo">cargo unknown</span>
            <button
              disabled={free < 1}
              title={free < 1 ? "hold full" : `take up to ${free} units`}
              onClick={() => onPlunder(tr.id)}
            >
              Plunder
            </button>
          </li>
        ))}
      </ul>
    </section>
  );
}
