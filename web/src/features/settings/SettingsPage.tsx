// Settings → Access control (admin only). Lists the IdP groups the backend knows
// about and lets an admin set each group's access level (Denied / View / Edit)
// plus an optional friendly label.
//
// The access-control endpoints are not in the generated OpenAPI schema, so — as
// with the auth/session endpoint — we use plain fetch with credentials so the
// session cookie rides along. Data fetching/mutations go through React Query
// (wired app-wide), matching the rest of the app. Admin is enforced server-side
// (403 otherwise); we also gate the page client-side for a clean UX.
import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useIsAdmin, type AccessLevel } from "../../app/auth/authContext";
import { API_BASE } from "../../config";
import "./settings.css";

type GroupGrant = {
  groupId: string;
  label: string;
  level: AccessLevel;
  userCount: number;
};

type GroupsResponse = { groups: GroupGrant[] };

const GROUPS_KEY = ["access", "groups"];

// Thrown so callers can distinguish a 403 (not admin) from other load failures.
class ForbiddenError extends Error {
  constructor() {
    super("forbidden");
    this.name = "ForbiddenError";
  }
}

async function fetchGroups(): Promise<GroupGrant[]> {
  const res = await fetch(`${API_BASE}/access/groups`, {
    credentials: "include",
    headers: { Accept: "application/json" },
  });
  if (res.status === 403) {
    throw new ForbiddenError();
  }
  if (!res.ok) {
    throw new Error(`Failed to load groups (${res.status})`);
  }
  const data = (await res.json()) as GroupsResponse;
  return data.groups ?? [];
}

async function putGrant(
  groupId: string,
  level: "view" | "edit",
  label?: string,
): Promise<void> {
  const res = await fetch(
    `${API_BASE}/access/grants/${encodeURIComponent(groupId)}`,
    {
      method: "PUT",
      credentials: "include",
      headers: { "Content-Type": "application/json", Accept: "application/json" },
      body: JSON.stringify({ level, ...(label ? { label } : {}) }),
    },
  );
  if (!res.ok) {
    throw new Error(`Save failed (${res.status})`);
  }
}

async function deleteGrant(groupId: string): Promise<void> {
  const res = await fetch(
    `${API_BASE}/access/grants/${encodeURIComponent(groupId)}`,
    { method: "DELETE", credentials: "include" },
  );
  if (!res.ok && res.status !== 204) {
    throw new Error(`Reset failed (${res.status})`);
  }
}

export function SettingsPage() {
  const isAdmin = useIsAdmin();

  if (!isAdmin) {
    return (
      <>
        <h1 className="page-title">Settings</h1>
        <p className="page-sub">Access control</p>
        <div className="card">
          <p className="muted" style={{ margin: 0 }}>
            You’re not authorized to view this page. Administrator access is
            required.
          </p>
        </div>
      </>
    );
  }

  return <AccessControl />;
}

