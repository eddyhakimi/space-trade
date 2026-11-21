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

/** Where the ship is at time t. */
export function shipPosition(ship, t) {
  if (ship.status !== "travelling" || ship.arriveAt <= ship.departAt) return { x: ship.x, y: ship.y };
  const p = Math.min(Math.max((t - ship.departAt) / (ship.arriveAt - ship.departAt), 0), 1);
  return { x: ship.x + (ship.toX - ship.x) * p, y: ship.y + (ship.toY - ship.y) * p };
}

/** Seconds until arrival. */
export function etaSeconds(ship, t, tickMs) {
  return Math.max(0, Math.ceil(((ship.arriveAt - t) * tickMs) / 1000));
}
