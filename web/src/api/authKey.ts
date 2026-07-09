// Basic (temporary) API-key auth for the PWA. The key lives in localStorage and is
// attached to every same-origin /api request via a global fetch wrapper, so the typed
// openapi-fetch client and all the hand-rolled feature fetches authenticate uniformly.
import { API_BASE } from "../config";

const KEY = "mlm_api_key";

export const getApiKey = (): string => localStorage.getItem(KEY) ?? "";

export function setApiKey(k: string): void {
  localStorage.setItem(KEY, k.trim());
  window.dispatchEvent(new Event("mlm-auth-changed"));
}

export function clearApiKey(): void {
  localStorage.removeItem(KEY);
  window.dispatchEvent(new Event("mlm-auth-changed"));
}

// reportUnauthorized clears the stored key and notifies the unlock gate. Called by the
// fetch wrapper on any 401 so a revoked/invalid key drops the app back to the gate.
function reportUnauthorized(): void {
  if (getApiKey()) clearApiKey();
  else window.dispatchEvent(new Event("mlm-auth-changed"));
}

let installed = false;

// installAuthFetch monkey-patches window.fetch once. Same-origin /api requests get the
// Authorization header; everything else (e.g. presigned S3 URLs on another origin) is
// left untouched.
export function installAuthFetch(): void {
  if (installed) return;
  installed = true;
  const orig = window.fetch.bind(window);

  window.fetch = async (input: RequestInfo | URL, init?: RequestInit): Promise<Response> => {
    const raw =
      typeof input === "string" ? input : input instanceof Request ? input.url : input.toString();
    let isApi = false;
    try {
      const u = new URL(raw, location.origin);
      isApi = u.origin === location.origin && u.pathname.startsWith(API_BASE);
    } catch {
      isApi = false;
    }

    let nextInput = input;
    let nextInit = init;
    const key = getApiKey();
    if (isApi && key) {
      if (input instanceof Request) {
        const headers = new Headers(input.headers);
        if (!headers.has("Authorization")) headers.set("Authorization", `Bearer ${key}`);
        nextInput = new Request(input, { headers });
      } else {
        const headers = new Headers(init?.headers);
        if (!headers.has("Authorization")) headers.set("Authorization", `Bearer ${key}`);
        nextInit = { ...init, headers };
      }
    }

    const res = await orig(nextInput as RequestInfo, nextInit);
    if (isApi && res.status === 401) reportUnauthorized();
    return res;
  };
}
