// Shared helpers for positioning ships between server snapshots.
// Times are in ticks (floats); positions in map units.

const PALETTE = ["#4fc3f7", "#ffb74d", "#ba68c8", "#81c784", "#f06292", "#fff176", "#a1887f", "#90a4ae"];

export function playerColor(players, playerId) {
  const i = players.findIndex((p) => p.id === playerId);
  return PALETTE[(i < 0 ? 0 : i) % PALETTE.length];
}

/**
 * Current game time: the snapshot's time plus what has passed since it
 * arrived (capped at one tick, so ships don't run ahead if snapshots stop).
 */
export function tickNow(game, now) {
  const { state, receivedAt } = game;
  const elapsed = Math.min(Math.max((now - receivedAt) / state.tickMs, 0), 1);
  return state.time + elapsed;
}

// --- Speed profile of one leg. Mirrors backend/internal/game/motion.go: keep them identical. ---
// Accelerate from v0 to a peak (at most vmax), cruise, brake to v1 at the target
// (0 at a full stop, higher at a waypoint the ship flies through).

export function legSpeeds(d, v0, v1, a, vmax) {
  v1 = Math.min(Math.max(v1, 0), vmax);
  v0 = Math.min(Math.max(v0, 0), vmax, Math.sqrt(v1 * v1 + 2 * a * d));
  v1 = Math.min(v1, Math.sqrt(v0 * v0 + 2 * a * d));
  return [v0, v1];
}

export function legProfile(d, v0, v1, a, vmax) {
  if (d <= 0 || a <= 0 || vmax <= 0) return { d: Math.max(d, 0), v0: 0, v1: 0, a, peak: 0, t1: 0, t2: 0, total: 0, d1: 0 };
  [v0, v1] = legSpeeds(d, v0, v1, a, vmax);
  let peak = vmax;
  if ((2 * vmax * vmax - v0 * v0 - v1 * v1) / (2 * a) > d) peak = Math.sqrt((2 * a * d + v0 * v0 + v1 * v1) / 2);
  const t1 = (peak - v0) / a;
  const d1 = (peak * peak - v0 * v0) / (2 * a);
  const d3 = (peak * peak - v1 * v1) / (2 * a);
  const t2 = t1 + (peak > 0 ? Math.max(d - d1 - d3, 0) / peak : 0);
  return { d, v0, v1, a, peak, t1, t2, total: t2 + (peak - v1) / a, d1 };
}

/** Distance covered and speed dt ticks into the leg. */
export function profileAt(p, dt) {
  if (p.total <= 0) return { dist: p.d, speed: p.v1 };
  if (dt <= 0) return { dist: 0, speed: p.v0 };
  if (dt < p.t1) return { dist: p.v0 * dt + (p.a * dt * dt) / 2, speed: p.v0 + p.a * dt };
  if (dt < p.t2) return { dist: p.d1 + p.peak * (dt - p.t1), speed: p.peak };
  if (dt < p.total) {
    const r = p.total - dt;
    return { dist: p.d - p.v1 * r - (p.a * r * r) / 2, speed: p.v1 + p.a * r };
  }
  return { dist: p.d, speed: p.v1 };
}

/** Fastest the ship may pass (x, y) and still brake for its next station or the end of the route. */
export function routeEndSpeed(x, y, to, route, a, vmax) {
  if (to || route.length === 0) return 0;
  const legs = [];
  for (const wp of route) {
    legs.push(Math.hypot(wp.x - x, wp.y - y));
    x = wp.x;
    y = wp.y;
    if (wp.to) break;
  }
  let v = 0;
  for (let i = legs.length - 1; i >= 0; i--) v = Math.min(vmax, Math.sqrt(v * v + 2 * a * legs[i]));
  return v;
}

// --- Ships ---

/**
 * The ship's current leg followed by the legs the server will start for its
 * queued stops (same math as Ship.startLeg/advance), so ships fly on through
 * a waypoint without waiting for the next snapshot.
 */
export function shipLegs(ship) {
  if (ship.status !== "travelling") return [];
  const leg = (x, y, toX, toY, v0, v1, departAt, arriveAt) => {
    const d = Math.hypot(toX - x, toY - y);
    return { x, y, toX, toY, d, departAt, arriveAt, p: legProfile(d, v0, v1, ship.accel, ship.speed) };
  };
  const legs = [leg(ship.x, ship.y, ship.toX, ship.toY, ship.v0, ship.v1, ship.departAt, ship.arriveAt)];
  const route = ship.route ?? [];
  route.forEach((wp, i) => {
    const prev = legs[legs.length - 1];
    const d = Math.hypot(wp.x - prev.toX, wp.y - prev.toY);
    const end = routeEndSpeed(wp.x, wp.y, wp.to, route.slice(i + 1), ship.accel, ship.speed);
    const [v0, v1] = legSpeeds(d, prev.p.v1, end, ship.accel, ship.speed);
    const duration = legProfile(d, v0, v1, ship.accel, ship.speed).total;
    legs.push(leg(prev.toX, prev.toY, wp.x, wp.y, v0, v1, prev.arriveAt, prev.arriveAt + duration));
  });
  return legs;
}

/**
 * Where the ship is at time t: position, speed (map units per tick), the
 * direction it flies in (null at rest) and the index of its leg in shipLegs.
 */
export function shipState(ship, t, legs = shipLegs(ship)) {
  if (legs.length === 0) return { x: ship.x, y: ship.y, speed: 0, angle: null, leg: 0 };
  let i = 0;
  while (i < legs.length - 1 && t >= legs[i].arriveAt) i++;
  const l = legs[i];
  const angle = Math.atan2(l.toY - l.y, l.toX - l.x);
  if (l.d === 0) return { x: l.toX, y: l.toY, speed: 0, angle, leg: i };
  const { dist, speed } = profileAt(l.p, t - l.departAt);
  const f = dist / l.d;
  return { x: l.x + (l.toX - l.x) * f, y: l.y + (l.toY - l.y) * f, speed, angle, leg: i };
}

/** Seconds until the ship reaches the end of its route. */
export function routeEtaSeconds(ship, t, tickMs) {
  const legs = shipLegs(ship);
  if (legs.length === 0) return 0;
  return Math.ceil((Math.max(0, legs[legs.length - 1].arriveAt - t) * tickMs) / 1000);
}

// How close a ship must be to plunder an AI trader; same as game.PlunderRadius.
export const PLUNDER_RADIUS = 40;

/**
 * Travelling AI traders within plunder range of ship at time t. None while
 * the ship is docked (no plundering at stations). The server re-checks.
 */
export function tradersInRange(ships, ship, t) {
  if (!ship || ship.status === "docked") return [];
  const me = shipState(ship, t);
  return ships.filter((s) => {
    if (!s.npc || s.status !== "travelling") return false;
    const p = shipState(s, t);
    return Math.hypot(p.x - me.x, p.y - me.y) <= PLUNDER_RADIUS;
  });
}
