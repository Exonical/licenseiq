"use client";

export const AUTH_STORAGE_KEY = "licenseiq.authToken";
const listeners = new Set<() => void>();

export type AuthProfile = {
  token: string | null;
  authenticated: boolean;
  keyId: string | null;
  label: string;
};

export function readStoredAuthToken() {
  if (typeof window === "undefined") {
    return null;
  }
  const token = window.localStorage.getItem(AUTH_STORAGE_KEY);
  return token && token.trim() ? token.trim() : null;
}

export function writeStoredAuthToken(token: string | null) {
  if (typeof window === "undefined") {
    return;
  }
  if (token && token.trim()) {
    window.localStorage.setItem(AUTH_STORAGE_KEY, token.trim());
  } else {
    window.localStorage.removeItem(AUTH_STORAGE_KEY);
  }
  for (const listener of listeners) {
    listener();
  }
}

export function subscribeStoredAuthToken(listener: () => void) {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

export function parseAuthToken(token: string | null): AuthProfile {
  const trimmed = token?.trim() ?? null;
  if (!trimmed) {
    return { token: null, authenticated: false, keyId: null, label: "Not signed in" };
  }
  const match = /^liq_([^\.]+)\.[^.]+$/.exec(trimmed);
  const keyId = match?.[1] ?? null;
  return {
    token: trimmed,
    authenticated: true,
    keyId,
    label: keyId ? `API key ${keyId}` : "Bearer token",
  };
}
