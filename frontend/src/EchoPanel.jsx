import { useEffect, useState } from "react";

// TEMP (M0): exercises the /api/v1/ws echo endpoint. Replaced by the game socket in M4.
const WS_URL = `${location.protocol === "https:" ? "wss" : "ws"}://${location.host}/api/v1/ws`;
const SEND_EVERY_MS = 1000;
const RECONNECT_MS = 2000;

export default function EchoPanel() {
  const [status, setStatus] = useState("connecting");
  const [last, setLast] = useState(null); // raw echoed text
  const [rtt, setRtt] = useState(null); // round-trip ms
  const [received, setReceived] = useState(0);

  useEffect(() => {
    let ws;
    let sendTimer;
    let reconnectTimer;
    let seq = 0;
    let disposed = false;

    const connect = () => {
      setStatus("connecting");
      ws = new WebSocket(WS_URL);

      ws.onopen = () => {
        setStatus("open");
        sendTimer = setInterval(() => {
          seq += 1;
          ws.send(JSON.stringify({ seq, sentAt: Date.now() }));
        }, SEND_EVERY_MS);
      };

      ws.onmessage = (e) => {
        setLast(e.data);
        setReceived((n) => n + 1);
        try {
          const { sentAt } = JSON.parse(e.data);
          setRtt(Date.now() - sentAt);
        } catch {
          setRtt(null);
        }
      };

      ws.onclose = () => {
        clearInterval(sendTimer);
        if (disposed) return;
        setStatus("closed");
        reconnectTimer = setTimeout(connect, RECONNECT_MS);
      };
    };

    connect();

    return () => {
      disposed = true;
      clearInterval(sendTimer);
      clearTimeout(reconnectTimer);
      ws.close();
    };
  }, []);

  const colors = { connecting: "#999", open: "#2e9d4f", closed: "#d33" };

  return (
    <section style={{ marginTop: "2rem" }}>
      <h2 style={{ fontSize: "1.1rem" }}>WebSocket echo</h2>
      <p style={{ color: colors[status] }}>
        <span style={{ marginRight: "0.5rem" }}>●</span>
        {status}
      </p>
      <p>
        Last echo: <code>{last ?? "—"}</code>
      </p>
      <p>
        Round trip: {rtt == null ? "—" : `${rtt} ms`} · Echoes received: {received}
      </p>
    </section>
  );
}
