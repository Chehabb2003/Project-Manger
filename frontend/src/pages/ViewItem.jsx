// src/pages/ViewItem.jsx
import { useEffect, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { deleteItem, getItem } from "../lib/api";

function ViewItem() {
  const { id } = useParams();
  const navigate = useNavigate();
  const [item, setItem] = useState(null);
  const [loading, setLoading] = useState(true);
  const [err, setErr] = useState("");

  useEffect(() => {
    let cancel = false;
    (async () => {
      setErr("");
      setLoading(true);
      try {
        const res = await getItem(id);
        if (!cancel) setItem(res);
      } catch (e) {
        if (!cancel) setErr(e.message || "Failed to load item");
      } finally {
        if (!cancel) setLoading(false);
      }
    })();
    return () => { cancel = true; };
  }, [id]);

  async function onDelete() {
    if (!confirm("Delete this item?")) return;
    try {
      await deleteItem(id);
      navigate("/vault");
    } catch (e) {
      alert(e.message || "Failed to delete");
    }
  }

  function copy(text) {
    navigator.clipboard.writeText(text || "");
  }

  if (loading) return <div className="container" style={{ maxWidth: 720, margin: "24px auto" }}>Loading…</div>;
  if (err) return <div className="container" style={{ maxWidth: 720, margin: "24px auto", color: "crimson" }}>{err}</div>;
  if (!item) return null;

  const f = item.fields || {};

  return (
    <div className="container" style={{ maxWidth: 720, margin: "24px auto" }}>
      <div style={{ display: "flex", alignItems: "center", gap: 12 }}>
        <h1 style={{ flex: 1, margin: 0 }}>{f.site || f.title || "(untitled)"}</h1>
        <Link to={`/items/${id}/edit`}><button>Edit</button></Link>
        <button onClick={onDelete} style={{ background: "#f33", color: "white" }}>Delete</button>
      </div>

      <div style={{ marginTop: 16, display: "grid", gap: 8 }}>
        {f.username && (
          <Field label="Username" value={f.username} onCopy={() => copy(f.username)} />
        )}
        {f.password && (
          <Field label="Password" masked value={f.password} onCopy={() => copy(f.password)} />
        )}
        {f.notes && (
          <div>
            <div style={{ fontWeight: 600, marginBottom: 4 }}>Notes</div>
            <pre style={{ whiteSpace: "pre-wrap", background: "#fafafa", padding: 12, border: "1px solid #eee" }}>
              {f.notes}
            </pre>
          </div>
        )}
      </div>
    </div>
  );
}

function Field({ label, value, masked, onCopy }) {
  const [show, setShow] = useState(false);
  return (
    <div style={{ display: "grid", gap: 4 }}>
      <div style={{ fontWeight: 600 }}>{label}</div>
      <div style={{ display: "flex", gap: 8, alignItems: "center" }}>
        <input
          readOnly
          type={masked && !show ? "password" : "text"}
          value={value}
          style={{ flex: 1 }}
        />
        {masked && <button onClick={() => setShow(s => !s)}>{show ? "Hide" : "Show"}</button>}
        <button onClick={onCopy}>Copy</button>
      </div>
    </div>
  );
}

export default ViewItem;
