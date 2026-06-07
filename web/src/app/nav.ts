// Navigation model. Catalogue is shore-only (master data authored on shore, §4.0).
import { isShore } from "../config";

export type NavItem = { to: string; label: string; icon: string; shoreOnly?: boolean };

export const NAV: NavItem[] = [
  { to: "/", label: "Dashboard", icon: "▦" },
  { to: "/deck", label: "Deck map", icon: "⚓" },
  { to: "/register", label: "Rope register", icon: "≣" },
  { to: "/inspections", label: "Inspections", icon: "✓" },
  { to: "/logbook", label: "Log book", icon: "❏" },
  { to: "/files", label: "Files & certs", icon: "📄" },
  { to: "/catalogue", label: "Catalogue", icon: "⚙", shoreOnly: true },
];

export const visibleNav = () => NAV.filter((n) => !n.shoreOnly || isShore());
