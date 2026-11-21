import { useEffect, useMemo, useRef, useState } from "react";
import { playerColor, shipPosition, tickNow } from "../geometry.js";

const SIZE = 1000;
const STATION_R = 14;
const SNAP_RADIUS = 40; // same as game.SnapRadius on the server (hover hint only)
const DOCK_RING = 34;

// Static background stars, generated once with a tiny deterministic PRNG.
const STARS = (() => {
  let seed = 7;
  const rnd = () => ((seed = (seed * 16807) % 2147483647) / 2147483647);
  return Array.from({ length: 120 }, () => ({ x: rnd() * SIZE, y: rnd() * SIZE, r: rnd() * 1.6 + 0.3 }));
})();

// Re-render every animation frame so travelling ships move smoothly.
function useNow() {
  const [now, setNow] = useState(() => performance.now());
  useEffect(() => {
    let id;
    const loop = (t) => {
      setNow(t);
      id = requestAnimationFrame(loop);
    };
    id = requestAnimationFrame(loop);
    return () => cancelAnimationFrame(id);
  }, []);
  return now;
}

// Converts a mouse event to map coordinates (viewBox units).
function toMapPoint(svg, e) {
  const pt = svg.createSVGPoint();
  pt.x = e.clientX;
  pt.y = e.clientY;
  const p = pt.matrixTransform(svg.getScreenCTM().inverse());
  return { x: Math.min(Math.max(p.x, 0), SIZE), y: Math.min(Math.max(p.y, 0), SIZE) };
}

function ShipMarker({ x, y, angle, color, mine, label }) {
  const s = mine ? 13 : 10;
  return (
    <g transform={`translate(${x} ${y}) rotate(${(angle * 180) / Math.PI})`} pointerEvents="none">
      <polygon
        points={`${s},0 ${-s * 0.7},${s * 0.6} ${-s * 0.4},0 ${-s * 0.7},${-s * 0.6}`}
        fill={color}
        stroke={mine ? "#fff" : "none"}
        strokeWidth={2}
      >
        <title>{label}</title>
      </polygon>
    </g>
  );
}

function TargetMarker({ x, y, color }) {
  return (
    <g transform={`translate(${x} ${y})`} stroke={color} strokeWidth={2} fill="none" pointerEvents="none">
      <circle r={9} />
      <line x1={-15} y1={0} x2={-5} y2={0} />
      <line x1={5} y1={0} x2={15} y2={0} />
      <line x1={0} y1={-15} x2={0} y2={-5} />
      <line x1={0} y1={5} x2={0} y2={15} />
    </g>
  );
}

export default function StarMap({ game, me, onMapClick }) {
  const svgRef = useRef(null);
  const now = useNow();
  const [hoverStation, setHoverStation] = useState(null);
  const { state } = game;
  const t = tickNow(game, now);
  const stations = useMemo(() => Object.fromEntries(state.stations.map((s) => [s.id, s])), [state.stations]);
  const ownerName = Object.fromEntries(state.players.map((p) => [p.id, p.name]));

  // Spread docked ships around their station.
  const dockedAt = {};
  for (const ship of state.ships) {
    if (ship.status === "docked") (dockedAt[ship.at] ??= []).push(ship.id);
  }

  const shipPose = (ship) => {
    if (ship.status === "docked") {
      const st = stations[ship.at];
      const group = dockedAt[ship.at];
      const angle = -Math.PI / 2 + (group.indexOf(ship.id) * 2 * Math.PI) / group.length;
      return { x: st.x + Math.cos(angle) * DOCK_RING, y: st.y + Math.sin(angle) * DOCK_RING, angle };
    }
    const { x, y } = shipPosition(ship, t);
    const angle = ship.status === "travelling" ? Math.atan2(ship.toY - ship.y, ship.toX - ship.x) : -Math.PI / 2;
    return { x, y, angle };
  };

  const handleClick = (e) => {
    const { x, y } = toMapPoint(svgRef.current, e);
    onMapClick(x, y);
  };

  // Highlight the station a click would dock at.
  const handleMove = (e) => {
    const p = toMapPoint(svgRef.current, e);
    const near = state.stations.find((s) => Math.hypot(s.x - p.x, s.y - p.y) <= SNAP_RADIUS);
    setHoverStation(near?.id ?? null);
  };

  const travelling = state.ships.filter((s) => s.status === "travelling");

  return (
    <svg
      ref={svgRef}
      className="starmap"
      viewBox={`0 0 ${SIZE} ${SIZE}`}
      role="img"
      aria-label="Sector map: click anywhere to fly there, click a station to dock"
      onClick={handleClick}
      onMouseMove={handleMove}
      onMouseLeave={() => setHoverStation(null)}
    >
      <rect width={SIZE} height={SIZE} fill="var(--space)" />
      {STARS.map((s, i) => (
        <circle key={i} cx={s.x} cy={s.y} r={s.r} fill="#fff" opacity={0.35} />
      ))}

      {/* routes: from the ship's current position to its target */}
      {travelling.map((s) => {
        const pos = shipPosition(s, t);
        const mine = s.ownerId === me;
        return (
          <line
            key={`route-${s.id}`}
            x1={pos.x}
            y1={pos.y}
            x2={s.toX}
            y2={s.toY}
            stroke={playerColor(state.players, s.ownerId)}
            strokeOpacity={mine ? 0.8 : 0.3}
            strokeDasharray="8 8"
            strokeWidth={2}
          />
        );
      })}

      {state.stations.map((st) => (
        <g key={st.id} className={st.id === hoverStation ? "station hover" : "station"}>
          <circle cx={st.x} cy={st.y} r={STATION_R} className="station-body" />
          <text x={st.x} y={st.y + STATION_R + 44} textAnchor="middle" className="station-label">
            {st.name}
          </text>
        </g>
      ))}

      {/* target marker for your own ship */}
      {travelling
        .filter((s) => s.ownerId === me && !s.to)
        .map((s) => (
          <TargetMarker key={`target-${s.id}`} x={s.toX} y={s.toY} color={playerColor(state.players, me)} />
        ))}

      {/* own ship last so it is drawn on top */}
      {[...state.ships]
        .sort((a, b) => (a.ownerId === me) - (b.ownerId === me))
        .map((ship) => (
          <ShipMarker
            key={ship.id}
            {...shipPose(ship)}
            color={playerColor(state.players, ship.ownerId)}
            mine={ship.ownerId === me}
            label={`${ship.name} (${ownerName[ship.ownerId] ?? "?"})`}
          />
        ))}
    </svg>
  );
}
