// frontend/src/pages/ViewItem.jsx
import { useEffect, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import Layout from "../components/Layout.jsx";
import { deleteItem, getItem } from "../lib/api";

function maskNumber(n) {
  const d = (n || "").replace(/\D/g, "");
  if (!d) return "—";
  const l4 = d.slice(-4);
  return `•••• •••• •••• ${l4}`;
}

export default function ViewItem() {
  const { id } = useParams();
  const navigate = useNavigate();
  const [item, setItem] = useState(null);
  const [err, setErr] = useState("");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      setLoading(true);
      setErr("");
      try {
        const it = await getItem(id);
        if (!cancelled) setItem(it);
      } catch (e) {
        if (!cancelled) setErr(e?.message || "Failed to load item.");
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [id]);

  async function onDelete() {
    if (!window.confirm("Delete this item?")) return;
    await deleteItem(id);
    navigate("/vault");
  }

  const tRaw = (item?.type || "").toLowerCase();
  const isCard = tRaw === "card";
  const isNote = tRaw === "note" || tRaw === "secure note" || tRaw === "secure-note";
  const f = item?.fields || {};

  const title =
    (isCard && "Card") || (isNote && "Secure note") || "Login item";

  return (
    <Layout>
      <section className="glass-card section" style={{ maxWidth: 720 }}>
        <div className="section-header">
          <div>
            <h1 className="section-title">{title}</h1>
            <p className="section-subtitle">
              {item?.fields?.site || item?.fields?.title || item?.fields?.name || "Protected entry"}
            </p>
          </div>
          <button type="button" className="btn btn-ghost" onClick={() => navigate(-1)}>
            <span aria-hidden="true">←</span>
            Back
          </button>
        </div>

        {err && (
          <div className="message message--error" role="alert">
            <span aria-hidden="true">⚠️</span>
            {err}
          </div>
        )}

        {loading ? (
          <div className="message message--info">
            <span aria-hidden="true">⏳</span>
            Unlocking item…
          </div>
        ) : (
          <>
            <table className="table-like">
              <tbody>
                {isCard ? (
                  <>
                    <tr>
                      <td>Cardholder</td>
                      <td>{f.cardholder || "—"}</td>
                    </tr>
                    <tr>
                      <td>Number</td>
                      <td>{maskNumber(f.number)}</td>
                    </tr>
                    <tr>
                      <td>Expiry</td>
                      <td>
                        {(f.exp_month || "—")}/{f.exp_year || "—"}
                      </td>
                    </tr>
                    <tr>
                      <td>Network</td>
                      <td>{f.network || "—"}</td>
                    </tr>
                    <tr>
                      <td>Notes</td>
                      <td>{f.notes || "—"}</td>
                    </tr>
                  </>
                ) : isNote ? (
                  <>
                    <tr>
                      <td>Title</td>
                      <td>{f.site || f.title || f.name || "—"}</td>
                    </tr>
                    <tr>
                      <td>Notes</td>
                      <td>{f.notes || "—"}</td>
                    </tr>
                  </>
                ) : (
                  <>
                    <tr>
                      <td>Site / Title</td>
                      <td>{f.site || f.title || f.name || "—"}</td>
                    </tr>
                    <tr>
                      <td>Username</td>
                      <td>{f.username || f.user || "—"}</td>
                    </tr>
                    <tr>
                      <td>Password</td>
                      <td>••••••••</td>
                    </tr>
                    <tr>
                      <td>Notes</td>
                      <td>{f.notes || "—"}</td>
                    </tr>
                  </>
                )}
              </tbody>
            </table>

            <div className="split-actions">
              <Link to={`/items/${id}/edit`} className="btn btn-primary">
                <span aria-hidden="true">✏️</span>
                Edit
              </Link>
              <button type="button" className="btn btn-danger" onClick={onDelete}>
                <span aria-hidden="true">🗑</span>
                Delete
              </button>
            </div>
          </>
        )}
      </section>
    </Layout>
  );
}
