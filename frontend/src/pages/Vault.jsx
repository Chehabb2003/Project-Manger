// frontend/src/pages/Vault.jsx
import { useEffect, useState } from "react";
import Layout from "../components/Layout.jsx";
import ItemRow from "../components/ItemRow.jsx";
import { listItems } from "../lib/api";

export default function Vault() {
  const [items, setItems] = useState([]);
  const [typeFilter, setTypeFilter] = useState(""); // "", "login", "note", "card"
  const [q, setQ] = useState("");
  const [err, setErr] = useState("");

  useEffect(() => {
    let cancel = false;
    (async () => {
      setErr("");
      try {
        const params = {};
        if (typeFilter) params.type = typeFilter;
        const { items } = await listItems(params);
        if (!cancel) setItems(items);
      } catch (e) {
        if (!cancel) setErr(e.message || "Failed to load");
      }
    })();
    return () => { cancel = true; };
  }, [typeFilter]);

  const filtered = items.filter((it) => {
    if (!q.trim()) return true;
    const f = it.fields || {};
    const hay = `${f.site || f.title || f.name || ""} ${f.username || f.user || ""} ${it.type || ""}`.toLowerCase();
    return hay.includes(q.toLowerCase());
  });

  return (
    <Layout>
      <div style={{ display: "flex", gap: 12, alignItems: "center" }}>
        <select value={typeFilter} onChange={(e)=>setTypeFilter(e.target.value)}>
          <option value="">All</option>
          <option value="login">Login</option>
          <option value="note">Secure Note</option>
          <option value="card">Card</option>
        </select>
        <input
          placeholder="Search..."
          value={q}
          onChange={(e)=>setQ(e.target.value)}
          style={{ flex: 1 }}
        />
      </div>

      {err && <div style={{ color: "crimson", marginTop: 8 }}>{err}</div>}

      <h1>Vault</h1>

      <table style={{ width: "100%" }}>
        <thead>
          <tr>
            <th style={{ textAlign: "left" }}>Site/Name</th>
            <th style={{ textAlign: "left" }}>Username</th>
            <th style={{ textAlign: "left" }}>Type</th>
            <th />
          </tr>
        </thead>
        <tbody>
          {filtered.map((it) => <ItemRow key={it.id} item={it} />)}
        </tbody>
      </table>
    </Layout>
  );
}
