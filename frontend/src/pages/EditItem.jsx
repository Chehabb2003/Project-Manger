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
    let cancelled = false;
    (async () => {
      setErr("");
      setLoading(true);
      try {
        const it = await getItem(id);
        if (!cancelled) {
          setInitial({
            type: (it?.type || "login").toLowerCase(),
            fields: it?.fields || {},
          });
        }
      } catch (e) {
        if (!cancelled) {
          setErr(e?.message || "Failed to load item.");
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
  }, [id]);

  async function handleSubmit(payload) {
    await updateItem(id, payload);
    setSaved(true);
    setTimeout(() => setSaved(false), 1800);
  }

  async function handleDelete() {
    if (!window.confirm("Delete this item?")) return;
    await deleteItem(id);
    navigate("/vault");
  }

  function handleCancel() {
    navigate(-1);
  }

  return (
    <Layout>
      <section className="glass-card section" style={{ maxWidth: 720 }}>
        <div className="section-header">
          <div>
            <h1 className="section-title">Edit item</h1>
            <p className="section-subtitle">
              Update secrets, adjust metadata, or archive sensitive information securely.
            </p>
          </div>
          <button type="button" className="btn btn-ghost" onClick={() => navigate(-1)}>
            <span aria-hidden="true">←</span>
            Back
          </button>
        </div>

        {saved && (
          <div className="message message--success" role="status">
            <span aria-hidden="true">💾</span>
            Saved to vault.
          </div>
        )}
        {err && (
          <div className="message message--error" role="alert">
            <span aria-hidden="true">⚠️</span>
            {err}
          </div>
        )}

        {loading ? (
          <div className="message message--info">
            <span aria-hidden="true">⏳</span>
            Retrieving encrypted item…
          </div>
        ) : (
          <ItemForm
            initial={initial}
            onSubmit={handleSubmit}
            onCancel={handleCancel}
            onDelete={handleDelete}
            submitLabel="Save changes"
          />
        )}
      </section>
    </Layout>
  );
}
