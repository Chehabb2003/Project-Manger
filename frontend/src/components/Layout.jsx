import { Link, useNavigate } from "react-router-dom";
import { useAuth } from "../context/AuthContext";
import { lock } from "../lib/api";

export default function Layout({ children }) {
  const { isUnlocked, setUnlocked } = useAuth();
  const navigate = useNavigate();

  async function onLock() {
    try {
      await lock();
    } catch {
      // ignore; best-effort
    } finally {
      setUnlocked(null);
      navigate("/unlock");
    }
  }

  return (
    <div>
      <header
        style={{
          display: "flex",
          alignItems: "center",
          gap: 12,
          padding: "12px 16px",
          borderBottom: "1px solid #eee",
        }}
      >
        <Link to={isUnlocked?.ok ? "/vault" : "/unlock"} style={{ textDecoration: "none" }}>
          <h2 style={{ margin: 0 }}>🔐 Vault</h2>
        </Link>

        <div style={{ marginLeft: "auto", display: "flex", alignItems: "center", gap: 8 }}>
          {isUnlocked?.ok ? (
            <>
              <span style={{ opacity: 0.7 }}>Vault: <code>{isUnlocked.vault}</code></span>
              <Link to="/items/new"><button>+ Add Item</button></Link>
              <button onClick={onLock} title="Lock vault">Lock</button>
            </>
          ) : (
            <Link to="/unlock"><button>Unlock</button></Link>
          )}
        </div>
      </header>

      <main style={{ maxWidth: 1000, margin: "20px auto", padding: "0 16px" }}>
        {children}
      </main>
    </div>
  );
}
