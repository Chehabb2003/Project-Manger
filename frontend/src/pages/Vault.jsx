import { useEffect, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import Layout from "../components/Layout.jsx";
import SearchBox from "../components/SearchBox.jsx";
import ItemRow from "../components/ItemRow.jsx";

import { listItems } from "../lib/api";
import { useAuth } from "../context/AuthContext";

export default function Vault() {
  const { isUnlocked } = useAuth();
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const [items, setItems] = useState([]);
  const [loading, setLoading] = useState(true);
  const [err, setErr] = useState("");

  const q = searchParams.get("q") || "";
  const type = searchParams.get("type") || "login";

  // Guard: redirect if locked
  useEffect(() => {
    if (!isUnlocked?.ok) navigate("/unlock");
  }, [isUnlocked, navigate]);

  // Fetch items whenever filters change
  useEffect(() => {
    let cancel = false;
    (async () => {
      setLoading(true);
      setErr("");
      try {
        const res = await listItems({ type, q });
        if (!cancel) setItems(res?.items || []);
      } catch (e) {
        if (!cancel) setErr(e?.message || "Failed to load items");
      } finally {
        if (!cancel) setLoading(false);
      }
    })();
    return () => { cancel = true; };
  }, [type, q]);

  function onSearch(next) {
    setSearchParams(prev => {
      const sp = new URLSearchParams(prev);
      if (next) sp.set("q", next); else sp.delete("q");
      sp.set("type", type);
      return sp;
    });
  }

  function onTypeChange(e) {
    const nextType = e.target.value;
    setSearchParams(prev => {
      const sp = new URLSearchParams(prev);
      if (q) sp.set("q", q); else sp.delete("q");
      sp.set("type", nextType);
      return sp;
    });
  }

  return (
    <Layout>
      <h1 style={{ marginTop: 0 }}>Vault</h1>

      <div style={{ display: "flex", gap: 8, margin: "12px 0 16px" }}>
        <select value={type} onChange={onTypeChange}>
          <option value="login">Login</option>
          <option value="note">Secure Note</option>
          <option value="card">Card</option>
        </select>

        {/* key={q} forces the SearchBox to re-initialize if URL changes */}
        <SearchBox key={q} defaultValue={q} onChange={onSearch} placeholder="Search…" />
      </div>

      {err && <div style={{ color: "crimson", marginBottom: 8 }}>{err}</div>}
      {loading ? (
        <div>Loading…</div>
      ) : items.length === 0 ? (
        <div>No items.</div>
      ) : (
        <table style={{ width: "100%", borderCollapse: "collapse" }}>
          <thead>
            <tr>
              <th align="left" style={{ borderBottom: "1px solid #eee", padding: 8 }}>Site/Name</th>
              <th align="left" style={{ borderBottom: "1px solid #eee", padding: 8 }}>Username</th>
              <th align="left" style={{ borderBottom: "1px solid #eee", padding: 8 }}>Type</th>
              <th style={{ borderBottom: "1px solid #eee", padding: 8 }} />
            </tr>
          </thead>
          <tbody>
            {items.map(item => <ItemRow key={item.id} item={item} />)}
          </tbody>
        </table>
      )}
    </Layout>
  );
}
