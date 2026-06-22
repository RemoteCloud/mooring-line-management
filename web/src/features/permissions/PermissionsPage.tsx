// Permissions → Access control (admin only). Lists the IdP groups the backend
// knows about and lets an admin set each group's access level (Denied / View /
// Edit). Group names come live from the IdP; the GUID is shown as a fallback.
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
import "./permissions.css";

type GroupGrant = {
  groupId: string;
  name: string;
  level: AccessLevel;
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

async function putGrant(groupId: string, level: "view" | "edit"): Promise<void> {
  const res = await fetch(
    `${API_BASE}/access/grants/${encodeURIComponent(groupId)}`,
    {
      method: "PUT",
      credentials: "include",
      headers: { "Content-Type": "application/json", Accept: "application/json" },
      body: JSON.stringify({ level }),
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

export function PermissionsPage() {
  const isAdmin = useIsAdmin();

  if (!isAdmin) {
    return (
      <>
        <h1 className="page-title">Permissions</h1>
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
        <Header onReload={() => void refetch()} />
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
      <Header onReload={() => void refetch()} />

      <div className="card">
        <div className="table-wrap">
          <table className="grid access-grid">
            <thead>
              <tr>
                <th>Group</th>
                <th>Access level</th>
                <th aria-label="Status" />
              </tr>
            </thead>
            <tbody>
              {groups.map((g) => (
                <GroupRow key={g.groupId} group={g} qc={qc} />
              ))}

              {isLoading && (
                <tr>
                  <td colSpan={3} className="muted access-empty">
                    Loading…
                  </td>
                </tr>
              )}
              {loadError && !isLoading && (
                <tr>
                  <td colSpan={3} className="err access-empty">
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
                  <td colSpan={3} className="muted access-empty">
                    No groups loaded. Use “Reload groups” to fetch them from
                    UserManagement.
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
  // Toggled true briefly after a successful save (a timer clears it) so the
  // "Saved ✓" badge shows without reading an impure clock during render.
  const [justSaved, setJustSaved] = useState(false);

  const mutation = useMutation({
    mutationFn: async (next: { level: AccessLevel }) => {
      if (next.level === "denied") {
        await deleteGrant(group.groupId);
      } else {
        await putGrant(group.groupId, next.level);
      }
    },
    // Optimistically update the cached row, snapshot for rollback.
    onMutate: async (next) => {
      await qc.cancelQueries({ queryKey: GROUPS_KEY });
      const prev = qc.getQueryData<GroupGrant[]>(GROUPS_KEY);
      qc.setQueryData<GroupGrant[]>(GROUPS_KEY, (gs) =>
        (gs ?? []).map((g) =>
          g.groupId === group.groupId ? { ...g, level: next.level } : g,
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
    // Refetch to stay authoritative (levels may change server-side).
    onSettled: () => {
      void qc.invalidateQueries({ queryKey: GROUPS_KEY });
    },
  });

  const saving = mutation.isPending;

  const onLevelChange = (next: AccessLevel) => {
    mutation.mutate({ level: next });
  };

  return (
    <tr className="access-row">
      <td>
        {group.name ? (
          group.name
        ) : (
          <code className="access-gid">{group.groupId}</code>
        )}
      </td>
      <td>
        <select
          className="input"
          value={group.level}
          disabled={saving}
          onChange={(e) => onLevelChange(e.target.value as AccessLevel)}
          aria-label={`Access level for ${group.name || group.groupId}`}
        >
          <option value="denied">Denied</option>
          <option value="view">View</option>
          <option value="edit">Edit</option>
        </select>
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

function Header({ onReload }: { onReload: () => void }) {
  return (
    <>
      <h1 className="page-title">Permissions</h1>
      <p className="page-sub">Access control</p>
      <p className="access-intro">
        Admins always have full access. Every other user gets the highest level
        granted to any of their groups; groups without a grant are denied.
      </p>
      <p style={{ margin: "0 0 20px" }}>
        <button type="button" className="btn ghost" onClick={onReload}>
          Reload groups
        </button>
      </p>
    </>
  );
}
