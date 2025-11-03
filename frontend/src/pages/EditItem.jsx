import { useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import Layout from "../components/Layout.jsx";
import ItemForm from "../components/ItemForm.jsx";
import { createItem, getItem, updateItem } from "../lib/api";

export default function EditItem() {
  const { id } = useParams();              // undefined for /items/new
  const isEdit = !!id;                     // edit when /items/:id/edit
  const navigate = useNavigate();

  const [initial, setInitial] = useState({ type: "login", fields: {} });
  const [loading, setLoading] = useState(isEdit);
  const [err, setErr] = useState("");

  // Load existing item if editing
  useEffect(() => {
    if (!isEdit) return;
    let cancel = false;
    (async () => {
      setLoading(true);
      setErr("");
      try {
        const it = await getItem(id);
        if (!cancel) {
          setInitial({
            type: it?.type || "login",
            fields: {
              site: it?.fields?.site || it?.fields?.title || "",
              username: it?.fields?.username || "",
              password: it?.fields?.password || "",
              notes: it?.fields?.notes || "",
            },
          });
        }
      } catch (e) {
        if (!cancel) setErr(e?.message || "Failed to load item");
      } finally {
        if (!cancel) setLoading(false);
      }
    })();
    return () => { cancel = true; };
  }, [id, isEdit]);

  async function handleSubmit(payload) {
    if (isEdit) {
      await updateItem(id, payload);
      navigate(`/items/${id}`);
    } else {
      const created = await createItem(payload);
      const newId = created?.id ?? created; // support either {id} or raw id
      navigate(`/items/${newId}`);
    }
  }

  function handleCancel() {
    navigate(isEdit ? `/items/${id}` : "/vault");
  }

  return (
    <Layout>
      <h1 style={{ marginTop: 0 }}>{isEdit ? "Edit Item" : "New Item"}</h1>
      {err && <div style={{ color: "crimson", marginBottom: 8 }}>{err}</div>}
      {loading ? (
        <div>Loading…</div>
      ) : (
        <ItemForm
          initial={initial}
          onSubmit={handleSubmit}
          onCancel={handleCancel}
          submitLabel={isEdit ? "Save" : "Create"}
        />
      )}
    </Layout>
  );
}
