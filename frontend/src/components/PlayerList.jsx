import { playerColor } from "../geometry.js";
import { formatHeat } from "../market.js";

// Leaderboard: players ranked by credits, richest first (ties by name).
export default function PlayerList({ players, me, tickMs }) {
  const ranked = [...players].sort((a, b) => b.credits - a.credits || a.name.localeCompare(b.name));

  return (
    <section className="panel">
      <h2>Leaderboard</h2>
      <ol className="players">
        {ranked.map((p, i) => (
          <li key={p.id} className={p.id === me ? "me" : undefined}>
            <span className="rank">{i + 1}</span>
            {/* colors follow the server's player order, so they match the ships on the map */}
            <span className="swatch" style={{ background: playerColor(players, p.id) }} />
            <span className={p.online ? "name" : "name offline"}>
              {p.name}
              {p.id === me && " (you)"}
              {p.heat > 0 && (
                <span className="wanted" title={`wanted for ${formatHeat(p.heat, tickMs)}`}>
                  {" "}
                  ☠
                </span>
              )}
            </span>
            <span className="credits" title="credits">
              {p.credits.toLocaleString()}
            </span>
            <span className={p.online ? "dot on" : "dot"} title={p.online ? "online" : "offline"} />
          </li>
        ))}
      </ol>
    </section>
  );
}
