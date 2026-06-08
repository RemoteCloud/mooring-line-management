import { useState } from "react";
import { NavLink, Outlet, useNavigate } from "react-router-dom";
import { visibleNav } from "./nav";
import { SCOPE } from "../config";
import { VesselSwitcher } from "./VesselSwitcher";
import { ConnBadge } from "./ConnBadge";
import { VesselProvider } from "./VesselContext";

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
    </header>
  );
}

export function AppShell() {
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
            {visibleNav().map((n) => (
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
            {/* /docs is a server-rendered page (proxied to the API), not an SPA
                route — a plain anchor does a real navigation past the router. */}
            <a className="nav-link" href="/docs" target="_blank" rel="noopener">
              <span className="nav-ico">⟨⟩</span>
              <span className="label">API docs</span>
            </a>
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
