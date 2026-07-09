// Users & API keys panel (basic, temporary auth). Admin-only: hidden for non-admins.
import { Fragment, useState } from "react";
import { CopyButton } from "../../components/ui";
import { dateLabel } from "../../lib/format";
import {
  useMe, useUsers, useCreateUser, useUpdateUser,
  useApiKeys, useCreateApiKey, useRevokeApiKey,
  type Role, type User, type NewApiKey, type CreateUserBody,
} from "./usersApi";

const ROLES: Role[] = ["admin", "vessel_user", "readonly"];

export function UsersSection() {
  const { data: me } = useMe();
  const isAdmin = me?.role === "admin";
  const { data: users = [], isLoading } = useUsers(!!isAdmin);
  const [adding, setAdding] = useState(false);
  const [openKeys, setOpenKeys] = useState<string | null>(null);

  if (!isAdmin) return null;

  return (
    <section className="card" style={{ maxWidth: 900, marginTop: 24 }}>
      <div className="section-head">
        <div>
          <h2 style={{ margin: 0 }}>Users &amp; API keys</h2>
          <p className="muted" style={{ margin: "4px 0 0" }}>
            Basic access control. Each user authenticates with an API key; revoke a key to cut off access.
          </p>
        </div>
        <button className="btn" onClick={() => setAdding(true)}>+ Add user</button>
      </div>

      {isLoading ? (
        <p className="muted">Loading…</p>
      ) : (
        <div className="table-wrap">
          <table className="grid">
            <thead>
              <tr><th>Name</th><th>Email</th><th>Role</th><th>Status</th><th></th></tr>
            </thead>
            <tbody>
              {users.map((u) => (
                <Fragment key={u.id}>
                  <UserRow
                    user={u}
                    keysOpen={openKeys === u.id}
                    onToggleKeys={() => setOpenKeys((p) => (p === u.id ? null : u.id))}
                  />
                  {openKeys === u.id && (
                    <tr>
                      <td colSpan={5} style={{ background: "var(--surface-2, rgba(0,0,0,0.03))" }}>
                        <ApiKeysPanel userId={u.id} />
                      </td>
                    </tr>
                  )}
                </Fragment>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {adding && <AddUserDialog onClose={() => setAdding(false)} />}
    </section>
  );
}

function UserRow({ user, keysOpen, onToggleKeys }: {
  user: User;
  keysOpen: boolean;
  onToggleKeys: () => void;
}) {
  const update = useUpdateUser();
  return (
    <tr>
      <td>{user.name}</td>
      <td className="mono" style={{ wordBreak: "break-all" }}>{user.email}</td>
      <td>
        <select
          className="input"
          value={user.role}
          disabled={update.isPending}
          onChange={(e) => update.mutate({ id: user.id, role: e.target.value as Role })}
        >
          {ROLES.map((r) => <option key={r} value={r}>{r}</option>)}
        </select>
      </td>
      <td>
        <span className={"pill " + (user.active ? "good" : "muted-pill")}>{user.active ? "Active" : "Disabled"}</span>
      </td>
      <td style={{ whiteSpace: "nowrap", textAlign: "right" }}>
        <button className="linkbtn" onClick={onToggleKeys}>{keysOpen ? "Hide keys" : "Keys"}</button>
        <button
          className="linkbtn"
          style={{ marginLeft: 12 }}
          disabled={update.isPending}
          onClick={() => update.mutate({ id: user.id, active: !user.active })}
        >
          {user.active ? "Disable" : "Enable"}
        </button>
      </td>
    </tr>
  );
}

function ApiKeysPanel({ userId }: { userId: string }) {
  const { data: keys = [], isLoading } = useApiKeys(userId);
  const create = useCreateApiKey();
  const revoke = useRevokeApiKey();
  const [issued, setIssued] = useState<NewApiKey | null>(null);
  const [name, setName] = useState("");

  const issue = async () => {
    const n = name.trim() || "API key";
    const k = await create.mutateAsync({ userId, name: n });
    if (k) setIssued(k);
    setName("");
  };

  return (
    <div style={{ padding: "10px 4px" }}>
      <div style={{ display: "flex", gap: 8, marginBottom: 10 }}>
        <input
          className="input"
          placeholder="Key label (e.g. Bridge tablet)"
          value={name}
          onChange={(e) => setName(e.target.value)}
          style={{ maxWidth: 280 }}
        />
        <button className="btn" onClick={issue} disabled={create.isPending}>
          {create.isPending ? "Issuing…" : "Issue key"}
        </button>
      </div>

      {isLoading ? (
        <p className="muted">Loading keys…</p>
      ) : keys.length === 0 ? (
        <p className="muted">No keys yet.</p>
      ) : (
        <table className="grid">
          <thead>
            <tr><th>Label</th><th>Prefix</th><th>Created</th><th>Last used</th><th>Status</th><th></th></tr>
          </thead>
          <tbody>
            {keys.map((k) => (
              <tr key={k.id}>
                <td>{k.name}</td>
                <td className="mono">{k.keyPrefix}…</td>
                <td>{dateLabel(k.createdAt)}</td>
                <td>{k.lastUsedAt ? dateLabel(k.lastUsedAt) : <span className="muted">never</span>}</td>
                <td>
                  <span className={"pill " + (k.revokedAt ? "muted-pill" : "good")}>{k.revokedAt ? "Revoked" : "Active"}</span>
                </td>
                <td style={{ textAlign: "right" }}>
                  {!k.revokedAt && (
                    <button
                      className="linkbtn danger"
                      disabled={revoke.isPending}
                      onClick={() => { if (confirm(`Revoke key “${k.name}”? This cannot be undone.`)) revoke.mutate({ id: k.id, userId }); }}
                    >
                      Revoke
                    </button>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      {issued && <IssuedKeyDialog k={issued} onClose={() => setIssued(null)} />}
    </div>
  );
}

function IssuedKeyDialog({ k, onClose }: { k: NewApiKey; onClose: () => void }) {
  return (
    <div className="overlay" onClick={onClose}>
      <div className="dialog" style={{ width: 540 }} onClick={(e) => e.stopPropagation()}>
        <h3>API key issued</h3>
        <p className="muted" style={{ marginTop: -6 }}>
          Copy it now — it is shown only once and cannot be retrieved later.
        </p>
        <div className="field">
          <label>{k.name}</label>
          <div style={{ display: "flex", gap: 8, alignItems: "center" }}>
            <input className="input mono" readOnly value={k.plainKey} onFocus={(e) => e.target.select()} />
            <CopyButton value={k.plainKey} />
          </div>
        </div>
        <div className="dialog-actions">
          <button className="btn" onClick={onClose}>Done</button>
        </div>
      </div>
    </div>
  );
}

function AddUserDialog({ onClose }: { onClose: () => void }) {
  const create = useCreateUser();
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [role, setRole] = useState<Role>("vessel_user");
  const [err, setErr] = useState<string | null>(null);

  const valid = name.trim() && email.trim();

  const submit = async () => {
    setErr(null);
    const body: CreateUserBody = { name: name.trim(), email: email.trim(), role };
    try {
      await create.mutateAsync(body);
      onClose();
    } catch (e) {
      setErr((e as Error).message);
    }
  };

  return (
    <div className="overlay" onClick={onClose}>
      <div className="dialog" style={{ width: 480 }} onClick={(e) => e.stopPropagation()}>
        <h3>Add user</h3>
        <div className="field">
          <label>Name</label>
          <input className="input" value={name} onChange={(e) => setName(e.target.value)} placeholder="e.g. Bosun" />
        </div>
        <div className="field">
          <label>Email</label>
          <input className="input" type="email" value={email} onChange={(e) => setEmail(e.target.value)} placeholder="name@vessel" />
        </div>
        <div className="field">
          <label>Role</label>
          <select className="input" value={role} onChange={(e) => setRole(e.target.value as Role)}>
            {ROLES.map((r) => <option key={r} value={r}>{r}</option>)}
          </select>
        </div>
        {err && <div className="err">{err}</div>}
        <div className="dialog-actions">
          <button className="btn ghost" onClick={onClose}>Cancel</button>
          <button className="btn" onClick={submit} disabled={!valid || create.isPending}>
            {create.isPending ? "Adding…" : "Add user"}
          </button>
        </div>
      </div>
    </div>
  );
}
