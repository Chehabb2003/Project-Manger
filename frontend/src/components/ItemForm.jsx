import { useEffect, useState } from "react";

/**
 * Reusable form for creating/updating an item.
 * Props:
 *  - initial: { type, fields: { site, username, password, notes } }
 *  - onSubmit: (payload) => Promise|void
 *  - onCancel: () => void
 *  - submitLabel: string ("Create" | "Save")
 */
export default function ItemForm({
  initial = { type: "login", fields: { site: "", username: "", password: "", notes: "" } },
  onSubmit,
  onCancel,
  submitLabel = "Save",
}) {
  const [type, setType] = useState(initial.type || "login");
  const [fields, setFields] = useState({
    site: "",
    username: "",
    password: "",
    notes: "",
    ...(initial.fields || {}),
  });
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    setType(initial.type || "login");
    setFields({
      site: "",
      username: "",
      password: "",
      notes: "",
      ...(initial.fields || {}),
    });
  }, [initial]);

  function change(k, v) {
    setFields((s) => ({ ...s, [k]: v }));
  }

  async function handleSubmit(e) {
    e.preventDefault();
    setErr("");
    setBusy(true);
    try {
      await onSubmit?.({ type, fields });
    } catch (e2) {
      setErr(e2?.message || "Failed to save");
    } finally {
      setBusy(false);
    }
  }

  return (
    <form onSubmit={handleSubmit} style={{ display: "grid", gap: 12 }}>
      <label>
        <div>Type</div>
        <select value={type} onChange={(e) => setType(e.target.value)}>
          <option value="login">Login</option>
          <option value="note">Secure Note</option>
          <option value="card">Card</option>
        </select>
      </label>

      <label>
        <div>Site / Title</div>
        <input value={fields.site} onChange={(e) => change("site", e.target.value)} placeholder="example.com" />
      </label>

      {type === "login" && (
        <>
          <label>
            <div>Username</div>
            <input value={fields.username} onChange={(e) => change("username", e.target.value)} />
          </label>
          <label>
            <div>Password</div>
            <input type="password" value={fields.password} onChange={(e) => change("password", e.target.value)} />
          </label>
        </>
      )}

      <label>
        <div>Notes</div>
        <textarea rows={4} value={fields.notes} onChange={(e) => change("notes", e.target.value)} />
      </label>

      {err && <div style={{ color: "crimson" }}>{err}</div>}

      <div style={{ display: "flex", gap: 8 }}>
        <button type="submit" disabled={busy}>{busy ? "Saving…" : submitLabel}</button>
        {onCancel && <button type="button" onClick={onCancel}>Cancel</button>}
      </div>
    </form>
  );
}
