import { useNavigate } from "react-router-dom";
import Layout from "../components/Layout.jsx";
import ItemForm from "../components/ItemForm.jsx";
import { createItem } from "../lib/api";

export default function NewItem() {
  const navigate = useNavigate();

  async function handleSubmit(payload) {
    await createItem(payload);
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
            <h1 className="section-title">Add a new item</h1>
            <p className="section-subtitle">Store login credentials with zero-knowledge encryption.</p>
          </div>
          <button type="button" className="btn btn-ghost" onClick={() => navigate(-1)}>
            <span aria-hidden="true">←</span>
            Back
          </button>
        </div>

        <ItemForm
          initial={{ type: "login", fields: {} }}
          onSubmit={handleSubmit}
          onCancel={handleCancel}
          submitLabel="Create"
        />
      </section>
    </Layout>
  );
}
