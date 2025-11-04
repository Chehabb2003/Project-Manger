// frontend/src/components/ItemRow.jsx
import { Link } from "react-router-dom";

function last4Digits(s) {
  const d = (s || "").replace(/\D/g, "");
  return d.slice(-4);
}

function resolveBadge() {
  return { className: "badge badge--login", label: "Login" };
}

export default function ItemRow({ item }) {
  const f = item?.fields || {};
  const badge = resolveBadge();

  let title = f.site || f.title || f.name || "(untitled)";
  let subtitle = f.username || f.user || "";
  const icon = "🔑";

  const updatedAt = item?.updated_at ? new Date(item.updated_at) : null;
  const timestamp = updatedAt ? updatedAt.toLocaleDateString() : "";

  return (
    <article className="item-card">
      <div className="item-card__top">
        <div>
          <div className="item-card__title">
            <span aria-hidden="true" style={{ marginRight: 8 }}>{icon}</span>
            {title}
          </div>
          <div className="item-card__meta">{subtitle || "—"}</div>
        </div>
        <span className={badge.className}>{badge.label}</span>
      </div>

      {timestamp && (
        <div className="helper-text">
          <span aria-hidden="true">🕒</span> Updated {timestamp}
        </div>
      )}

      <div className="item-card__footer">
        <Link to={`/items/${item.id}`} className="btn btn-ghost">
          <span aria-hidden="true">👁</span>
          View
        </Link>
        <Link to={`/items/${item.id}/edit`} className="btn btn-primary">
          <span aria-hidden="true">✏️</span>
          Edit
        </Link>
      </div>
    </article>
  );
}
