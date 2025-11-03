// src/components/ItemRow.jsx
import { Link } from "react-router-dom";

function ItemRow({ item }) {
  const f = item?.fields || {};
  return (
    <tr>
      <td style={{ padding: 8 }}>{f.site || f.title || "(untitled)"}</td>
      <td style={{ padding: 8 }}>{f.username || "—"}</td>
      <td style={{ padding: 8 }}>{item?.type}</td>
      <td style={{ padding: 8, textAlign: "right", whiteSpace: "nowrap" }}>
        <Link to={`/items/${item.id}`}><button>View</button></Link>{" "}
        <Link to={`/items/${item.id}/edit`}><button>Edit</button></Link>
      </td>
    </tr>
  );
}

export default ItemRow;
