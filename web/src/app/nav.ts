// Navigation model. Catalogue (master data) is reachable from both scopes — per
// product decision, onboard crew can add vendors/models on deck too.
import { isShore } from "../config";

export type NavItem = {
  to: string;
  label: string;
  icon: string;
  shoreOnly?: boolean;
  adminOnly?: boolean;
};

export const NAV: NavItem[] = [
  { to: "/", label: "Dashboard", icon: "▦" },
  { to: "/deck", label: "Deck map", icon: "⚓" },
  { to: "/register", label: "Rope register", icon: "≣" },
  { to: "/inspections", label: "Inspections", icon: "✓" },
  { to: "/logbook", label: "Log book", icon: "❏" },
  { to: "/files", label: "Files & certs", icon: "📄" },
  { to: "/catalogue", label: "Catalogue", icon: "⚙" },
  { to: "/permissions", label: "Permissions", icon: "⚙︎", adminOnly: true },
];

// Filters by scope only. Admin-gating (adminOnly) is applied where the nav is
// rendered (AppShell), which has access to auth context.
export const visibleNav = () => NAV.filter((n) => !n.shoreOnly || isShore());
