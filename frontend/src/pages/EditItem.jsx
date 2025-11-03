// frontend/src/pages/EditItem.jsx
import { useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import Layout from "../components/Layout.jsx";
import ItemForm from "../components/ItemForm.jsx";
import { getItem, updateItem, deleteItem } from "../lib/api";

export default function EditItem() {
  const { id } = useParams();
  const navigate = useNavigate();

  const [initial, setInitial] = useState({ type: "login", fields: {} });
  const [loading, setLoading] = useState(true);
  const [err, setErr] = useState("");
  const [saved, setSaved] = useState(false);

  useEffect(() => {
    let cancel = false;
    (async () => {
      setErr("");
      setLoading(true);
      try {
        const it = await getItem(id);
        if (!cancel) {
          setInitial({
            type: (it?.type || "login").toLowerCase(),
            fields: it?.fields || {},
          });
        }
      } catch (e) {
        if (!cancel) setErr(e?.message || "Failed to load item");
      } finally {
        if (!cancel) setLoading(false);
      }
    })();
    return () => { cancel = true; };
  }, [id]);

  async function handleSubmit(payload) {
    await updateItem(id, payload);
    setSaved(true);
    setTimeout(() => setSaved(false), 1500); // stay on page
  }

  async function handleDelete() {
    if (!window.confirm("Delete this item?")) return;
    await deleteItem(id);
    navigate("/vault");
  }

  function handleCancel() {
    navigate(-1);
  }

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
      <div style={{ display: "flex", alignItems: "center", gap: 10, marginBottom: 12 }}>
        <button aria-label="Back" title="Back" style={iconBtn} onClick={() => navigate(-1)}>
          ←
        </button>
        <h1 style={{ margin: 0 }}>Edit Item</h1>
        {saved && <span style={{ marginLeft: 12, color: "#79d279", fontWeight: 600 }}>Saved ✓</span>}
      </div>

      {err && <div style={{ color: "crimson", marginBottom: 8 }}>{err}</div>}
      {loading ? (
        <div>Loading…</div>
      ) : (
        <ItemForm
          initial={initial}
          onSubmit={handleSubmit}
          onCancel={handleCancel}
          onDelete={handleDelete}
          submitLabel="Save"
        />
      )}
    </Layout>
  );
}
