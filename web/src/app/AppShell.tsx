import { useState } from "react";
import { NavLink, Outlet, useNavigate } from "react-router-dom";
import { visibleNav } from "./nav";
import { SCOPE } from "../config";
import { VesselSwitcher } from "./VesselSwitcher";
import { ConnBadge } from "./ConnBadge";
import { VesselProvider } from "./VesselContext";
import { useAuth, useIsAdmin } from "./auth/authContext";

function UserMenu() {
  const { user, permissions, logout } = useAuth();
  if (!user) return null;
  return (
    <div className="user-menu">
      {!permissions.canWrite && (
        <span className="ro-badge" title="You have read-only access">
          Read-only
        </span>
      )}
      <div className="user-id">
        <span className="user-email" title={user.name || user.email}>
          {user.email || user.name}
        </span>
        {(user.positionName || user.positionId) && (
          <span className="user-position">
            {user.positionName || user.positionId}
          </span>
        )}
      </div>
      <button className="btn ghost" onClick={() => void logout()}>
        Logout
      </button>
    </div>
  );
}

function Topbar() {
  const navigate = useNavigate();
  const [q, setQ] = useState("");
  return (
    <header className="topbar">
      <form
        className="search"
        onSubmit={(e) => {
          e.preventDefault();
          navigate("/register?q=" + encodeURIComponent(q));
        }}
      >
        <span>🔍</span>
        <input
          placeholder="Search ID, serial, location…"
          aria-label="Global search"
          value={q}
          onChange={(e) => setQ(e.target.value)}
        />
      </form>
      <div className="spacer" />
      <ConnBadge />
      <span className={"scope-badge " + SCOPE}>{SCOPE.toUpperCase()}</span>
      <VesselSwitcher />
      <UserMenu />
    </header>
  );
}

export function AppShell() {
  const isAdmin = useIsAdmin();
  // Admin-only items (Permissions) are hidden from non-admins; scope filtering is
  // already handled by visibleNav().
  const navItems = visibleNav().filter((n) => !n.adminOnly || isAdmin);
  return (
    <VesselProvider>
      <div className="shell">
        <aside className="sidebar">
          <div className="brand">
            <div className="brand-mark">M</div>
            <div>
              <div className="brand-name">Mooring</div>
              <div className="brand-sub">Line management</div>
            </div>
          </div>
          <nav>
            {navItems.map((n) => (
              <NavLink
                key={n.to}
                to={n.to}
                end={n.to === "/"}
                className={({ isActive }) => "nav-link" + (isActive ? " active" : "")}
              >
                <span className="nav-ico">{n.icon}</span>
                <span className="label">{n.label}</span>
              </NavLink>
            ))}
          </nav>
        </aside>

        <div className="main">
          <Topbar />
          <main className="content">
            <Outlet />
          </main>
        </div>
      </div>
    </VesselProvider>
  );
}
