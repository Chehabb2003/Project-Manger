import { useState } from "react";
import { useNavigate } from "react-router-dom";
import Layout from "../components/Layout.jsx";
import { changePassword } from "../lib/api.js";

function describePolicyIssues(pw) {
  const issues = [];
  if (pw.length < 12) issues.push("at least 12 characters");
  if (!/[A-Z]/.test(pw)) issues.push("an uppercase letter");
  if (!/[a-z]/.test(pw)) issues.push("a lowercase letter");
  if (!/[0-9]/.test(pw)) issues.push("a digit");
  if (!/[^A-Za-z0-9]/.test(pw)) issues.push("a symbol");
  if (/\s/.test(pw)) issues.push("no spaces");
  return issues;
}

export default function ChangePassword() {
  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const [confirm, setConfirm] = useState("");
  const [pwErr, setPwErr] = useState("");
  const [err, setErr] = useState("");
  const [msg, setMsg] = useState("");
  const [busy, setBusy] = useState(false);
  const navigate = useNavigate();

  async function onSubmit(e) {
    e.preventDefault();
    setErr("");
    setMsg("");
    setPwErr("");

    if (!current.trim()) {
      setErr("Enter your current password to continue.");
      return;
    }
    if (next !== confirm) {
      setErr("The new passwords do not match.");
      return;
    }

    const issues = describePolicyIssues(next);
    if (issues.length) {
      setPwErr("Password still needs " + issues.join(", "));
      return;
    }

    setBusy(true);
    try {
      const res = await changePassword(current, next);
      setMsg(res?.note || "Password updated successfully.");
      setCurrent("");
      setNext("");
      setConfirm("");
    } catch (e2) {
      setErr(e2?.message ? `Failed: ${e2.message}` : "Failed to change password.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <Layout>
      <section className="glass-card section" style={{ maxWidth: 520 }}>
        <div>
          <h1 className="section-title">Strengthen your master key</h1>
          <p className="section-subtitle">
            Changing your password rotates the encryption keys used across your vault.
          </p>
        </div>

        {msg && (
          <div className="message message--success" role="status">
            <span aria-hidden="true">✅</span>
            {msg}
          </div>
        )}
        {err && (
          <div className="message message--error" role="alert">
            <span aria-hidden="true">⚠️</span>
            {err}
          </div>
        )}
        {pwErr && (
          <div className="message message--info" role="alert">
            <span aria-hidden="true">🔐</span>
            {pwErr}
          </div>
        )}

        <form className="form-card" onSubmit={onSubmit}>
          <div className="form-field">
            <label className="input-label" htmlFor="current-password">
              Current password
            </label>
            <input
              id="current-password"
              className="input"
              type="password"
              value={current}
              onChange={(e) => setCurrent(e.target.value)}
              autoComplete="current-password"
            />
          </div>

          <div className="form-row">
            <div className="form-field">
              <label className="input-label" htmlFor="new-password">
                New password
              </label>
              <input
                id="new-password"
                className="input"
                type="password"
                value={next}
                onChange={(e) => setNext(e.target.value)}
                autoComplete="new-password"
              />
            </div>
            <div className="form-field">
              <label className="input-label" htmlFor="confirm-password">
                Confirm
              </label>
              <input
                id="confirm-password"
                className="input"
                type="password"
                value={confirm}
                onChange={(e) => setConfirm(e.target.value)}
                autoComplete="new-password"
              />
            </div>
          </div>

          <p className="helper-text">
            Use a unique passphrase you&apos;ll remember. VaultCraft can manage the rest of your secrets.
          </p>

          <div className="split-actions">
            <button type="submit" className="btn btn-primary" disabled={busy}>
              {busy ? "Updating…" : "Update password"}
            </button>
            <button type="button" className="btn btn-ghost" onClick={() => navigate(-1)}>
              Cancel
            </button>
          </div>
        </form>
      </section>
    </Layout>
  );
}
