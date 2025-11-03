// frontend/src/components/Layout.jsx
import { useEffect, useState } from "react";
import { Link, useLocation, useNavigate } from "react-router-dom";
import { session as getSession, lock as apiLock } from "../lib/api";

export default function Layout({ children }) {
  const [isUnlocked, setIsUnlocked] = useState(false);
  const [vaultName, setVaultName] = useState(
    typeof localStorage !== "undefined" ? localStorage.getItem("vaultName") || "" : ""
  );
  const nav = useNavigate();
  const loc = useLocation();

  async function probe() {
    try {
      const s = await getSession();
      setIsUnlocked(!!s.unlocked);
      setVaultName(s.vault || "");
      try { localStorage.setItem("vaultName", s.vault || ""); } catch {}
    } catch {
      setIsUnlocked(false);
    }
  }

  useEffect(() => { probe(); }, [loc.pathname]);

  async function onLockUnlock() {
    if (isUnlocked) {
      try { await apiLock(); } catch {}
      setIsUnlocked(false);
      nav("/unlock");
    } else {
      nav("/unlock");
    }
  }

  const headerBtn = {
    padding: "10px 16px",
    borderRadius: 10,
    border: "1px solid #444",
    background: "transparent",
    color: "inherit",
    cursor: "pointer",
  };

  return (
    <div style={{ maxWidth: 900, margin: "0 auto", padding: 16 }}>
      <header style={{ display: "flex", alignItems: "center", gap: 14, marginBottom: 12 }}>
        <Link to="/vault" style={{ textDecoration: "none", color: "inherit", fontWeight: 700 }}>
          <span role="img" aria-label="lock">🔒</span> <span>Vault</span>
        </Link>
        <div style={{ opacity: 0.7 }}>{vaultName ? `Vault: ${vaultName}` : ""}</div>
        <div style={{ marginLeft: "auto", display: "flex", gap: 10 }}>
          {isUnlocked && (
            <Link to="/items/new"><button style={headerBtn}>+ Add Item</button></Link>
          )}
          <button style={headerBtn} onClick={onLockUnlock}>
            {isUnlocked ? "Lock" : "Unlock"}
          </button>
        </div>
      </header>
      <hr style={{ borderColor: "#333" }} />
      <main>{children}</main>
    </div>
  );
}
