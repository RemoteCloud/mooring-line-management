// Deploy-time configuration. The same build runs onboard (single vessel) or shore
// (fleet); VITE_SCOPE selects which. Onboard hides the vessel switcher and fleet views.

export type Scope = "onboard" | "shore";

export const SCOPE: Scope =
  (import.meta.env.VITE_SCOPE as Scope) === "onboard" ? "onboard" : "shore";

// Onboard deployments are pinned to one vessel.
export const ONBOARD_VESSEL_ID: string | undefined =
  import.meta.env.VITE_VESSEL_ID || undefined;

export const isOnboard = () => SCOPE === "onboard";
export const isShore = () => SCOPE === "shore";

// All API calls go through the dev proxy / same-origin /api prefix.
export const API_BASE = "/api";
