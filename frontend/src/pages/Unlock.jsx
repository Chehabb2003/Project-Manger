// src/pages/Unlock.jsx — Authentication landing
import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { login, signup, requestPasswordReset, verifyLogin } from "../lib/api";

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
  const [loginChallenge, setLoginChallenge] = useState(null);
  const [twoFACode, setTwoFACode] = useState("");
  const [totpSetup, setTotpSetup] = useState(null);
  const [copyNote, setCopyNote] = useState("");

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
    setLoginChallenge(null);
    setTwoFACode("");
    setTotpSetup(null);
    setCopyNote("");
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
        if (loginChallenge) {
          const pin = twoFACode.trim();
          if (!pin) {
            setErr("Enter the six-digit verification code.");
            return;
          }
          const res = await verifyLogin(loginChallenge.challenge_id, pin);
          if (res?.token) {
            setMsg("Success! Unlocking your vault…");
            setLoginChallenge(null);
            setTwoFACode("");
            setPassword("");
            setIdentifier("");
            setTimeout(() => navigate("/vault"), 300);
          } else {
            setErr("Unexpected server response.");
          }
          return;
        }

        const ident = identifier.trim();
        if (!ident || !password) {
          setErr("Please enter your username/email and password.");
          return;
        }
        const data = await login(ident, password);
        if (data?.token) {
          setMsg("Welcome back! Redirecting to your vault…");
          setTotpSetup(null);
          setTimeout(() => navigate("/vault"), 400);
        } else if (data?.challenge_id) {
          setLoginChallenge(data);
          setTwoFACode("");
          setMsg(data?.note || "Enter the code from your authenticator app.");
        } else {
          setErr("Unexpected server response.");
        }
        return;
      }

      if (mode === "signup") {
        if (loginChallenge) {
          const pin = twoFACode.trim();
          if (!pin) {
            setErr("Enter the six-digit verification code from your authenticator.");
            return;
          }
          const res = await verifyLogin(loginChallenge.challenge_id, pin);
          if (res?.token) {
            setMsg("Authenticator confirmed! Unlocking your new vault…");
            setLoginChallenge(null);
            setTwoFACode("");
            setPassword("");
            setConfirm("");
            setTotpSetup(null);
            setTimeout(() => navigate("/vault"), 350);
          } else {
            setErr("Unexpected server response.");
          }
          return;
        }

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
        const data = await signup(uname, mail, password);
        if (data?.totp_secret) {
          setTotpSetup({ secret: data.totp_secret, uri: data.totp_uri || "" });
          setCopyNote("");
          setLoginChallenge({
            challenge_id: data.challenge_id,
            expires_at: data.expires_at,
            fromSignup: true,
          });
          setTwoFACode("");
          setMsg("Add the secret to your authenticator, then enter the 6-digit code to finish signup.");
        } else {
          setMsg("Account created! Taking you to your new vault…");
          setTimeout(() => navigate("/vault"), 400);
        }
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
      if (mode === "login" && !loginChallenge) {
        setLoginChallenge(null);
        setTwoFACode("");
      }
    } finally {
      setBusy(false);
    }
  }

  async function copyTotpSecret() {
    if (!totpSetup?.secret || typeof navigator === "undefined" || !navigator.clipboard) {
      setCopyNote("Copy is not available in this browser.");
      return;
    }
    try {
      await navigator.clipboard.writeText(totpSetup.secret);
      setCopyNote("Secret copied to clipboard.");
    } catch (errCopy) {
      setCopyNote("Could not copy secret automatically.");
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
            VaultCraft keeps every credential and secret wrapped in resilient encryption.
            Breathe easy knowing your digital life lives behind a handcrafted vault.
          </p>
          <ul className="auth-features">
            <li className="auth-feature">
              <span>🛡</span>
              <div>Zero-knowledge encryption with automatic key rotation.</div>
            </li>
            <li className="auth-feature">
              <span>⚡️</span>
              <div>Instant unlocks, global search, and elegant login cards.</div>
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
                disabled={!!loginChallenge}
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
                disabled={!!loginChallenge}
              />
            </div>
          </>
        )}

        {loginChallenge && (mode === "login" || mode === "signup") && (
          <div className="form-field">
            <label className="input-label" htmlFor="twofa-code">
              Verification code
            </label>
            <input
              id="twofa-code"
              className="input"
              placeholder="Enter the 6-digit code"
              value={twoFACode}
              onChange={(e) => setTwoFACode(e.target.value)}
              inputMode="numeric"
            />
            {loginChallenge.expires_at && (
              <p className="helper-text">
                Code expires at {new Date(loginChallenge.expires_at).toLocaleTimeString()}.
              </p>
            )}
          </div>
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
                ? loginChallenge
                  ? "Verify & unlock"
                  : "Create your vault"
                : mode === "login"
                ? loginChallenge
                  ? "Verify code"
                  : "Unlock vault"
                : "Send reset email"}
            </button>
          </form>
          {totpSetup && (
            <div className="totp-setup-card" role="status">
              <h3>Finish setting up 2-step verification</h3>
              <p>
                Add this secret to your authenticator app (such as Google Authenticator, 1Password, or Authy).
                You&apos;ll need its rolling code every time you sign in.
              </p>
              <div className="totp-secret">
                <span className="secret-label">Secret</span>
                <code>{totpSetup.secret}</code>
                <button
                  type="button"
                  className="btn btn-tertiary"
                  onClick={copyTotpSecret}
                >
                  Copy secret
                </button>
              </div>
              {/* {totpSetup.uri && (
                <p className="helper-text">
                  Authenticator URI:&nbsp;
                  <a href={totpSetup.uri}>{totpSetup.uri}</a>
                </p>
              )}
              {copyNote && <p className="helper-text">{copyNote}</p>} */}
              <p className="helper-text">
                Enter a fresh code from your authenticator above, then click <strong>Verify &amp; unlock</strong> to continue.
              </p>
            </div>
          )}
        </section>
      </div>
    </div>
  );
}
