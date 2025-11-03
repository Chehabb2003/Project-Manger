// frontend/src/pages/Unlock.jsx
import { useState } from "react";
import { useNavigate } from "react-router-dom";
import Layout from "../components/Layout.jsx";
import api from "../lib/api";

export default function Unlock() {
  const [vault, setVault] = useState("./dev.vlt");
  const [master, setMaster] = useState("");
  const [err, setErr] = useState("");
  const nav = useNavigate();

  async function onSubmit(e) {
    e.preventDefault();
    setErr("");
    try {
      const res = await api.unlock(vault, master);
      try { localStorage.setItem("vaultName", res?.vault || ""); } catch {}
      nav("/vault");
    } catch (e) {
      setErr(e.message || "Failed to unlock");
    }
  }

  return (
    <Layout>
      <h1>Unlock</h1>
      {err && <div style={{ color: "crimson", marginBottom: 8 }}>{err}</div>}
      <form onSubmit={onSubmit} style={{ maxWidth: 420 }}>
        <label>Vault Path</label><br/>
        <input value={vault} onChange={(e)=>setVault(e.target.value)} />
        <div style={{ height: 12 }} />
        <label>Master Password</label><br/>
        <input type="password" value={master} onChange={(e)=>setMaster(e.target.value)} />
        <div style={{ height: 12 }} />
        <button type="submit">Unlock</button>
      </form>
    </Layout>
  );
}
