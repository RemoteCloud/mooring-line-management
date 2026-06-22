// Auth context, types and hooks. Kept separate from the provider component so the
// provider file only exports a component (satisfies react-refresh/only-export-components).
import { createContext, useContext } from "react";

export type AuthStatus = "loading" | "authenticated" | "unauthenticated";

export type AuthUser = {
  id: string;
  email: string;
  name: string;
  sub: string;
};

export type AccessLevel = "denied" | "view" | "edit";

export type AuthPermissions = {
  admin: boolean;
  level: AccessLevel;
  canRead: boolean;
  canWrite: boolean;
};

export type SessionResponse = {
  authenticated: boolean;
  user?: AuthUser;
  groups?: string[];
  permissions?: AuthPermissions;
};

export type AuthState = {
  status: AuthStatus;
  user: AuthUser | null;
  groups: string[];
  permissions: AuthPermissions;
  login: (returnTo?: string) => void;
  logout: () => Promise<void>;
  refresh: () => Promise<void>;
};

export const DEFAULT_PERMISSIONS: AuthPermissions = {
  admin: false,
  level: "denied",
  canRead: false,
  canWrite: false,
};

export const AuthContext = createContext<AuthState | null>(null);

export function useAuth(): AuthState {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error("useAuth must be used within <AuthProvider>");
  }
  return ctx;
}

// Convenience hook for permission-gating write actions across feature pages.
export function useCanWrite(): boolean {
  return useAuth().permissions.canWrite;
}

// Convenience hook for admin-gating (settings / access control).
export function useIsAdmin(): boolean {
  return useAuth().permissions.admin;
}

// Convenience hook for reading the full permissions object.
export function usePermissions(): AuthPermissions {
  return useAuth().permissions;
}
