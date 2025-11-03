// frontend/src/components/ItemForm.jsx
import { useState, useMemo } from "react";

function digitsOnly(s) { return (s || "").replace(/\D/g, ""); }
function luhnCheck(num) {
  const s = digitsOnly(num);
  if (!s) return false;
  let sum = 0, dbl = false;
  for (let i = s.length - 1; i >= 0; i--) {
    let n = s.charCodeAt(i) - 48;
    if (dbl) { n *= 2; if (n > 9) n -= 9; }
    sum += n; dbl = !dbl;
  }
  return sum % 10 === 0;
}

export default function ItemForm({
  initial = { type: "login", fields: {} },
  onSubmit,
  onCancel,
  submitLabel = "Save",
  onDelete, // optional
}) {
  const [type, setType] = useState((initial.type || "login").toLowerCase());
  const kind = (type || "").toLowerCase();

  // COMMON
  const [site, setSite] = useState(initial.fields?.site || initial.fields?.title || "");
  const [notes, setNotes] = useState(initial.fields?.notes || "");

  // LOGIN
  const [username, setUsername] = useState(initial.fields?.username || initial.fields?.user || "");
  const [password, setPassword] = useState(initial.fields?.password || "");

  // CARD
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
      if (!site) return setErr("Please enter a site/title.");
      payload.fields = { site, username, password, notes };

    } else if (kind === "card") {
      if (!cardholder) return setErr("Please enter the cardholder name.");
      if (!cardNumber) return setErr("Please enter the card number.");
      if (!luhnCheck(cardNumber)) return setErr("Card number failed Luhn check.");
      if (!expMonth || !expYear) return setErr("Please enter expiry month and year.");
      if (!cvv) return setErr("Please enter CVV.");

      const numDigits = digitsOnly(cardNumber);
      const last4 = numDigits.slice(-4);

      payload.fields = {
        cardholder,
        number: numDigits,
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
    <form onSubmit={submit} style={{ maxWidth: 520 }}>
      <label>Type</label><br/>
      <select value={kind} onChange={(e)=>setType((e.target.value||"").toLowerCase())}>
        <option value="login">Login</option>
        <option value="note">Secure Note</option>
        <option value="card">Card</option>
      </select>

      <div style={{ height: 12 }} />
      <label>Site / Title</label><br/>
      <input
        placeholder={placeholderForSite}
        value={site}
        onChange={(e) => setSite(e.target.value)}
      />

      <div style={{ height: 12 }} />

      {kind === "login" && (
        <>
          <label>Username</label><br/>
          <input value={username} onChange={(e)=>setUsername(e.target.value)} />
          <div style={{ height: 12 }} />
          <label>Password</label><br/>
          <input type="password" value={password} onChange={(e)=>setPassword(e.target.value)} />
          <div style={{ height: 12 }} />
        </>
      )}

      {kind === "card" && (
        <>
          <label>Cardholder</label><br/>
          <input placeholder="Full name" value={cardholder} onChange={(e)=>setCardholder(e.target.value)} />

          <div style={{ height: 12 }} />
          <label>Card Number</label><br/>
          <input
            placeholder="1234 5678 9012 3456"
            value={cardNumber}
            onChange={(e)=>setCardNumber(e.target.value)}
            inputMode="numeric"
            autoComplete="cc-number"
          />

          <div style={{ display: "flex", gap: 12, marginTop: 12 }}>
            <div style={{ flex: 1 }}>
              <label>Exp. Month (MM)</label><br/>
              <input placeholder="MM" value={expMonth} onChange={(e)=>setExpMonth(e.target.value)} autoComplete="cc-exp-month" />
            </div>
            <div style={{ flex: 1 }}>
              <label>Exp. Year (YY or YYYY)</label><br/>
              <input placeholder="YY" value={expYear} onChange={(e)=>setExpYear(e.target.value)} autoComplete="cc-exp-year" />
            </div>
            <div style={{ flex: 1 }}>
              <label>CVV</label><br/>
              <input placeholder="CVV" value={cvv} onChange={(e)=>setCvv(e.target.value)} inputMode="numeric" autoComplete="cc-csc" />
            </div>
          </div>

          <div style={{ height: 12 }} />
          <label>Network</label><br/>
          <input placeholder="Visa / Mastercard / Amex" value={network} onChange={(e)=>setNetwork(e.target.value)} />

          <div style={{ height: 12 }} />
        </>
      )}

      <label>Notes</label><br/>
      <textarea rows={5} value={notes} onChange={(e)=>setNotes(e.target.value)} />

      <div style={{ height: 12 }} />
      {err && <div style={{ color: "#f55", marginBottom: 8 }}>{err}</div>}

      <button type="submit">{submitLabel}</button>{" "}
      <button type="button" onClick={onCancel}>Cancel</button>{" "}
      {typeof onDelete === "function" && (
        <button
          type="button"
          onClick={onDelete}
          title="Delete"
          aria-label="Delete"
          style={{ marginLeft: 8 }}
        >
          🗑 Delete
        </button>
      )}
    </form>
  );
}
