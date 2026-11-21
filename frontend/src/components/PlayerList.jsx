import { playerColor } from "../geometry.js";

export default function PlayerList({ players, me }) {
  return (
    <section className="panel">
      <h2>Players</h2>
      <ul className="players">
        {players.map((p) => (
          <li key={p.id}>
            <span className="swatch" style={{ background: playerColor(players, p.id) }} />
            <span className={p.online ? "name" : "name offline"}>
              {p.name}
              {p.id === me && " (you)"}
            </span>
            <span className={p.online ? "dot on" : "dot"} title={p.online ? "online" : "offline"} />
          </li>
        ))}
      </ul>
    </section>
  );
}
