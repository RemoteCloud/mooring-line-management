// Minimal auth gate for the basic API-key system. Renders the app only when a key is
// stored and validates; on any 401 the fetch wrapper clears the key and the gate returns.
import { useEffect, useState, type ReactNode } from "react";
import { API_BASE } from "../config";
import { getApiKey, setApiKey, clearApiKey } from "../api/authKey";

export function UnlockGate({ children }: { children: ReactNode }) {
  const [authed, setAuthed] = useState<boolean>(() => !!getApiKey());

  useEffect(() => {
    const onChange = () => setAuthed(!!getApiKey());
    window.addEventListener("mlm-auth-changed", onChange);
    return () => window.removeEventListener("mlm-auth-changed", onChange);
  }, []);

  if (authed) return <>{children}</>;
  return <UnlockForm />;
}

function UnlockForm() {
  const [key, setKey] = useState("");
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    const k = key.trim();
    if (!k) return;
    setBusy(true);
    setErr(null);
    // Store first so the global fetch wrapper attaches the key, then validate via /me.
    setApiKey(k);
    try {
      const res = await fetch(`${API_BASE}/me`);
      if (!res.ok) {
        clearApiKey();
        setErr(res.status === 401 ? "Key not recognised." : `Validation failed (${res.status}).`);
      }
      // On success, setApiKey already fired mlm-auth-changed → gate unlocks.
    } catch {
      clearApiKey();
      setErr("Could not reach the server.");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="unlock-gate">
      <form className="unlock-card" onSubmit={submit}>
        <h1>Mooring Line Management</h1>
        <p className="muted">Enter your API key to continue.</p>
        <input
          className="input"
          type="password"
          autoFocus
          placeholder="mlm_…"
          value={key}
          onChange={(e) => setKey(e.target.value)}
        />
        {err && <div className="err">{err}</div>}
        <button className="btn" type="submit" disabled={busy || !key.trim()}>
          {busy ? "Checking…" : "Unlock"}
        </button>
      </form>
    </div>
  );
}
