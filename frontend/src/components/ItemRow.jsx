// frontend/src/components/ItemRow.jsx
import { Link } from "react-router-dom";

function last4Digits(s) {
  const d = (s || "").replace(/\D/g, "");
  return d.slice(-4);
}

export default function ItemRow({ item }) {
  const t = (item?.type || "").toLowerCase();
  const isCard = t === "card";
  const isNote = t === "note" || t === "secure note" || t === "secure-note";
  const f = item?.fields || {};

  let siteTitle = f.site || f.title || f.name || "(untitled)";
  let uname = f.username || f.user || "—";

  if (isCard) {
    const l4 = last4Digits(f.number);
    const holder = f.cardholder || "Card";
    siteTitle = l4 ? `${holder} · •••• ${l4}` : holder;
    uname = f.network || "—";
  } else if (isNote) {
    uname = "—";
  }

  return (
    <tr>
      <td style={{ padding: 8 }}>{siteTitle}</td>
      <td style={{ padding: 8 }}>{uname}</td>
      <td style={{ padding: 8 }}>{t}</td>
      <td style={{ padding: 8, textAlign: "right", whiteSpace: "nowrap" }}>
        <Link to={`/items/${item.id}`}><button>View</button></Link>{" "}
        <Link to={`/items/${item.id}/edit`}><button>Edit</button></Link>
      </td>
    </tr>
  );
}
