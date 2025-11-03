import { createContext, useContext, useMemo, useState } from "react";

const AuthCtx = createContext(null);

export function AuthProvider({ children }) {
  const [isUnlocked, setUnlocked] = useState(null); // { ok: true, vault: 'dev.vlt' } after unlock
  const value = useMemo(() => ({ isUnlocked, setUnlocked }), [isUnlocked]);
  return <AuthCtx.Provider value={value}>{children}</AuthCtx.Provider>;
}

export function useAuth() {
  const ctx = useContext(AuthCtx);
  if (!ctx) throw new Error("useAuth must be used within <AuthProvider>");
  return ctx;
}
