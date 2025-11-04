import { useEffect, useState } from "react";
import { Link, useLocation, useNavigate } from "react-router-dom";
import { session as getSession, lock as apiLock } from "../lib/api";

export default function Layout({ children }) {
  const [isUnlocked, setIsUnlocked] = useState(false);
  const [vaultName, setVaultName] = useState(
    typeof localStorage !== "undefined" ? localStorage.getItem("vaultName") || "" : ""
  );
  const [username, setUsername] = useState(
    typeof localStorage !== "undefined" ? localStorage.getItem("username") || "" : ""
  );
  const nav = useNavigate();
  const loc = useLocation();

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const s = await getSession();
        if (cancelled) return;
        setIsUnlocked(!!s.unlocked);
        setVaultName(s.vault || "");
        setUsername(s.user || "");
        try {
          localStorage.setItem("vaultName", s.vault || "");
          localStorage.setItem("username", s.user || "");
        } catch {
        }
      } catch {
        if (!cancelled) {
          setIsUnlocked(false);
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [loc.pathname]);

  async function onLockUnlock() {
    if (isUnlocked) {
      try {
        await apiLock();
      } catch {
      }
      setIsUnlocked(false);
      nav("/unlock");
    } else {
      nav("/unlock");
    }
  }

  const userLabel = username ? `Signed in as ${username}` : "";

  return (
    <div className="app-shell">
      <header className="app-header">
        <Link to="/vault" className="brand">
          <span className="brand-icon" aria-hidden="true">🔐</span>
          <span className="brand-text">
            <strong>VaultCraft</strong>
            <span>Secure password vault</span>
          </span>
        </Link>

        <div className="header-actions">
          {userLabel && (
            <span className="vault-chip">
              <span aria-hidden="true">👤</span>
              {userLabel}
            </span>
          )}

          {isUnlocked && (
            <Link to="/items/new" className="btn btn-primary">
              <span aria-hidden="true">＋</span>
              New Item
            </Link>
          )}

          {isUnlocked && (
            <Link to="/settings/password" className="btn btn-ghost">
              <span aria-hidden="true">🛡</span>
              Security
            </Link>
          )}

          <button type="button" className="btn btn-ghost" onClick={onLockUnlock}>
            <span aria-hidden="true">{isUnlocked ? "🔒" : "🔓"}</span>
            {isUnlocked ? "Lock Vault" : "Unlock"}
          </button>
        </div>
      </header>

      <main className="app-main">{children}</main>

      <footer className="app-footer">
        <span>VaultCraft · crafted for effortless security</span>
        <span>Stay encrypted · Stay serene</span>
      </footer>
    </div>
  );
}
