// frontend/src/components/ItemForm.jsx
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
  const [type, setType] = useState((initial.type || "login").toLowerCase());
  const kind = (type || "").toLowerCase();

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

  const placeholderForSite = useMemo(
    () => (kind === "login" ? "example.com" : "Optional display title"),
    [kind]
  );

  async function submit(e) {
    e.preventDefault();
    setErr("");

    const payload = { type: kind, fields: {} };

    if (kind === "login") {
      if (!site) return setErr("Please add a website or title for this login.");
      payload.fields = { site, username, password, notes };
    } else if (kind === "card") {
      if (!cardholder) return setErr("Please enter the cardholder name.");
      if (!cardNumber) return setErr("Please enter the card number.");
      if (!luhnCheck(cardNumber)) return setErr("Card number failed the validity check.");
      if (!expMonth || !expYear) return setErr("Please enter the expiration month and year.");
      if (!cvv) return setErr("Please enter the CVV/CVC.");

      const digits = digitsOnly(cardNumber);
      const last4 = digits.slice(-4);

      payload.fields = {
        cardholder,
        number: digits,
        exp_month: expMonth,
        exp_year: expYear,
        cvv,
        network,
        notes,
        site: site || (last4 ? `Card •••• ${last4}` : "Card"),
      };
    } else if (kind === "note") {
      payload.fields = { site, notes };
    } else {
      payload.fields = { site, notes };
    }

    await onSubmit(payload);
  }

  return (
    <form className="form-card" onSubmit={submit}>
      <div className="form-field">
        <label className="input-label" htmlFor="item-type">
          Item type
        </label>
        <select
          id="item-type"
          className="select"
          value={kind}
          onChange={(e) => setType((e.target.value || "").toLowerCase())}
        >
          <option value="login">Login</option>
          <option value="note">Secure note</option>
          <option value="card">Card</option>
        </select>
      </div>

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

      {kind === "login" && (
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
      )}

      {kind === "card" && (
        <>
          <div className="form-field">
            <label className="input-label" htmlFor="card-holder">
              Cardholder
            </label>
            <input
              id="card-holder"
              className="input"
              placeholder="Full name as it appears on the card"
              value={cardholder}
              onChange={(e) => setCardholder(e.target.value)}
            />
          </div>

          <div className="form-field">
            <label className="input-label" htmlFor="card-number">
              Card number
            </label>
            <input
              id="card-number"
              className="input"
              placeholder="1234 5678 9012 3456"
              value={cardNumber}
              onChange={(e) => setCardNumber(e.target.value)}
              inputMode="numeric"
              autoComplete="cc-number"
            />
          </div>

          <div className="form-row">
            <div className="form-field">
              <label className="input-label" htmlFor="card-exp-month">
                Exp. month
              </label>
              <input
                id="card-exp-month"
                className="input"
                placeholder="MM"
                value={expMonth}
                onChange={(e) => setExpMonth(e.target.value)}
                autoComplete="cc-exp-month"
              />
            </div>
            <div className="form-field">
              <label className="input-label" htmlFor="card-exp-year">
                Exp. year
              </label>
              <input
                id="card-exp-year"
                className="input"
                placeholder="YY or YYYY"
                value={expYear}
                onChange={(e) => setExpYear(e.target.value)}
                autoComplete="cc-exp-year"
              />
            </div>
            <div className="form-field">
              <label className="input-label" htmlFor="card-cvv">
                CVV
              </label>
              <input
                id="card-cvv"
                className="input"
                placeholder="CVV"
                value={cvv}
                onChange={(e) => setCvv(e.target.value)}
                inputMode="numeric"
                autoComplete="cc-csc"
              />
            </div>
          </div>

          <div className="form-field">
            <label className="input-label" htmlFor="card-network">
              Network
            </label>
            <input
              id="card-network"
              className="input"
              placeholder="Visa / Mastercard / Amex"
              value={network}
              onChange={(e) => setNetwork(e.target.value)}
            />
          </div>
        </>
      )}

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
