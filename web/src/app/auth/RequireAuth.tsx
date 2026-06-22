// Route guard. Wraps the AppShell branch of the router:
//   loading        → minimal full-screen placeholder
//   unauthenticated → kick off a full-page redirect to the IdP via login()
//   authenticated   → render the protected app
//
// Loop-breaker: every login redirect is a full page load, so we count attempts
// in sessionStorage. If we bounce back unauthenticated too many times in a
// short window — or the backend hands back ?auth_error=… — we STOP redirecting
// and show an error with a manual retry, instead of spamming the IdP forever.
import { useEffect, useState, type ReactNode } from "react";
import { useAuth } from "./authContext";

const ATTEMPTS_KEY = "mlm_login_attempts";
const MAX_ATTEMPTS = 3;
const WINDOW_MS = 30_000;

function recentAttempts(): number[] {
  let arr: number[] = [];
  try {
    arr = JSON.parse(sessionStorage.getItem(ATTEMPTS_KEY) || "[]");
  } catch {
    arr = [];
  }
  const now = Date.now();
  return arr.filter((t) => now - t < WINDOW_MS);
}

function recordAttempt(): number {
  const arr = recentAttempts();
  arr.push(Date.now());
  sessionStorage.setItem(ATTEMPTS_KEY, JSON.stringify(arr));
  return arr.length;
}

function clearAttempts() {
  sessionStorage.removeItem(ATTEMPTS_KEY);
}

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
        padding: 24,
        textAlign: "center",
        color: "var(--text-dim)",
        background: "var(--bg)",
      }}
    >
      {children}
    </div>
  );
}

function authErrorParam(): string | null {
  return new URLSearchParams(window.location.search).get("auth_error");
}

export function RequireAuth({ children }: { children: ReactNode }) {
  const { status, login } = useAuth();
  // Block (show error instead of redirecting) when the backend reported an
  // auth error, or we've already looped past the attempt budget.
  const [blocked, setBlocked] = useState(
    () => !!authErrorParam() || recentAttempts().length >= MAX_ATTEMPTS,
  );
  const reason = authErrorParam();

  useEffect(() => {
    if (status === "authenticated") {
      clearAttempts();
      return;
    }
    if (status !== "unauthenticated") return;

    if (reason || recentAttempts().length >= MAX_ATTEMPTS) {
      setBlocked(true);
      return;
    }
    if (recordAttempt() >= MAX_ATTEMPTS) {
      setBlocked(true);
      return;
    }
    login();
  }, [status, reason, login]);

  if (status === "authenticated") {
    return <>{children}</>;
  }

  if (status === "unauthenticated" && blocked) {
    return (
      <FullScreen>
        <h2 style={{ color: "var(--text)", margin: 0 }}>Sign-in didn’t complete</h2>
        <p style={{ maxWidth: 460 }}>
          We were redirected back without an active session
          {reason ? (
            <>
              {" "}
              (<code>{reason}</code>)
            </>
          ) : null}
          . This usually means a cookie was blocked or the sign-in was
          cancelled. Check that third-party cookies aren’t blocked for this
          site, then try again.
        </p>
        <button
          type="button"
          className="btn"
          onClick={() => {
            clearAttempts();
            login("/");
          }}
        >
          Try signing in again
        </button>
      </FullScreen>
    );
  }

  if (status === "unauthenticated") {
    return (
      <FullScreen>
        <div className="auth-spinner" aria-hidden />
        <div>Redirecting to sign-in…</div>
      </FullScreen>
    );
  }

  return (
    <FullScreen>
      <div className="auth-spinner" aria-hidden />
      <div>Loading…</div>
    </FullScreen>
  );
}
