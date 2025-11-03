// frontend/src/pages/ViewItem.jsx
import { useEffect, useState } from "react";
import { useParams, Link, useNavigate } from "react-router-dom";
import { getItem, deleteItem } from "../lib/api";
import Layout from "../components/Layout.jsx";

function maskNumber(n) {
  const d = (n || "").replace(/\D/g, "");
  if (!d) return "—";
  const l4 = d.slice(-4);
  return `•••• •••• •••• ${l4}`;
}

export default function ViewItem() {
  const { id } = useParams();
  const nav = useNavigate();
  const [item, setItem] = useState(null);
  const [err, setErr] = useState("");

  useEffect(() => {
    (async () => {
      try {
        setErr("");
        const it = await getItem(id);
        setItem(it);
      } catch (e) {
        setErr(e.message || "Failed to load");
      }
    })();
  }, [id]);

  async function onDelete() {
    try {
      await deleteItem(id);
      nav("/vault");
    } catch (e) {
      setErr(e.message || "Failed to delete");
    }
  }

  const tRaw = (item?.type || "").toLowerCase();
  const isCard = tRaw === "card";
  const isNote = tRaw === "note" || tRaw === "secure note" || tRaw === "secure-note";
  const f = item?.fields || {};

  const iconBtn = {
    border: "1px solid #444",
    background: "transparent",
    color: "inherit",
    padding: "4px 10px",
    borderRadius: 8,
    cursor: "pointer",
  };

  return (
    <Layout>
      {err && <div style={{ color: "#f55" }}>{err}</div>}
      {!item ? (
        <div>Loading…</div>
      ) : (
        <div style={{ maxWidth: 640 }}>
          <div style={{ display: "flex", alignItems: "center", gap: 10, marginBottom: 12 }}>
            <button aria-label="Back" title="Back" style={iconBtn} onClick={() => nav(-1)}>
              ←
            </button>
            <h1 style={{ margin: 0 }}>{isCard ? "Card" : isNote ? "Secure Note" : "Item"}</h1>
          </div>

          {isCard ? (
            <table>
              <tbody>
                <tr><td>Cardholder</td><td>{f.cardholder || "—"}</td></tr>
                <tr><td>Number</td><td>{maskNumber(f.number)}</td></tr>
                <tr><td>Expiry</td><td>{(f.exp_month || "—")}/{(f.exp_year || "—")}</td></tr>
                <tr><td>CVV</td><td>{"•••"}</td></tr>
                <tr><td>Network</td><td>{f.network || "—"}</td></tr>
                <tr><td>Notes</td><td>{f.notes || "—"}</td></tr>
              </tbody>
            </table>
          ) : isNote ? (
            <table>
              <tbody>
                <tr><td>Title</td><td>{f.site || f.title || f.name || "—"}</td></tr>
                <tr><td>Notes</td><td>{f.notes || "—"}</td></tr>
              </tbody>
            </table>
          ) : (
            <table>
              <tbody>
                <tr><td>Site / Title</td><td>{f.site || f.title || f.name || "—"}</td></tr>
                <tr><td>Username</td><td>{f.username || f.user || "—"}</td></tr>
                <tr><td>Password</td><td>{"••••••••"}</td></tr>
                <tr><td>Notes</td><td>{f.notes || "—"}</td></tr>
              </tbody>
            </table>
          )}

          <div style={{ marginTop: 12 }}>
            <Link to={`/items/${id}/edit`}><button>Edit</button></Link>{" "}
            <button onClick={onDelete}>Delete</button>
          </div>
        </div>
      )}
    </Layout>
  );
}
