import { Fragment, useMemo, useState } from "react";
import { useVessel } from "../../app/VesselContext";
import {
  useWebhooks, useWebhookEvents, useCreateWebhook, useUpdateWebhook, useDeleteWebhook, useTestWebhook,
  type WebhookSubscription, type WebhookInput,
} from "./api";
import { UsersSection } from "./UsersSection";

export function SettingsPage() {
  return (
    <>
      <h1 className="page-title">Settings</h1>
      <p className="page-sub">Integrations and delivery configuration.</p>
      <WebhooksSection />
      <UsersSection />
    </>
  );
}

function WebhooksSection() {
  const { vesselId } = useVessel();
  const { data: hooks = [], isLoading } = useWebhooks(vesselId);
  const del = useDeleteWebhook(vesselId);
  const test = useTestWebhook();
  const [editing, setEditing] = useState<WebhookSubscription | "new" | null>(null);
  const [testMsg, setTestMsg] = useState<{ id: string; ok: boolean; text: string } | null>(null);

  const runTest = async (h: WebhookSubscription) => {
    setTestMsg(null);
    try {
      await test.mutateAsync(h.id);
      setTestMsg({ id: h.id, ok: true, text: "Test delivery sent — endpoint accepted it." });
    } catch (e) {
      setTestMsg({ id: h.id, ok: false, text: (e as Error).message });
    }
  };

  return (
    <section className="card" style={{ maxWidth: 900 }}>
      <div className="section-head">
        <div>
          <h2 style={{ margin: 0 }}>Webhooks</h2>
          <p className="muted" style={{ margin: "4px 0 0" }}>
            POST domain events (line registered, turned, inspection logged…) to your systems with custom headers and a templated payload.
          </p>
        </div>
        <button className="btn" onClick={() => setEditing("new")}>+ Add webhook</button>
      </div>

      {isLoading ? (
        <p className="muted">Loading…</p>
      ) : hooks.length === 0 ? (
        <p className="muted">No webhooks yet. Add one to start delivering events.</p>
      ) : (
        <div className="table-wrap">
          <table className="grid">
            <thead>
              <tr><th>Name</th><th>URL</th><th>Events</th><th>Status</th><th></th></tr>
            </thead>
            <tbody>
              {hooks.map((h) => (
                <Fragment key={h.id}>
                  <tr>
                    <td>{h.name || <span className="muted">—</span>}</td>
                    <td className="mono" style={{ wordBreak: "break-all" }}>{h.url}</td>
                    <td>
                      {h.events.length === 0
                        ? <span className="badge">all events</span>
                        : h.events.map((e) => <span key={e} className="badge" style={{ marginRight: 4 }}>{e}</span>)}
                    </td>
                    <td>
                      <span className={"pill " + (h.active ? "good" : "muted-pill")}>{h.active ? "Active" : "Paused"}</span>
                    </td>
                    <td style={{ whiteSpace: "nowrap", textAlign: "right" }}>
                      <button className="linkbtn" onClick={() => runTest(h)} disabled={test.isPending}>Test</button>
                      <button className="linkbtn" style={{ marginLeft: 12 }} onClick={() => setEditing(h)}>Edit</button>
                      <button className="linkbtn danger" style={{ marginLeft: 12 }}
                        onClick={() => { if (confirm(`Delete webhook ${h.name || h.url}?`)) del.mutate(h.id); }}>Delete</button>
                    </td>
                  </tr>
                  {testMsg?.id === h.id && (
                    <tr>
                      <td colSpan={5} className={testMsg.ok ? "ok-row" : "err-row"}>{testMsg.text}</td>
                    </tr>
                  )}
                </Fragment>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {editing && (
        <WebhookDialog
          vesselId={vesselId}
          existing={editing === "new" ? null : editing}
          onClose={() => setEditing(null)}
        />
      )}
    </section>
  );
}

const DEFAULT_TEMPLATE = `{
  "event": "{{event.type}}",
  "at": "{{event.time}}",
  "vessel": "{{vessel.id}}",
  "line": "{{payload.lineId}}"
}`;

function WebhookDialog({ vesselId, existing, onClose }: {
  vesselId?: string;
  existing: WebhookSubscription | null;
  onClose: () => void;
}) {
  const { data: catalog = [] } = useWebhookEvents();
  const create = useCreateWebhook(vesselId);
  const update = useUpdateWebhook(vesselId);
  const editing = !!existing;

  const [name, setName] = useState(existing?.name ?? "");
  const [url, setUrl] = useState(existing?.url ?? "");
  const [secret, setSecret] = useState("");
  const [events, setEvents] = useState<string[]>(existing?.events ?? []);
  const [headerRows, setHeaderRows] = useState<{ k: string; v: string }[]>(
    Object.entries(existing?.headers ?? {}).map(([k, v]) => ({ k, v })),
  );
  const [template, setTemplate] = useState(existing?.payloadTemplate ?? "");
  const [active, setActive] = useState(existing?.active ?? true);
  const [err, setErr] = useState<string | null>(null);

  const allVars = useMemo(() => {
    const set = new Set<string>();
    catalog.forEach((e) => e.variables.forEach((v) => set.add(v)));
    return Array.from(set).sort();
  }, [catalog]);

  const toggleEvent = (t: string) =>
    setEvents((prev) => (prev.includes(t) ? prev.filter((x) => x !== t) : [...prev, t]));

  const valid = url.trim().length > 0;
  const busy = create.isPending || update.isPending;

  const submit = async () => {
    setErr(null);
    const headers: Record<string, string> = {};
    for (const { k, v } of headerRows) if (k.trim()) headers[k.trim()] = v;
    const body: WebhookInput = {
      name: name.trim(),
      url: url.trim(),
      secret: secret.trim() || undefined,
      events,
      headers,
      payloadTemplate: template.trim() || undefined,
      active,
    };
    try {
      if (editing && existing) await update.mutateAsync({ id: existing.id, body });
      else await create.mutateAsync(body);
      onClose();
    } catch (e) {
      setErr((e as Error).message);
    }
  };

  return (
    <div className="overlay" onClick={onClose}>
      <div className="dialog" style={{ width: 600 }} onClick={(e) => e.stopPropagation()}>
        <h3>{editing ? "Edit webhook" : "Add webhook"}</h3>

        <div className="field">
          <label>Name</label>
          <input className="input" value={name} placeholder="e.g. Maintenance system" onChange={(e) => setName(e.target.value)} />
        </div>
        <div className="field">
          <label>Endpoint URL</label>
          <input className="input" value={url} placeholder="https://example.com/hooks/mooring" onChange={(e) => setUrl(e.target.value)} />
        </div>
        <div className="field">
          <label>Signing secret {editing && existing?.hasSecret && <span className="muted" style={{ fontWeight: 400 }}>· set — leave blank to keep</span>}</label>
          <input className="input" type="password" value={secret} placeholder={editing ? "•••••• (unchanged)" : "HMAC-SHA256 key (optional)"} onChange={(e) => setSecret(e.target.value)} />
          <span className="muted" style={{ fontSize: 12 }}>Sent as <code>X-Signature-256: sha256=…</code> over the request body.</span>
        </div>

        <div className="field">
          <label>Events <span className="muted" style={{ fontWeight: 400 }}>· none selected = all events</span></label>
          <div className="check-grid">
            {catalog.map((e) => (
              <label key={e.type} className="check" title={e.description}>
                <input type="checkbox" checked={events.includes(e.type)} onChange={() => toggleEvent(e.type)} />
                <span>{e.type}</span>
              </label>
            ))}
          </div>
        </div>

        <div className="field">
          <label>Custom headers</label>
          {headerRows.map((row, i) => (
            <div key={i} className="kv-row">
              <input className="input" placeholder="Header" value={row.k}
                onChange={(e) => setHeaderRows((r) => r.map((x, idx) => idx === i ? { ...x, k: e.target.value } : x))} />
              <input className="input" placeholder="Value (supports {{variables}})" value={row.v}
                onChange={(e) => setHeaderRows((r) => r.map((x, idx) => idx === i ? { ...x, v: e.target.value } : x))} />
              <button className="btn ghost" onClick={() => setHeaderRows((r) => r.filter((_, idx) => idx !== i))}>−</button>
            </div>
          ))}
          <button className="btn ghost" onClick={() => setHeaderRows((r) => [...r, { k: "", v: "" }])}>+ Header</button>
        </div>

        <div className="field">
          <label>Payload template <span className="muted" style={{ fontWeight: 400 }}>· empty = default JSON envelope</span></label>
          <textarea className="input" rows={6} style={{ fontFamily: "var(--mono)", fontSize: 13 }}
            value={template} placeholder={DEFAULT_TEMPLATE} onChange={(e) => setTemplate(e.target.value)} />
          <button className="linkbtn" onClick={() => setTemplate(DEFAULT_TEMPLATE)}>Insert example</button>
          {allVars.length > 0 && (
            <details style={{ marginTop: 6 }}>
              <summary className="muted" style={{ fontSize: 12, cursor: "pointer" }}>Available variables</summary>
              <div className="var-list">
                {allVars.map((v) => <code key={v}>{`{{${v}}}`}</code>)}
              </div>
            </details>
          )}
        </div>

        <label className="check" style={{ marginTop: 4 }}>
          <input type="checkbox" checked={active} onChange={(e) => setActive(e.target.checked)} />
          <span>Active</span>
        </label>

        {err && <div className="err" style={{ marginTop: 10 }}>{err}</div>}

        <div className="dialog-actions">
          <button className="btn ghost" onClick={onClose}>Cancel</button>
          <button className="btn" onClick={submit} disabled={!valid || busy}>{busy ? "Saving…" : editing ? "Save changes" : "Add webhook"}</button>
        </div>
      </div>
    </div>
  );
}
