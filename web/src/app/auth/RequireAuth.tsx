// Route guard. Wraps the AppShell branch of the router:
//   loading        → minimal full-screen placeholder
//   unauthenticated → kick off a full-page redirect to the IdP via login()
//   authenticated   → render the protected app
import { useEffect, type ReactNode } from "react";
import { useAuth } from "./authContext";

function FullScreen({ children }: { children: ReactNode }) {
  return (
    <div
      style={{
        minHeight: "100vh",
        display: "flex",
        flexDirection: "column",
        alignItems: "center",
        justifyContent: "center",
        gap: 16,
        color: "var(--text-dim)",
        background: "var(--bg)",
      }}
    >
      <div className="auth-spinner" aria-hidden />
      <div>{children}</div>
    </div>
  );
}

export function RequireAuth({ children }: { children: ReactNode }) {
  const { status, login } = useAuth();

  useEffect(() => {
    if (status === "unauthenticated") {
      login();
    }
  }, [status, login]);

  if (status === "authenticated") {
    return <>{children}</>;
  }

  if (status === "unauthenticated") {
    return <FullScreen>Redirecting to sign-in…</FullScreen>;
  }

  return <FullScreen>Loading…</FullScreen>;
}
