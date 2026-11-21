import { useCallback, useEffect, useRef, useState } from "react";

const WS_URL = `${location.protocol === "https:" ? "wss" : "ws"}://${location.host}/api/v1/ws`;
const RECONNECT_MS = 2000;
// An automatic reconnect whose join is rejected (e.g. the server hasn't noticed
// our old connection is gone yet) is retried this often before giving up.
const MAX_REJOIN_ATTEMPTS = 5;

// ?slot=2 keeps a separate session, so one browser can play several players
// (handy for local testing).
const slot = new URLSearchParams(location.search).get("slot");
const SESSION_KEY = "spacetrade.session" + (slot ? `.${slot}` : "");

export function loadSession() {
  try {
    return JSON.parse(localStorage.getItem(SESSION_KEY));
  } catch {
    return null;
  }
}

export function saveSession(session) {
  try {
    localStorage.setItem(SESSION_KEY, JSON.stringify(session));
  } catch {
    /* storage unavailable: the session just won't survive a reload */
  }
}

export function clearSession() {
  try {
    localStorage.removeItem(SESSION_KEY);
  } catch {
    /* ignore */
  }
}

/**
 * Connects to the game, joins with `session` ({name, token?}) and keeps the
 * latest state. Reconnects automatically after the connection drops.
 * `onJoinFailed(msg)` is called when the server rejects the join: right away
 * for a fresh join from the join screen, after MAX_REJOIN_ATTEMPTS for
 * automatic reconnects.
 */
export function useGameSocket(session, onJoinFailed) {
  const [status, setStatus] = useState("connecting"); // connecting | open | reconnecting
  const [game, setGame] = useState(null); // { state, receivedAt }
  const [me, setMe] = useState(null); // playerId
  const [error, setError] = useState(null); // { msg, at } from rejected commands
  const wsRef = useRef(null);
  const onJoinFailedRef = useRef(onJoinFailed);
  onJoinFailedRef.current = onJoinFailed;

  useEffect(() => {
    if (!session) return;
    // The token can change on welcome; keep it here so reconnects use the latest one
    // without re-running this effect.
    const current = { ...session };
    let ws;
    let retryTimer;
    let disposed = false;
    let attempt = 0; // connections opened so far
    let rejected = 0; // join rejections in a row

    const connect = () => {
      let welcomed = false;
      let joinFailed = false;
      attempt += 1;
      // Another tab may have joined since (e.g. after a server restart) and
      // saved a newer token for the same captain: use it.
      const stored = loadSession();
      if (stored?.name === current.name && stored.token) current.token = stored.token;
      ws = new WebSocket(WS_URL);
      wsRef.current = ws;

      ws.onopen = () => {
        ws.send(JSON.stringify({ type: "join", name: current.name, token: current.token ?? "" }));
      };

      ws.onmessage = (e) => {
        const m = JSON.parse(e.data);
        switch (m.type) {
          case "welcome":
            welcomed = true;
            rejected = 0;
            current.token = m.token;
            saveSession({ name: current.name, token: m.token });
            setMe(m.playerId);
            setStatus("open");
            break;
          case "state":
            setGame({ state: m, receivedAt: performance.now() });
            break;
          case "error":
            if (!welcomed) {
              // A fresh join from the join screen fails right away (e.g. name taken);
              // automatic reconnects retry a few times first.
              const freshJoin = attempt === 1 && !session.token;
              rejected += 1;
              if (freshJoin || rejected >= MAX_REJOIN_ATTEMPTS) {
                joinFailed = true;
                onJoinFailedRef.current?.(m.msg);
              }
            } else {
              setError({ msg: m.msg, at: Date.now() });
            }
            break;
        }
      };

      ws.onclose = () => {
        if (disposed || joinFailed) return;
        setStatus("reconnecting");
        retryTimer = setTimeout(connect, RECONNECT_MS);
      };
    };

    setStatus("connecting");
    connect();

    return () => {
      disposed = true;
      clearTimeout(retryTimer);
      ws.close();
    };
  }, [session]);

  // queue: add (x, y) as a stop after the current route instead of flying there now.
  const travel = useCallback((shipId, x, y, queue = false) => {
    const ws = wsRef.current;
    if (ws?.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({ type: "travel", shipId, x, y, queue }));
    }
  }, []);

  return { status, game, me, error, travel };
}
