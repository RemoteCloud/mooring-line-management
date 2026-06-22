// OIDC auth state for the SPA. The session cookie is HttpOnly, so JS cannot read
// it — auth state is determined solely by GET /api/auth/session. The auth endpoints
// are not in the generated OpenAPI schema, so we use plain fetch with credentials so
// the session cookie rides along. All redirects to the IdP/login MUST be full-page
// browser navigations (window.location), never fetch — they are cross-origin 302s.
//
// The session is fetched via React Query (already wired app-wide) so we avoid manual
// effect/setState juggling and get caching + refetch-on-focus for free.
import { useCallback, useMemo, type ReactNode } from "react";
import { useQuery } from "@tanstack/react-query";
import { API_BASE } from "../../config";
import {
  AuthContext,
  DEFAULT_PERMISSIONS,
  type AuthState,
  type AuthStatus,
  type AuthUser,
  type SessionResponse,
} from "./authContext";

async function fetchSession(): Promise<SessionResponse> {
  const res = await fetch(`${API_BASE}/auth/session`, {
    credentials: "include",
    headers: { Accept: "application/json" },
  });
  if (!res.ok) {
    // 401 (or anything else) → not authenticated.
    return { authenticated: false };
  }
  return (await res.json()) as SessionResponse;
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const { data, isLoading, refetch } = useQuery({
    queryKey: ["auth", "session"],
    queryFn: fetchSession,
    staleTime: 60_000,
    retry: false,
    refetchOnWindowFocus: true,
  });

  const authenticated = !!data?.authenticated && !!data.user;
  const status: AuthStatus = isLoading
    ? "loading"
    : authenticated
      ? "authenticated"
      : "unauthenticated";

  const login = useCallback((returnTo?: string) => {
    const raw = returnTo ?? window.location.pathname + window.location.search;
    // Never feed an auth path back as return_to — otherwise a failed login keeps
    // wrapping the previous login URL into the next one (an ever-growing,
    // self-nesting redirect loop). Fall back to the app root.
    const path = raw.split("?")[0];
    const target =
      path.startsWith("/auth/") || path.startsWith("/api/auth/") ? "/" : raw;
    window.location.href =
      `${API_BASE}/auth/login?return_to=` + encodeURIComponent(target);
  }, []);

  const logout = useCallback(async () => {
    try {
      await fetch(`${API_BASE}/auth/logout`, {
        method: "POST",
        credentials: "include",
      });
    } finally {
      window.location.href = "/";
    }
  }, []);

  const refresh = useCallback(async () => {
    await refetch();
  }, [refetch]);

  const value = useMemo<AuthState>(
    () => ({
      status,
      user: authenticated ? (data!.user as AuthUser) : null,
      groups: data?.groups ?? [],
      permissions: data?.permissions ?? DEFAULT_PERMISSIONS,
      login,
      logout,
      refresh,
    }),
    [status, authenticated, data, login, logout, refresh],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}
