import { useEffect, useRef, useState } from "react";
import { NavLink, Outlet, useNavigate } from "react-router-dom";
import { visibleNav } from "./nav";
import { SCOPE } from "../config";
import { VesselSwitcher } from "./VesselSwitcher";
import { ConnBadge } from "./ConnBadge";
import { VesselProvider } from "./VesselContext";

// Collapsed to an icon by default so the nav tabs keep their room; expands to a
// full input on click and collapses again when empty/blurred or on Escape.
function GlobalSearch() {
  const navigate = useNavigate();
  const [q, setQ] = useState("");
  const [open, setOpen] = useState(false);

  if (!open) {
    return (
      <button type="button" className="icon-btn" aria-label="Search" title="Search" onClick={() => setOpen(true)}>
        🔍
      </button>
    );
  }
  return (
    <form
      className="search"
      onSubmit={(e) => {
        e.preventDefault();
        if (q.trim()) navigate("/register?q=" + encodeURIComponent(q));
        setOpen(false);
      }}
    >
      <span>🔍</span>
      <input
        autoFocus
        placeholder="Search ID, serial, location…"
        aria-label="Global search"
        value={q}
        onChange={(e) => setQ(e.target.value)}
        onKeyDown={(e) => e.key === "Escape" && setOpen(false)}
        onBlur={() => !q && setOpen(false)}
      />
    </form>
  );
}

// GridMenu is the app-grid launcher: a small popover holding the server-rendered
// doc UIs (/docs, /swagger). Those are real navigations (not SPA routes), so they
// stay plain anchors that open in a new tab.
function GridMenu() {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (!open) return;
    const onDoc = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    const onEsc = (e: KeyboardEvent) => e.key === "Escape" && setOpen(false);
    document.addEventListener("mousedown", onDoc);
    document.addEventListener("keydown", onEsc);
    return () => {
      document.removeEventListener("mousedown", onDoc);
      document.removeEventListener("keydown", onEsc);
    };
  }, [open]);

  return (
    <div className="grid-menu" ref={ref}>
      <button
        type="button"
        className="icon-btn"
        aria-label="Apps & API"
        aria-expanded={open}
        title="Apps & API reference"
        onClick={() => setOpen((o) => !o)}
      >
        ▦
      </button>
      {open && (
        <div className="grid-menu-panel" role="menu">
          <div className="grid-menu-head">API reference</div>
          <a className="grid-menu-item" href="/docs" target="_blank" rel="noopener" onClick={() => setOpen(false)}>
            <span className="nav-ico">⟨⟩</span> API docs
          </a>
          <a className="grid-menu-item" href="/swagger" target="_blank" rel="noopener" onClick={() => setOpen(false)}>
            <span className="nav-ico">⬚</span> Swagger
          </a>
        </div>
      )}
    </div>
  );
}

export function AppShell() {
  return (
    <VesselProvider>
      <div className="appframe">
        <header className="appbar">
          <div className="appbar-brand">
            <div className="brand-mark">M</div>
            <span className="brand-parent">MARANICS</span>
            <span className="brand-div">|</span>
            <span className="brand-name">Mooring</span>
          </div>

          <nav className="appbar-nav">
            {visibleNav().map((n) => (
              <NavLink
                key={n.to}
                to={n.to}
                end={n.to === "/"}
                title={n.label}
                className={({ isActive }) => "topnav-link" + (isActive ? " active" : "")}
              >
                <span className="nav-ico">{n.icon}</span>
                <span className="label">{n.label}</span>
              </NavLink>
            ))}
          </nav>

          <div className="appbar-right">
            <GlobalSearch />
            <ConnBadge />
            <span className={"scope-badge " + SCOPE}>{SCOPE.toUpperCase()}</span>
            <VesselSwitcher />
            <GridMenu />
            <button type="button" className="icon-btn" aria-label="Notifications" title="Notifications (none)">
              🔔
            </button>
            <span className="avatar" title="Crew">ML</span>
          </div>
        </header>

        <main className="content">
          <Outlet />
        </main>
      </div>
    </VesselProvider>
  );
}
