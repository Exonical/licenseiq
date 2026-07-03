"use client";

import type { ReactNode } from "react";
import { createContext, useContext, useEffect, useMemo, useSyncExternalStore } from "react";
import { parseAuthToken, readStoredAuthToken, subscribeStoredAuthToken, writeStoredAuthToken } from "@/lib/auth";
import { setAuthTokenProvider } from "@/lib/api/client";

export type AuthContextValue = {
  token: string | null;
  authenticated: boolean;
  keyId: string | null;
  label: string;
  setToken: (token: string | null) => void;
  signOut: () => void;
};

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const hydrated = useSyncExternalStore(() => () => {}, () => true, () => false);
  const token = useSyncExternalStore(
    subscribeStoredAuthToken,
    readStoredAuthToken,
    () => null,
  );

  useEffect(() => {
    setAuthTokenProvider(() => token ?? undefined);
  }, [token]);

  const value = useMemo<AuthContextValue>(() => {
    const profile = parseAuthToken(token);
    return {
      token: profile.token,
      authenticated: profile.authenticated,
      keyId: profile.keyId,
      label: profile.label,
      setToken: writeStoredAuthToken,
      signOut: () => writeStoredAuthToken(null),
    };
  }, [token]);

  if (!hydrated) {
    return null;
  }

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error("useAuth must be used within AuthProvider");
  }
  return context;
}
