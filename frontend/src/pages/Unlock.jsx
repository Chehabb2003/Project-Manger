// src/pages/Unlock.jsx
import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { unlock } from "../lib/api";
import { useAuth } from "../context/AuthContext";

export default function Unlock() {
  const [vault, setVault] = useState("./dev.vlt");
  const [master, setMaster] = useState("");
  const [loading, setLoading] = useState(false);
  const [err, setErr] = useState("");
  const navigate = useNavigate();
  const { setUnlocked } = useAuth();

  async function onSubmit(e) {
    e.preventDefault();
    setErr(""); setLoading(true);
    try {
      const res = await unlock(vault, master);
      if (!res?.ok) throw new Error("Unlock failed");
      setUnlocked({ ok: true, vault: res.vault || vault });
      navigate("/vault");
    } catch (e2) {
      setErr(e2.message || "Failed to unlock");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div style={{ maxWidth: 480, margin: "40px auto" }}>
      <h1>Unlock Vault</h1>
      <p style={{ opacity: 0.7 }}>If you see this, routing & render are OK ✅</p>
      <form onSubmit={onSubmit} style={{ display: "grid", gap: 12 }}>
        <label>Vault file <input value={vault} onChange={e => setVault(e.target.value)} /></label>
        <label>Master <input type="password" value={master} onChange={e => setMaster(e.target.value)} /></label>
        {err && <div style={{ color: "crimson" }}>{err}</div>}
        <button disabled={loading || !master}>{loading ? "Unlocking…" : "Unlock"}</button>
      </form>
    </div>
  );
}
