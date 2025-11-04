// frontend/src/pages/Vault.jsx
import { useEffect, useMemo, useState } from "react";
import Layout from "../components/Layout.jsx";
import ItemRow from "../components/ItemRow.jsx";
import { listItems } from "../lib/api";

export default function Vault() {
  const [items, setItems] = useState([]);
  const [typeFilter, setTypeFilter] = useState("login");
  const [query, setQuery] = useState("");
  const [err, setErr] = useState("");
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      setLoading(true);
      setErr("");
      try {
        const params = { type: "login" };
        const { items: data } = await listItems(params);
        if (!cancelled) {
          setItems(Array.isArray(data) ? data : []);
        }
      } catch (e) {
        if (!cancelled) {
          setErr(e?.message || "Failed to load items");
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [typeFilter]);

  const filtered = useMemo(() => {
    const needle = query.trim().toLowerCase();
    if (!needle) return items;
    return items.filter((item) => {
      const f = item?.fields || {};
      const haystack = `${item?.type || ""} ${f.site || f.title || f.name || ""} ${f.username || f.user || ""} ${
        f.notes || ""
      }`.toLowerCase();
      return haystack.includes(needle);
    });
  }, [items, query]);

  return (
    <Layout>
      <section className="glass-card section">
        <div className="section-header">
          <div>
            <h1 className="section-title">Your secure vault</h1>
            <p className="section-subtitle">
              {items.length
                ? `${items.length} login${items.length === 1 ? "" : "s"} tucked behind quantum-grade encryption`
                : "Start by adding your first login to your vault."}
            </p>
          </div>
          <div className="filter-bar">
            {/* Type filter removed (login-only) */}
            <input
              className="input"
              placeholder="Search everything…"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              aria-label="Search"
            />
          </div>
        </div>
        {err && (
          <div className="message message--error" role="alert">
            <span aria-hidden="true">⚠️</span>
            {err}
          </div>
        )}
      </section>

      {loading ? (
        <div className="glass-card">
          <div className="message message--info">
            <span aria-hidden="true">⏳</span>
            Unlocking your encrypted vault…
          </div>
        </div>
      ) : filtered.length === 0 ? (
        <div className="glass-card empty-state">
          <p>{items.length === 0 ? "Nothing here yet—why not add your first login?" : "No items match your search."}</p>
        </div>
      ) : (
        <div className="item-grid">
          {filtered.map((item) => (
            <ItemRow key={item.id} item={item} />
          ))}
        </div>
      )}
    </Layout>
  );
}
