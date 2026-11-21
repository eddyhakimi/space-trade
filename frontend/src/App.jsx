import { useEffect, useState } from "react";
import EchoPanel from "./EchoPanel.jsx";

export default function App() {
  const [status, setStatus] = useState("checking");

  useEffect(() => {
    const check = () =>
      fetch("/api/v1/health")
        .then((res) => (res.ok ? res.json() : Promise.reject()))
        .then((data) => setStatus(data.status === "alive" ? "alive" : "down"))
        .catch(() => setStatus("down"));

    check();
    const id = setInterval(check, 5000);
    return () => clearInterval(id);
  }, []);

  const colors = { checking: "#999", alive: "#2e9d4f", down: "#d33" };
  const labels = {
    checking: "Checking backend…",
    alive: "Backend is alive",
    down: "Backend is unreachable",
  };

  return (
    <main style={{ fontFamily: "system-ui, sans-serif", padding: "3rem", textAlign: "center" }}>
      <h1>Space Trade</h1>
      <p style={{ fontSize: "1.5rem", color: colors[status] }}>
        <span style={{ marginRight: "0.5rem" }}>●</span>
        {labels[status]}
      </p>
      <EchoPanel />
    </main>
  );
}
