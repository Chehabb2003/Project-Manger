// src/pages/Unlock.jsx — Authentication landing
import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { login, signup, requestPasswordReset } from "../lib/api";

const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

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

const MODES = ["login", "signup", "forgot"];

export default function Unlock() {
  const [mode, setMode] = useState("login");
  const [identifier, setIdentifier] = useState("");
  const [username, setUsername] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [pwErr, setPwErr] = useState("");
  const [msg, setMsg] = useState("");
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);

  const navigate = useNavigate();

  function selectMode(next) {
    if (next === mode) return;
    setMode(next);
    setIdentifier("");
    setUsername("");
    setEmail("");
    setPassword("");
    setConfirm("");
    setPwErr("");
    setMsg("");
    setErr("");
  }

  async function onSubmit(e) {
    e.preventDefault();
    if (busy) return;
    setErr("");
    setMsg("");
    setPwErr("");

    try {
      setBusy(true);
      if (mode === "login") {
        const ident = identifier.trim();
        if (!ident || !password) {
          setErr("Please enter your username/email and password.");
          return;
        }
        await login(ident, password);
        setMsg("Welcome back! Redirecting to your vault…");
        setTimeout(() => navigate("/vault"), 400);
        return;
      }

      if (mode === "signup") {
        const uname = username.trim();
        const mail = email.trim().toLowerCase();
        if (!uname) {
          setErr("Choose a username to continue.");
          return;
        }
        if (!mail || !emailRegex.test(mail)) {
          setErr("Please provide a valid email address.");
          return;
        }
        if (password !== confirm) {
          setErr("Passwords do not match.");
          return;
        }
        const issues = checkPasswordPolicy(password);
        if (issues.length) {
          setPwErr("Your password still needs " + issues.join(", "));
          return;
        }
        await signup(uname, mail, password);
        setMsg("Account created! Taking you to your new vault…");
        setTimeout(() => navigate("/vault"), 400);
        return;
      }

      // Forgot password
      const mail = email.trim().toLowerCase();
      if (!mail || !emailRegex.test(mail)) {
        setErr("Enter the email address associated with your vault.");
        return;
      }
      const res = await requestPasswordReset(mail);
      setMsg(res?.note || "If the account exists, we just sent a reset link.");
    } catch (e2) {
      setErr(e2?.message ? `Oops — ${e2.message}` : "Something went wrong.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="auth-page">
      <div className="auth-wrapper">
        <aside className="auth-side">
          <span className="auth-badge">VaultCraft</span>
          <h1 className="auth-heading">
            Effortless, beautiful password security for the modern web.
          </h1>
          <p className="auth-copy">
            VaultCraft keeps every credential, note, and secret wrapped in resilient encryption.
            Breathe easy knowing your digital life lives behind a handcrafted vault.
          </p>
          <ul className="auth-features">
            <li className="auth-feature">
              <span>🛡</span>
              <div>Zero-knowledge encryption with automatic key rotation.</div>
            </li>
            <li className="auth-feature">
              <span>⚡️</span>
              <div>Instant unlocks, global search, and elegant item cards.</div>
            </li>
            <li className="auth-feature">
              <span>🌙</span>
              <div>Immersive glassmorphism interface built for focus and delight.</div>
            </li>
          </ul>
        </aside>

        <section className="auth-side">
          <div className="mode-switch" role="tablist" aria-label="Authentication modes">
            {MODES.map((m) => (
              <button
                key={m}
                type="button"
                className={`mode-btn ${mode === m ? "active" : ""}`}
                onClick={() => selectMode(m)}
                role="tab"
                aria-selected={mode === m}
              >
                {m === "login" && "Sign in"}
                {m === "signup" && "Create account"}
                {m === "forgot" && "Forgot password"}
              </button>
            ))}
          </div>

          {msg && (
            <div className="message message--success" role="status">
              <span aria-hidden="true">✨</span>
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

          <form className="auth-form" onSubmit={onSubmit}>
            {mode === "login" && (
              <>
                <div className="form-field">
                  <label className="input-label" htmlFor="auth-identifier">
                    Username or Email
                  </label>
                  <input
                    id="auth-identifier"
                    className="input"
                    placeholder="you@example.com"
                    value={identifier}
                    onChange={(e) => setIdentifier(e.target.value)}
                    autoComplete="username"
                  />
                </div>
                <div className="form-field">
                  <label className="input-label" htmlFor="auth-password">
                    Password
                  </label>
                  <input
                    id="auth-password"
                    className="input"
                    type="password"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    autoComplete="current-password"
                  />
                </div>
              </>
            )}

            {mode === "signup" && (
              <>
                <div className="form-field">
                  <label className="input-label" htmlFor="signup-username">
                    Username
                  </label>
                  <input
                    id="signup-username"
                    className="input"
                    placeholder="your-handle"
                    value={username}
                    onChange={(e) => setUsername(e.target.value)}
                    autoComplete="username"
                  />
                </div>
                <div className="form-field">
                  <label className="input-label" htmlFor="signup-email">
                    Email
                  </label>
                  <input
                    id="signup-email"
                    className="input"
                    placeholder="you@example.com"
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                    autoComplete="email"
                  />
                </div>
                <div className="form-field">
                  <label className="input-label" htmlFor="signup-password">
                    Master password
                  </label>
                  <input
                    id="signup-password"
                    className="input"
                    type="password"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    autoComplete="new-password"
                  />
                </div>
                <div className="form-field">
                  <label className="input-label" htmlFor="signup-confirm">
                    Confirm password
                  </label>
                  <input
                    id="signup-confirm"
                    className="input"
                    type="password"
                    value={confirm}
                    onChange={(e) => setConfirm(e.target.value)}
                    autoComplete="new-password"
                  />
                </div>
                <p className="helper-text">
                  Tip: use a strong passphrase you&apos;ll remember — VaultCraft takes care of the rest.
                </p>
              </>
            )}

            {mode === "forgot" && (
              <>
                <div className="form-field">
                  <label className="input-label" htmlFor="forgot-email">
                    Account email
                  </label>
                  <input
                    id="forgot-email"
                    className="input"
                    placeholder="you@example.com"
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                    autoComplete="email"
                  />
                </div>
                <p className="helper-text">
                  We&apos;ll email you a secure link to reset your master password. The link expires after 15 minutes.
                </p>
              </>
            )}

            <button type="submit" className="btn btn-primary" disabled={busy}>
              {busy
                ? "Working…"
                : mode === "signup"
                ? "Create your vault"
                : mode === "login"
                ? "Unlock vault"
                : "Send reset email"}
            </button>
          </form>
        </section>
      </div>
    </div>
  );
}
