import { useMemo, useState } from "react";
import { useSearchParams, Link } from "react-router-dom";
import { resetPassword } from "../lib/api.js";

function checkPasswordPolicy(pw) {
  const issues = [];
  if (pw.length < 12) issues.push("at least 12 characters");
  if (!/[A-Z]/.test(pw)) issues.push("an uppercase letter");
  if (!/[a-z]/.test(pw)) issues.push("a lowercase letter");
  if (!/[0-9]/.test(pw)) issues.push("a digit");
  if (!/[^A-Za-z0-9]/.test(pw)) issues.push("a symbol");
  if (/\s/.test(pw)) issues.push("no spaces");
  return issues;
}

export default function ResetPassword() {
  const [params] = useSearchParams();
  const initialToken = useMemo(() => params.get("token") || "", [params]);

  const [token, setToken] = useState(initialToken);
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [pwErr, setPwErr] = useState("");
  const [err, setErr] = useState("");
  const [msg, setMsg] = useState("");
  const [busy, setBusy] = useState(false);

  async function onSubmit(e) {
    e.preventDefault();
    if (busy) return;
    setErr("");
    setMsg("");
    setPwErr("");

    const tk = token.trim();
    if (!tk) {
      setErr("Reset token required. Use the link from your email.");
      return;
    }
    if (password !== confirm) {
      setErr("Passwords do not match.");
      return;
    }
    const issues = checkPasswordPolicy(password);
    if (issues.length) {
      setPwErr("Password still needs " + issues.join(", "));
      return;
    }

    try {
      setBusy(true);
      const res = await resetPassword(tk, password);
      setMsg(res?.note || "Password updated. You can now sign in.");
      setPassword("");
      setConfirm("");
    } catch (e2) {
      setErr(`Failed: ${e2?.message || "unknown error"}`);
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="auth-page">
      <div className="glass-card" style={{ width: "min(520px, 100%)" }}>
        <span className="auth-badge">Password recovery</span>
        <h1 className="section-title">Reset your master password</h1>
        <p className="section-subtitle">
          Paste the reset token from your email and choose a new master password. Tokens expire after 15 minutes.
        </p>

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
          {/* <div className="form-field">
            <label className="input-label" htmlFor="reset-token">
              Reset token
            </label>
            <input
              id="reset-token"
              className="input"
              value={token}
              onChange={(e) => setToken(e.target.value)}
              placeholder="Paste your reset token"
            />
          </div> */}
          <div className="form-field">
            <label className="input-label" htmlFor="reset-password">
              New password
            </label>
            <input
              id="reset-password"
              className="input"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              autoComplete="new-password"
            />
          </div>
          <div className="form-field">
            <label className="input-label" htmlFor="reset-confirm">
              Confirm password
            </label>
            <input
              id="reset-confirm"
              className="input"
              type="password"
              value={confirm}
              onChange={(e) => setConfirm(e.target.value)}
              autoComplete="new-password"
            />
          </div>

          <button type="submit" className="btn btn-primary" disabled={busy}>
            {busy ? "Updating…" : "Set new password"}
          </button>
        </form>

        <p className="helper-text">
          All set? <Link to="/unlock">Return to the login portal</Link>.
        </p>
      </div>
    </div>
  );
}
