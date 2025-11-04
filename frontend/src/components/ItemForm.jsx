import { useMemo, useState } from "react";

function digitsOnly(s) {
  return (s || "").replace(/\D/g, "");
}

function luhnCheck(num) {
  const s = digitsOnly(num);
  if (!s) return false;
  let sum = 0;
  let dbl = false;
  for (let i = s.length - 1; i >= 0; i -= 1) {
    let n = s.charCodeAt(i) - 48;
    if (dbl) {
      n *= 2;
      if (n > 9) n -= 9;
    }
    sum += n;
    dbl = !dbl;
  }
  return sum % 10 === 0;
}

export default function ItemForm({
  initial = { type: "login", fields: {} },
  onSubmit,
  onCancel,
  submitLabel = "Save",
  onDelete,
}) {
  const kind = "login";

  const [site, setSite] = useState(initial.fields?.site || initial.fields?.title || "");
  const [notes, setNotes] = useState(initial.fields?.notes || "");
  const [username, setUsername] = useState(initial.fields?.username || initial.fields?.user || "");
  const [password, setPassword] = useState(initial.fields?.password || "");
  const [showPassword, setShowPassword] = useState(false);
  const [cardholder, setCardholder] = useState(initial.fields?.cardholder || "");
  const [cardNumber, setCardNumber] = useState(initial.fields?.number || "");
  const [expMonth, setExpMonth] = useState(initial.fields?.exp_month || "");
  const [expYear, setExpYear] = useState(initial.fields?.exp_year || "");
  const [cvv, setCvv] = useState(initial.fields?.cvv || "");
  const [network, setNetwork] = useState(initial.fields?.network || "");
  const [err, setErr] = useState("");

  const placeholderForSite = useMemo(() => "example.com", []);

  async function submit(e) {
    e.preventDefault();
    setErr("");

    const payload = { type: "login", fields: {} };

    if (!site) return setErr("Please add a website or title for this login.");
    payload.fields = { site, username, password, notes };

    await onSubmit(payload);
  }

  return (
    <form className="form-card" onSubmit={submit}>

      <div className="form-field">
        <label className="input-label" htmlFor="item-site">
          Site / title
        </label>
        <input
          id="item-site"
          className="input"
          placeholder={placeholderForSite}
          value={site}
          onChange={(e) => setSite(e.target.value)}
        />
      </div>

      {
        <>
          <div className="form-row">
            <div className="form-field">
              <label className="input-label" htmlFor="item-username">
                Username
              </label>
              <input
                id="item-username"
                className="input"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                autoComplete="username"
              />
            </div>
            <div className="form-field">
              <label className="input-label" htmlFor="item-password">
                Password
              </label>
              <div className="input-with-toggle">
                <input
                  id="item-password"
                  className="input"
                  type={showPassword ? "text" : "password"}
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  autoComplete="current-password"
                />
                <button
                  type="button"
                  className="input-toggle"
                  onClick={() => setShowPassword((prev) => !prev)}
                  aria-label={showPassword ? "Hide password" : "Show password"}
                >
                  {showPassword ? "🙈" : "👁"}
                </button>
              </div>
            </div>
          </div>
        </>
      }

      <div className="form-field">
        <label className="input-label" htmlFor="item-notes">
          Notes
        </label>
        <textarea
          id="item-notes"
          className="textarea"
          rows={5}
          value={notes}
          onChange={(e) => setNotes(e.target.value)}
        />
      </div>

      {err && <div className="error-text">{err}</div>}

      <div className="split-actions">
        <button type="submit" className="btn btn-primary">
          {submitLabel}
        </button>
        <button type="button" className="btn btn-ghost" onClick={onCancel}>
          Cancel
        </button>
        {typeof onDelete === "function" && (
          <button type="button" className="btn btn-danger" onClick={onDelete}>
            🗑 Delete
          </button>
        )}
      </div>
    </form>
  );
}