function AccessControl() {
  const qc = useQueryClient();
  const { data: groups = [], isLoading, error, refetch } = useQuery({
    queryKey: GROUPS_KEY,
    queryFn: fetchGroups,
    retry: false,
  });

  const forbidden = error instanceof ForbiddenError;
  const loadError =
    error && !forbidden
      ? error instanceof Error
        ? error.message
        : "Failed to load"
      : null;

  if (forbidden) {
    return (
      <>
        <Header />
        <div className="card">
          <p className="muted" style={{ margin: 0 }}>
            You’re not authorized to manage access control (403). Administrator
            access is required.
          </p>
        </div>
      </>
    );
  }

  return (
    <>
      <Header />

      <div className="card">
        <div className="table-wrap">
          <table className="grid access-grid">
            <thead>
              <tr>
                <th>Group</th>
                <th>Users</th>
                <th>Access level</th>
                <th>Label</th>
                <th aria-label="Status" />
              </tr>
            </thead>
            <tbody>
              {groups.map((g) => (
                <GroupRow key={g.groupId} group={g} qc={qc} />
              ))}

              {isLoading && (
                <tr>
                  <td colSpan={5} className="muted access-empty">
                    Loading…
                  </td>
                </tr>
              )}
              {loadError && !isLoading && (
                <tr>
                  <td colSpan={5} className="err access-empty">
                    {loadError}{" "}
                    <button
                      type="button"
                      className="btn ghost access-retry"
                      onClick={() => void refetch()}
                    >
                      Retry
                    </button>
                  </td>
                </tr>
              )}
              {!isLoading && !loadError && groups.length === 0 && (
                <tr>
                  <td colSpan={5} className="muted access-empty">
                    No groups found yet. Groups appear here once users sign in.
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </div>
    </>
  );
}

function GroupRow({
  group,
  qc,
}: {
  group: GroupGrant;
  qc: ReturnType<typeof useQueryClient>;
}) {
  // Draft label text (so typing doesn't immediately PUT). React's recommended
  // "adjust state during render" pattern: when the server-side label changes
  // underneath us, reset the draft to match instead of using an effect.
  const serverLabel = group.label ?? "";
  const [labelDraft, setLabelDraft] = useState(serverLabel);
  const [syncedLabel, setSyncedLabel] = useState(serverLabel);
  if (serverLabel !== syncedLabel) {
    setSyncedLabel(serverLabel);
    setLabelDraft(serverLabel);
  }

  // Toggled true briefly after a successful save (a timer clears it) so the
  // "Saved ✓" badge shows without reading an impure clock during render.
  const [justSaved, setJustSaved] = useState(false);

  const mutation = useMutation({
    mutationFn: async (next: { level: AccessLevel; label: string }) => {
      if (next.level === "denied") {
        await deleteGrant(group.groupId);
      } else {
        await putGrant(group.groupId, next.level, next.label || undefined);
      }
    },
    // Optimistically update the cached row, snapshot for rollback.
    onMutate: async (next) => {
      await qc.cancelQueries({ queryKey: GROUPS_KEY });
      const prev = qc.getQueryData<GroupGrant[]>(GROUPS_KEY);
      qc.setQueryData<GroupGrant[]>(GROUPS_KEY, (gs) =>
        (gs ?? []).map((g) =>
          g.groupId === group.groupId
            ? { ...g, level: next.level, label: next.label }
            : g,
        ),
      );
      return { prev };
    },
    onError: (_e, _next, ctx) => {
      if (ctx?.prev) qc.setQueryData(GROUPS_KEY, ctx.prev);
    },
    onSuccess: () => {
      setJustSaved(true);
      window.setTimeout(() => setJustSaved(false), 1800);
    },
    // Refetch to stay authoritative (userCount etc. may change server-side).
    onSettled: () => {
      void qc.invalidateQueries({ queryKey: GROUPS_KEY });
    },
  });

  const saving = mutation.isPending;

  const onLevelChange = (next: AccessLevel) => {
    mutation.mutate({ level: next, label: labelDraft });
  };

  const onLabelSave = () => {
    if (labelDraft === (group.label ?? "")) return; // nothing changed
    if (group.level === "denied") return; // label only meaningful when granted
    mutation.mutate({ level: group.level, label: labelDraft });
  };

  return (
    <tr className="access-row">
      <td>
        {group.label ? (
          group.label
        ) : (
          <code className="access-gid">{group.groupId}</code>
        )}
      </td>
      <td className="muted">{group.userCount}</td>
      <td>
        <select
          className="input"
          value={group.level}
          disabled={saving}
          onChange={(e) => onLevelChange(e.target.value as AccessLevel)}
          aria-label={`Access level for ${group.label || group.groupId}`}
        >
          <option value="denied">Denied</option>
          <option value="view">View</option>
          <option value="edit">Edit</option>
        </select>
      </td>
      <td>
        <input
          className="input access-label-input"
          type="text"
          placeholder={
            group.level === "denied" ? "Grant access first" : "optional label"
          }
          value={labelDraft}
          disabled={group.level === "denied" || saving}
          onChange={(e) => setLabelDraft(e.target.value)}
          onBlur={onLabelSave}
          onKeyDown={(e) => {
            if (e.key === "Enter") {
              (e.target as HTMLInputElement).blur();
            }
          }}
          aria-label={`Label for ${group.groupId}`}
        />
      </td>
      <td className="access-status">
        {saving && <span className="muted">Saving…</span>}
        {!saving && justSaved && <span className="access-saved">Saved ✓</span>}
        {mutation.isError && (
          <span className="err" title={String(mutation.error)}>
            {mutation.error instanceof Error
              ? mutation.error.message
              : "Error"}
          </span>
        )}
      </td>
    </tr>
  );
}

function Header() {
  return (
    <>
      <h1 className="page-title">Settings</h1>
      <p className="page-sub">Access control</p>
      <p className="access-intro">
        Admins always have full access. Every other user gets the highest level
        granted to any of their groups; groups without a grant are denied.
      </p>
    </>
  );
}
