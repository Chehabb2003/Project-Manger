// src/components/SearchBox.jsx
import { useEffect, useState } from "react";

function SearchBox({ defaultValue = "", onChange, placeholder = "Search…" }) {
  const [val, setVal] = useState(defaultValue);

  useEffect(() => {
    const t = setTimeout(() => onChange?.(val), 200);
    return () => clearTimeout(t);
  }, [val, onChange]);

  return (
    <input
      value={val}
      onChange={(e) => setVal(e.target.value)}
      placeholder={placeholder}
      style={{ flex: 1 }}
    />
  );
}

export default SearchBox;
