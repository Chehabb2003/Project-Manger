// frontend/src/pages/NewItem.jsx
import Layout from "../components/Layout.jsx";
import ItemForm from "../components/ItemForm.jsx";
import { createItem } from "../lib/api";
import { useNavigate } from "react-router-dom";

export default function NewItem() {
  const navigate = useNavigate();

  const iconBtn = {
    border: "1px solid #444",
    background: "transparent",
    color: "inherit",
    padding: "4px 10px",
    borderRadius: 8,
    cursor: "pointer",
  };

  async function handleSubmit(payload) {
    await createItem(payload);
    navigate("/vault"); // go back to list after create
  }

  function handleCancel() {
    navigate(-1);
  }

  return (
    <Layout>
      <div style={{ display: "flex", alignItems: "center", gap: 10, marginBottom: 12 }}>
        <button aria-label="Back" title="Back" style={iconBtn} onClick={() => navigate(-1)}>
          ←
        </button>
        <h1 style={{ margin: 0 }}>New Item</h1>
      </div>

      <ItemForm
        initial={{ type: "login", fields: {} }}
        onSubmit={handleSubmit}
        onCancel={handleCancel}
        submitLabel="Create"
      />
    </Layout>
  );
}
